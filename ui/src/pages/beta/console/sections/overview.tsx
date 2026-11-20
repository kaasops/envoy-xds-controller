import { Box, Link, Paper, Stack, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { httpStatus, nodePath } from '../../api'
import { nodeSection, useNodeInventory } from '../nodeInventory'
import SectionTable, { type Column } from '../SectionTable'
import { streamCounts } from '../streams'

interface Observation {
	node: string
	state: string
	detail: string
	href: string
}

const columns: Column<Observation>[] = [
	{
		key: 'node',
		label: 'Node ID',
		value: row => row.node,
		render: row => (
			<Link component={RouterLink} to={'/beta/nodes?' + new URLSearchParams({ selected: row.node })}>
				{row.node}
			</Link>
		)
	},
	{ key: 'status', label: 'Observation', value: row => row.state },
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
	for (const node of data?.nodes ?? []) {
		for (const state of ['NACK', 'Awaiting ACK', 'Unknown'] as const) {
			const count = node.responses[state]
			if (count)
				observations.push({
					node: node.nodeId,
					state,
					detail: count + (count === 1 ? ' stream' : ' streams'),
					href: nodeSection('clients', node.nodeId, state)
				})
		}
		if (node.freeze)
			observations.push({
				node: node.nodeId,
				state: 'Frozen',
				detail: node.freeze.reason,
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
	const metrics = [
		{ label: 'Nodes', value: data?.nodes.length, href: '/beta/nodes' },
		{ label: 'xDS streams', value: data?.clients.clients.length, href: '/beta/clients' },
		{ label: 'Streams with NACK', value: counts?.NACK, href: '/beta/clients?status=NACK' },
		{ label: 'Awaiting ACK', value: counts?.['Awaiting ACK'], href: '/beta/clients?status=Awaiting+ACK' }
	]
	return (
		<SectionTable
			title='Overview'
			columns={columns}
			rows={observations}
			rowId={row => JSON.stringify([row.node, row.state])}
			error={query.error}
			loading={query.isFetching}
			updatedAt={query.dataUpdatedAt}
			onRefresh={() => void query.refetch()}
			emptyText='No pending responses, rejections, frozen nodes or missing streams in this view.'
			description={
				<Stack component='span' spacing={1.5}>
					{data?.clients.controllerPod && (
						<Typography component='span' variant='body2'>
							xDS streams on {data.clients.controllerPod}
						</Typography>
					)}
					<Box
						component='span'
						sx={{
							display: 'grid',
							gridTemplateColumns: { xs: 'repeat(2, minmax(0,1fr))', md: 'repeat(4, minmax(0,1fr))' },
							gap: 1
						}}
					>
						{metrics.map(metric => (
							<Paper
								key={metric.label}
								variant='outlined'
								component={RouterLink}
								to={metric.href}
								sx={{
									p: 1.5,
									color: 'text.primary',
									textDecoration: 'none',
									'&:hover': { borderColor: 'primary.main' }
								}}
							>
								<Typography component='span' variant='body2' color='text.secondary'>
									{metric.label}
								</Typography>
								<Typography component='span' variant='h5' sx={{ display: 'block' }}>
									{metric.value ?? 'Unavailable'}
								</Typography>
							</Paper>
						))}
					</Box>
				</Stack>
			}
		/>
	)
}
