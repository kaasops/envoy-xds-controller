import { ConsoleTime } from '../../time/ConsoleTime'
import { useConsoleTime } from '../../time/useConsoleTime'
import { Alert, Box, Link, Stack, Typography } from '@mui/material'
import { Code, ConnectError } from '@connectrpc/connect'
import { useQuery } from '@tanstack/react-query'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'
import client from '../../../../api/axiosApiClient'
import {
	apiError,
	fleetQueryKey,
	httpStatus,
	manualQueryOptions,
	nodePath,
	typeLabel,
	type SnapshotState
} from '../../api'
import StatusHelp from '../StatusHelp'
import SectionTable, { type Column } from '../SectionTable'
import { appliesToAllNodes, loadVirtualServices, virtualServicesKey } from '../resourceData'
import {
	clientsQueryKey,
	getClients,
	streamClientLabel,
	streamClientSearchValues,
	streamId,
	streamRowId,
	type ClientStream
} from '../streams'

interface Observation {
	id: string
	title: string
	source: string
	client?: ClientStream
	nodes: string[]
	subject: string
	at?: string
	message: string
	href: string
	linkLabel: string
}

function readable(error: unknown) {
	const status = httpStatus(error)
	return (
		status !== 401 &&
		status !== 403 &&
		!(
			error instanceof ConnectError &&
			(error.code === Code.PermissionDenied || error.code === Code.Unauthenticated)
		)
	)
}

export default function Events() {
	const { formatTime } = useConsoleTime()
	const columns: Column<Observation>[] = [
		{
			key: 'event',
			label: 'Event',
			value: row => row.title,
			render: row =>
				row.source === 'xDS stream' ? (
					<StatusHelp status='NACK'>{row.title}</StatusHelp>
				) : row.source === 'Snapshot' ? (
					<StatusHelp status='Paused'>{row.title}</StatusHelp>
				) : (
					row.title
				)
		},
		{
			key: 'at',
			label: 'Time',
			value: row => formatTime(row.at) || 'Not reported',
			sortValue: row => Date.parse(row.at ?? '') || 0,
			render: row => <ConsoleTime value={row.at} fallback='Not reported' />
		},
		{
			key: 'node',
			identifier: true,
			label: 'Node',
			value: row => row.nodes,
			appliesToAllNodes: row => row.source === 'Virtual service' && appliesToAllNodes(row.nodes)
		},
		{
			key: 'source',
			label: 'Source',
			value: row => (row.client ? [row.source, ...streamClientSearchValues(row.client)] : row.source),
			render: row => (row.client ? streamClientLabel(row.client) : row.source)
		},
		{ key: 'resource', label: 'Resource', value: row => row.subject }
	]
	const [params] = useSearchParams()
	const node = params.get('node') || undefined
	const clients = useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: clientsQueryKey(node),
		queryFn: ({ signal }) => getClients(signal, node)
	})
	const pauses = useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: [...fleetQueryKey, 'pause-records'],
		queryFn: async ({ signal }) => {
			const { data } = await client.get<{
				pausedNodes: Record<string, NonNullable<SnapshotState['pause']>> | null
			}>('/pausedNodes', { signal })
			return data.pausedNodes || {}
		}
	})
	const services = useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: virtualServicesKey,
		queryFn: ({ signal }) => loadVirtualServices(signal)
	})
	const observations: Observation[] = []
	if (clients.data && readable(clients.error)) {
		const { controllerPod } = clients.data
		for (const stream of clients.data.clients) {
			for (const type of stream.types) {
				for (const rejection of type.rejections) {
					const selected = streamRowId(controllerPod, stream)
					const filter = new URLSearchParams({ selected })
					if (stream.nodeId) filter.set('node', stream.nodeId)
					observations.push({
						id: JSON.stringify(['rejection', selected, type.typeUrl, rejection.nonce]),
						title: 'xDS NACK',
						source: 'xDS stream',
						client: stream,
						nodes: stream.nodeId ? [stream.nodeId] : [],
						subject:
							rejection.resources?.map(resource => resource.name).join(', ') || typeLabel(type.typeUrl),
						at: rejection.rejectedAt,
						message: rejection.error,
						linkLabel: streamClientLabel(stream) + ' · Open ' + streamId(stream),
						href: '/beta/clients?' + filter
					})
				}
			}
		}
	}
	if (readable(pauses.error)) {
		for (const [nodeId, pause] of Object.entries(pauses.data || {})) {
			observations.push({
				id: JSON.stringify(['pause', nodeId]),
				title: 'Updates paused',
				source: 'Snapshot',
				nodes: [nodeId],
				subject: nodeId,
				at: pause.at,
				message: pause.reason,
				href: nodePath(nodeId),
				linkLabel: 'Open node snapshots'
			})
		}
	}
	if (readable(services.error)) {
		for (const service of services.data || []) {
			if (service.status?.invalid)
				observations.push({
					id: JSON.stringify(['service', service.uid]),
					title: 'Virtual service rejected',
					source: 'Virtual service',
					nodes: service.nodeIds,
					subject: service.name,
					message: service.status.message,
					href: '/beta/res-virtual-services?' + new URLSearchParams({ selected: service.uid }),
					linkLabel: 'Open virtual service'
				})
		}
	}
	observations.sort((a, b) => (b.at ? Date.parse(b.at) : 0) - (a.at ? Date.parse(a.at) : 0))
	const sources = [
		{ label: 'xDS streams', query: clients },
		{ label: 'Node pauses', query: pauses },
		{ label: 'Virtual services', query: services }
	]
	const failed = sources.filter(source => source.query.isError)
	const times = sources
		.filter(source => source.query.data && readable(source.query.error))
		.map(source => source.query.dataUpdatedAt)
	return (
		<Stack gap={1}>
			{failed.map(({ label, query }) => (
				<Alert key={label} severity='warning'>
					{label}{' '}
					{query.data && readable(query.error)
						? 'refresh failed. Showing the last response.'
						: 'unavailable.'}{' '}
					{apiError(query.error)}
				</Alert>
			))}
			<SectionTable
				title='Events'
				tableHeading='Recent observations'
				rows={observations}
				rowId={row => row.id}
				columns={columns}
				description={[
					'Current rejections, paused updates and rejected virtual services.',
					clients.data?.controllerPod && readable(clients.error)
						? 'Streams on ' + clients.data.controllerPod
						: ''
				]
					.filter(Boolean)
					.join(' ')}
				loading={sources.some(source => source.query.isFetching)}
				updatedAt={times.length ? Math.min(...times) : undefined}
				onRefresh={() => {
					for (const source of sources) void source.query.refetch()
				}}
				emptyText={failed.length ? 'No events from the available sources.' : 'No current events.'}
				detail={row => (
					<Stack gap={1.5}>
						<Typography variant='subtitle2'>{row.subject}</Typography>
						<Box sx={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{row.message}</Box>
						<Link component={RouterLink} to={row.href}>
							{row.linkLabel}
						</Link>
					</Stack>
				)}
			/>
		</Stack>
	)
}
