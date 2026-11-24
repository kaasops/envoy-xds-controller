import { useState } from 'react'
import { Alert, Box, Button, ButtonBase, Link, Paper, Stack, Typography } from '@mui/material'
import ErrorOutline from '@mui/icons-material/ErrorOutline'
import EventOutlined from '@mui/icons-material/EventOutlined'
import ChevronRight from '@mui/icons-material/ChevronRight'
import { Link as RouterLink } from 'react-router-dom'
import type { NodeOverviewResponse } from '../../../common/types/overviewApiTypes'
import { nodeOverviewPath, nodePath, typeLabel, type SnapshotState } from '../api'
import { StatusBadges, StreamClientLink, StreamDetails } from '../console/sections/clients'
import {
	noConfiguration,
	nodesWithoutSnapshot,
	streamConfiguration,
	streamClientLabel,
	streamClientSearchValues,
	streamId,
	streamRowId,
	streamStatuses,
	type ClientsResponse
} from '../console/streams'
import ConfigurationTable from './ConfigurationTable'
import { certificateId } from './configurationData'

function utc(value?: string) {
	if (!value) return ''
	const date = new Date(value)
	return Number.isNaN(date.getTime()) ? '' : date.toISOString().slice(0, 19).replace('T', ' ') + ' UTC'
}

function firstNames(names?: string[] | null) {
	if (!names?.length) return ''
	return names.length > 2 ? names.slice(0, 2).join(', ') + ' +' + (names.length - 2) + ' more' : names.join(', ')
}

interface AlertEntry {
	id: string
	title: string
	description: string
	category: 'stream' | 'certificate' | 'configuration'
	href: string
}

interface AlertProps {
	nodeId: string
	overview?: NodeOverviewResponse
	clients?: ClientsResponse
	freeze?: SnapshotState['freeze']
}

export function NodeAlerts({ nodeId, overview, clients, freeze }: AlertProps) {
	const [expanded, setExpanded] = useState(false)
	const alerts: AlertEntry[] = []
	const pod = clients?.controllerPod ?? ''
	if (nodesWithoutSnapshot(clients).has(nodeId)) {
		alerts.push({
			id: 'missing-snapshot',
			title: noConfiguration,
			description: pod
				? 'Connected to ' + pod + ' without a snapshot for this node ID.'
				: 'No snapshot for this connected node ID.',
			category: 'configuration',
			href: nodeOverviewPath(nodeId, 'streams')
		})
	}
	for (const stream of clients?.clients ?? []) {
		if (stream.nodeId !== nodeId) continue
		const key = streamRowId(pod, stream)
		for (const type of stream.types) {
			for (const rejection of type.rejections) {
				const resources = firstNames(rejection.resources?.map(resource => resource.name))
				alerts.push({
					id: JSON.stringify([key, type.typeUrl, rejection.nonce]),
					title: streamClientLabel(stream) + ' · ' + typeLabel(type.typeUrl) + ' NACK on ' + streamId(stream),
					description: [
						resources,
						stream.envoyVersion && 'Envoy ' + stream.envoyVersion,
						utc(rejection.rejectedAt)
					]
						.filter(Boolean)
						.join(' · '),
					category: 'stream',
					href: nodeOverviewPath(nodeId, 'streams', key)
				})
			}
		}
	}

	const expiring = (overview?.certificates ?? [])
		.filter(cert => cert.status === 'warning' || cert.status === 'critical' || cert.status === 'expired')
		.sort((a, b) => a.daysUntilExpiry - b.daysUntilExpiry)
	for (const cert of expiring) {
		const id = certificateId(cert)
		let expiry = ' expires in ' + cert.daysUntilExpiry + (cert.daysUntilExpiry === 1 ? ' day' : ' days')
		if (cert.status === 'expired' || cert.daysUntilExpiry <= 0) {
			expiry = Date.parse(cert.notAfter) > Date.now() ? ' expires within 24 hours' : ' has expired'
		}
		alerts.push({
			id: 'certificate/' + id,
			title: id + expiry,
			description: [firstNames(cert.usedByDomains), utc(cert.notAfter)].filter(Boolean).join(' · '),
			category: 'certificate',
			href: nodeOverviewPath(nodeId, 'certificates', id)
		})
	}
	const visible = expanded ? alerts : alerts.slice(0, 4)
	return (
		<Stack gap={2}>
			{freeze && (
				<Alert severity='info'>
					<Typography variant='subtitle2'>Snapshot publication is frozen</Typography>
					{[freeze.reason, freeze.by, utc(freeze.at)].filter(Boolean).join(' · ')}
					{' · '}
					<Link component={RouterLink} to={nodePath(nodeId)}>
						View snapshots
					</Link>
				</Alert>
			)}
			<Paper component='section' variant='outlined' aria-label='Needs attention' sx={{ overflow: 'hidden' }}>
				<Stack direction='row' alignItems='center' justifyContent='space-between' sx={{ px: 2, py: 1.25 }}>
					<Typography component='h2' variant='subtitle1' fontWeight={600}>
						Needs attention
					</Typography>
					{overview && clients && (
						<Typography variant='caption' color='text.secondary'>
							{alerts.length}
						</Typography>
					)}
				</Stack>
				{visible.map(alert => {
					const action = {
						stream: 'Inspect stream',
						certificate: 'View certificate',
						configuration: 'View streams'
					}[alert.category]
					const Icon = alert.category === 'certificate' ? EventOutlined : ErrorOutline
					return (
						<ButtonBase
							key={alert.id}
							component={RouterLink}
							to={alert.href}
							aria-label={action + ': ' + alert.title}
							sx={{
								width: '100%',
								textAlign: 'left',
								justifyContent: 'flex-start',
								gap: 1.5,
								px: 2,
								py: 1,
								borderTop: 1,
								borderColor: 'divider',
								'&:hover': { bgcolor: 'action.hover' },
								'&.Mui-focusVisible': {
									outline: '2px solid',
									outlineColor: 'primary.main',
									outlineOffset: -2
								}
							}}
						>
							<Icon color={alert.category === 'stream' ? 'error' : 'warning'} fontSize='small' />
							<Box component='span' sx={{ flex: 1, minWidth: 0, overflowWrap: 'anywhere' }}>
								<Typography component='span' display='block' variant='subtitle2'>
									{alert.title}
								</Typography>
								<Typography component='span' variant='caption' color='text.secondary'>
									{alert.description}
								</Typography>
							</Box>
							<Typography component='span' variant='caption' color='primary'>
								{action}
							</Typography>
							<ChevronRight fontSize='small' color='action' />
						</ButtonBase>
					)
				})}
				{!alerts.length && (
					<Typography
						sx={{ px: 2, py: 1.25, borderTop: 1, borderColor: 'divider' }}
						variant='body2'
						color='text.secondary'
					>
						{overview && clients
							? 'No open stream rejections or certificate expiry warnings in these responses.'
							: 'No alerts from the available data.'}
					</Typography>
				)}
				{alerts.length > 4 && (
					<Button size='small' sx={{ m: 1 }} onClick={() => setExpanded(!expanded)}>
						{expanded ? 'Show fewer' : 'Show all ' + alerts.length}
					</Button>
				)}
			</Paper>
		</Stack>
	)
}

interface StreamProps {
	clients: ClientsResponse
	nodeId: string
	selected: string
}

export function NodeStreams({ clients, nodeId, selected }: StreamProps) {
	const streams = clients.clients.filter(stream => stream.nodeId === nodeId)
	const pod = clients.controllerPod
	const selection = streams.find(stream => streamRowId(pod, stream) === selected)
	return (
		<Box>
			<Box sx={{ p: 2 }}>
				<Typography component='h2' variant='subtitle1' fontWeight={600}>
					xDS streams
				</Typography>
				<Typography variant='body2' color='text.secondary'>
					{pod ? 'Connections observed on ' + pod : 'Connections reported by this controller'}
				</Typography>
			</Box>
			{streams.length ? (
				<ConfigurationTable
					label='xDS streams'
					rows={streams}
					rowId={stream => streamRowId(pod, stream)}
					selected={selected}
					columns={[
						{
							label: 'Envoy client',
							value: stream => streamClientSearchValues(stream).join(' '),
							render: stream => (
								<StreamClientLink
									stream={stream}
									to={nodeOverviewPath(nodeId, 'streams', streamRowId(pod, stream))}
								/>
							)
						},
						{ label: 'Stream', value: streamId },
						{
							label: 'Configuration',
							value: stream => streamConfiguration(stream, clients.snapshotVersions)
						},
						{ label: 'Envoy version', value: stream => stream.envoyVersion || 'Unknown' },
						{
							label: 'Response',
							value: stream => streamStatuses(stream).join(', '),
							render: stream => <StatusBadges statuses={streamStatuses(stream)} />
						}
					]}
				/>
			) : (
				<Typography variant='body2' sx={{ px: 2, pb: 2 }}>
					No streams visible on this controller.
				</Typography>
			)}
			{selection && (
				<Box
					data-node-detail
					tabIndex={-1}
					sx={{ p: 2, borderTop: 1, borderColor: 'divider', overflowWrap: 'anywhere' }}
				>
					<Typography component='h3' variant='subtitle1' fontWeight={600} mb={1}>
						{streamClientLabel(selection)} · Responses by type
					</Typography>
					<StreamDetails
						stream={selection}
						pod={pod}
						versions={clients.snapshotVersions?.[nodeId]}
						resourcePath={(typeKey, name) => nodeOverviewPath(nodeId, 'hashes', typeKey + '/' + name)}
					/>
				</Box>
			)}
			{selected && !selection && (
				<Alert severity='info'>The selected stream is no longer present on this controller.</Alert>
			)}
		</Box>
	)
}
