import ArrowForwardOutlined from '@mui/icons-material/ArrowForwardOutlined'
import { Box, Chip, Link, Paper, Stack, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { httpStatus, nodeOverviewPath, nodePath } from '../../api'
import { nodeSection, useNodeInventory } from '../nodeInventory'
import StatusHelp, { type HelpStatus } from '../StatusHelp'
import SectionTable, { type Column } from '../SectionTable'
import { noConfiguration, nodesWithoutSnapshot, streamCounts } from '../streams'

interface Observation {
	node: string
	state: HelpStatus
	detail: string
	href: string
}

const columns: Column<Observation>[] = [
	{
		key: 'node',
		label: 'Node ID',
		identifier: true,
		value: row => row.node,
		render: row => (
			<Link component={RouterLink} to={nodeOverviewPath(row.node)}>
				{row.node}
			</Link>
		)
	},
	{
		key: 'status',
		label: 'Observation',
		value: row => row.state,
		render: row => {
			const badge = (
				<Chip
					data-status
					size='small'
					variant='outlined'
					color={row.state === 'NACK' ? 'error' : row.state === 'Awaiting ACK' ? 'warning' : 'default'}
					label={row.state}
					sx={{
						maxWidth: '100%',
						height: 'auto',
						minHeight: 24,
						'& .MuiChip-label': { whiteSpace: 'normal' }
					}}
				/>
			)
			return (
				<StatusHelp status={row.state} to={row.state === 'Paused' ? row.href : undefined}>
					{badge}
				</StatusHelp>
			)
		}
	},
	{
		key: 'details',
		label: 'Details',
		value: row => row.detail,
		render: row => (
			<Link component={RouterLink} to={row.href}>
				{row.detail}
			</Link>
		)
	}
]

export default function Overview() {
	const query = useNodeInventory()
	const data = httpStatus(query.error) === 403 || httpStatus(query.error) === 401 ? undefined : query.data
	const observations: Observation[] = []
	const missing = nodesWithoutSnapshot(data?.clients)
	for (const node of data?.nodes ?? []) {
		if (missing.has(node.nodeId)) {
			observations.push({
				node: node.nodeId,
				state: noConfiguration,
				detail: 'View connected xDS streams',
				href: nodeSection('clients', node.nodeId)
			})
		}
		for (const state of ['NACK', 'Awaiting ACK', 'Subscribed', 'Unknown'] as const) {
			const count = node.responses[state]
			if (count && !((state === 'Unknown' || state === 'Subscribed') && missing.has(node.nodeId)))
				observations.push({
					node: node.nodeId,
					state,
					detail: count + (count === 1 ? ' stream' : ' streams'),
					href: nodeSection('clients', node.nodeId, state)
				})
		}
		if (node.responses.ACK && !data?.clients.snapshotVersions && !node.responses.Unknown)
			observations.push({
				node: node.nodeId,
				state: 'Unknown',
				detail: 'Snapshot information is unavailable',
				href: nodeSection('clients', node.nodeId)
			})
		if (node.pause)
			observations.push({
				node: node.nodeId,
				state: 'Paused',
				detail: node.pause.reason,
				href: nodePath(node.nodeId)
			})
		if (!node.streams.length)
			observations.push({
				node: node.nodeId,
				state: 'No streams',
				detail: 'View xDS streams',
				href: nodeSection('clients', node.nodeId)
			})
	}
	const counts = data && streamCounts(data.clients.clients)
	const healthy = (data?.nodes.length ?? 0) - new Set(observations.map(row => row.node)).size
	const knowHealth = query.isSuccess && !query.isFetching && Boolean(data?.nodes.length)
	const inventory = [
		{ label: 'Nodes', value: data?.nodes.length, href: '/beta/nodes' },
		{ label: 'xDS streams', value: data?.clients.clients.length, href: '/beta/clients' }
	]
	const signals = [
		{ label: 'Streams with NACK', value: counts?.NACK, color: 'error.main', href: '/beta/clients?status=NACK' },
		{
			label: 'Awaiting ACK',
			value: counts?.['Awaiting ACK'],
			color: 'warning.main',
			href: '/beta/clients?status=Awaiting+ACK'
		}
	]
	return (
		<SectionTable
			title='Overview'
			tableTitle='Needs attention'
			columns={columns}
			rows={observations}
			rowId={row => JSON.stringify([row.node, row.state])}
			error={query.error}
			loading={query.isFetching}
			updatedAt={query.dataUpdatedAt}
			onRefresh={() => void query.refetch()}
			emptyText={
				knowHealth
					? 'All nodes are healthy.'
					: query.isSuccess
						? 'No nodes are visible.'
						: 'No current observations.'
			}
			description={data?.clients.controllerPod && 'xDS streams on ' + data.clients.controllerPod}
			summary={
				<Stack spacing={1.5}>
					<Paper
						variant='outlined'
						sx={{
							display: 'grid',
							gridTemplateColumns: {
								xs: '1fr 1fr',
								sm: knowHealth ? '1fr 1fr minmax(12em, 1.35fr)' : '1fr 1fr'
							},
							alignItems: 'stretch'
						}}
					>
						{inventory.map((item, i) => (
							<Link
								component={RouterLink}
								to={item.href}
								key={item.href}
								underline='hover'
								sx={{
									px: 2,
									py: 1.25,
									display: 'block',
									minWidth: 0,
									borderLeft: i ? 1 : 0,
									borderColor: 'divider'
								}}
							>
								<Typography
									component='span'
									display='block'
									variant='caption'
									fontWeight={600}
									color='text.secondary'
								>
									{item.label}
								</Typography>
								<Typography
									component='span'
									variant='subtitle2'
									sx={{ fontVariantNumeric: 'tabular-nums' }}
								>
									{item.value ?? 'Unavailable'}
								</Typography>
							</Link>
						))}
						{knowHealth && (
							<Box
								sx={{
									px: 2,
									py: 1.25,
									gridColumn: { xs: '1 / -1', sm: 'auto' },
									borderStyle: 'solid',
									borderWidth: 0,
									borderTopWidth: { xs: 1, sm: 0 },
									borderLeftWidth: { xs: 0, sm: 1 },
									borderColor: 'divider'
								}}
							>
								<Link component={RouterLink} to='/beta/nodes' variant='subtitle2'>
									{healthy} {healthy === 1 ? 'node is' : 'nodes are'} healthy
								</Link>
								<Typography display='block' variant='caption' color='text.secondary'>
									on the responding controller.
								</Typography>
							</Box>
						)}
					</Paper>
					<Box
						sx={{
							display: 'grid',
							gridTemplateColumns: { xs: '1fr', sm: 'repeat(2,minmax(0,1fr))' },
							gap: 1.5
						}}
					>
						{signals.map(signal => (
							<Paper
								key={signal.href}
								component={RouterLink}
								to={signal.href}
								variant='outlined'
								sx={{
									px: 2,
									py: 1.5,
									minWidth: 0,
									color: 'text.primary',
									textDecoration: 'none',
									borderLeftWidth: signal.value ? 3 : 1,
									borderLeftColor: signal.value ? signal.color : 'divider',
									'&:hover': { borderColor: 'primary.main' },
									'&:focus-visible': {
										outline: '2px solid',
										outlineColor: 'primary.main',
										outlineOffset: 2
									}
								}}
							>
								<Stack direction='row' justifyContent='space-between' alignItems='center' gap={1}>
									<Typography
										component='span'
										variant='body2'
										color={signal.value ? signal.color : 'text.secondary'}
										fontWeight={600}
									>
										{signal.label}
									</Typography>
									<ArrowForwardOutlined aria-hidden sx={{ color: 'primary.main', fontSize: 16 }} />
								</Stack>
								<Typography
									component='span'
									display='block'
									mt={0.5}
									variant={signal.value === undefined ? 'subtitle2' : 'h4'}
									sx={{ fontVariantNumeric: 'tabular-nums' }}
								>
									{signal.value ?? 'Unavailable'}
								</Typography>
							</Paper>
						))}
					</Box>
				</Stack>
			}
		/>
	)
}
