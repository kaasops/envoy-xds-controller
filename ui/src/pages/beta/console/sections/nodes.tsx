import { Button, Link, Stack, Typography } from '@mui/material'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { nodePath, fleetQueryKey, httpStatus, type SnapshotState } from '../../api'
import NodeListAction from '../../NodeListAction'
import { getNodeInventory, nodeInventoryKey, nodeSection, useNodeInventory, type NodeEntry } from '../nodeInventory'
import SectionTable, { type Column } from '../SectionTable'
import { streamId, streamRowId, type StreamStatus } from '../streams'

const statuses: StreamStatus[] = ['NACK', 'Awaiting ACK', 'ACK', 'Unknown']

const nodeHref = (row: NodeEntry) =>
	row.cached || row.freeze ? nodePath(row.nodeId) : nodeSection('clients', row.nodeId)

const columns: Column<NodeEntry>[] = [
	{
		key: 'node',
		label: 'Node ID',
		value: row => row.nodeId,
		render: row => (
			<Link component={RouterLink} to={nodeHref(row)}>
				{row.nodeId}
			</Link>
		)
	},
	{
		key: 'status',
		label: 'Stream responses',
		value: row => (row.streams.length ? statuses.filter(status => row.responses[status]) : ['No streams']),
		render: row => {
			if (!row.streams.length)
				return (
					<Link component={RouterLink} to={nodeSection('clients', row.nodeId)}>
						No streams
					</Link>
				)
			return (
				<Stack direction='row' flexWrap='wrap' gap={1.5}>
					{statuses
						.filter(status => row.responses[status])
						.map(status => (
							<Link
								key={status}
								component={RouterLink}
								to={nodeSection('clients', row.nodeId, status)}
								color={status === 'NACK' ? 'error.main' : 'primary.main'}
							>
								{row.responses[status]} {status}
							</Link>
						))}
				</Stack>
			)
		}
	},
	{
		key: 'streams',
		label: 'xDS streams',
		value: row => row.streams.length,
		align: 'right',
		render: row => (
			<Link
				component={RouterLink}
				to={nodeSection('clients', row.nodeId)}
				aria-label={row.streams.length + ' xDS streams for ' + row.nodeId}
			>
				{row.streams.length}
			</Link>
		)
	},
	...['Listeners', 'Routes', 'Clusters'].map(
		(label): Column<NodeEntry> => ({
			key: label.toLowerCase(),
			label,
			value: row => (row.cached ? 'Browse' : 'No snapshot'),
			render: row =>
				row.cached ? (
					<Link
						component={RouterLink}
						to={nodeSection(label.toLowerCase(), row.nodeId)}
						aria-label={label + ' for ' + row.nodeId}
					>
						Browse
					</Link>
				) : (
					'No snapshot'
				)
		})
	),
	{
		key: 'freeze',
		label: 'Publication',
		value: row => (row.freeze ? 'Frozen' : row.cached ? 'Not frozen' : 'No snapshot'),
		render: row =>
			row.freeze || row.cached ? (
				<Link component={RouterLink} to={nodePath(row.nodeId)}>
					{row.freeze ? 'Frozen' : 'Not frozen'}
				</Link>
			) : (
				'No snapshot'
			)
	}
]

function NodeDetail({ row, controllerPod }: { row: NodeEntry; controllerPod: string }) {
	return (
		<Stack spacing={1.5}>
			<Typography variant='body2'>
				{row.cached ? 'Snapshot present in the controller cache.' : 'No snapshot in the controller cache.'}
			</Typography>
			{row.freeze && (
				<Typography variant='body2'>
					Frozen {new Date(row.freeze.at).toLocaleString()} · {row.freeze.reason}
				</Typography>
			)}
			{(row.cached || row.freeze) && (
				<Link component={RouterLink} to={nodePath(row.nodeId)}>
					Inspect snapshots
				</Link>
			)}
			<Typography component='h4' variant='subtitle2'>
				xDS streams ({row.streams.length})
			</Typography>
			{row.streams.length ? (
				row.streams.map(stream => (
					<Link
						key={streamId(stream)}
						component={RouterLink}
						to={
							nodeSection('clients', row.nodeId) +
							'&selected=' +
							encodeURIComponent(streamRowId(controllerPod, stream))
						}
					>
						{stream.protocol}/{stream.streamId} · {stream.address}
					</Link>
				))
			) : (
				<Typography variant='body2' color='text.secondary'>
					No connected streams on this controller.
				</Typography>
			)}
		</Stack>
	)
}

export default function Nodes() {
	const [params, setParams] = useSearchParams()
	const query = useNodeInventory()
	const cache = useQueryClient()
	async function onFrozen(next: SnapshotState) {
		await cache.cancelQueries({ queryKey: nodeInventoryKey, exact: true })
		if ([401, 403].includes(httpStatus(cache.getQueryState(nodeInventoryKey)?.error) ?? 0)) return
		cache.setQueryData<Awaited<ReturnType<typeof getNodeInventory>>>(
			nodeInventoryKey,
			current =>
				current && {
					...current,
					nodes: current.nodes.map(node =>
						node.nodeId === next.nodeId ? { ...node, freeze: next.freeze ?? undefined } : node
					)
				}
		)
		await cache.invalidateQueries({ queryKey: fleetQueryKey, refetchType: 'none' })
	}
	const actions: Column<NodeEntry> = {
		key: 'actions',
		label: 'Actions',
		value: row => (row.freeze ? 'Unfreeze' : 'Freeze'),
		render: row => (
			<NodeListAction
				nodeId={row.nodeId}
				frozen={Boolean(row.freeze)}
				disabled={query.isFetching || query.isError}
				onFrozen={onFrozen}
			/>
		)
	}
	const frozenOnly = params.get('frozen') === 'true'
	const nodes = query.data?.nodes ?? []
	const frozenCount = nodes.filter(node => node.freeze).length
	const pod = query.data?.clients.controllerPod
	function filterFrozen(showFrozen: boolean) {
		setParams(current => {
			const next = new URLSearchParams(current)
			if (showFrozen) next.set('frozen', 'true')
			else next.delete('frozen')
			next.delete('selected')
			next.delete('page')
			return next
		})
	}
	return (
		<SectionTable
			title='Nodes'
			rows={frozenOnly ? nodes.filter(node => node.freeze) : nodes}
			columns={[...columns, actions]}
			loading={query.isFetching}
			error={query.error}
			updatedAt={query.dataUpdatedAt}
			onRefresh={() => void query.refetch()}
			description={
				query.data && (
					<>
						{nodes.length} nodes ·{' '}
						<Link
							component='button'
							type='button'
							sx={{ font: 'inherit', verticalAlign: 'baseline' }}
							onClick={() => filterFrozen(true)}
						>
							{frozenCount} frozen
						</Link>
						{pod && ' · xDS streams on ' + pod}
					</>
				)
			}
			rowId={node => node.nodeId}
			rowHref={nodeHref}
			detail={row => <NodeDetail row={row} controllerPod={pod ?? ''} />}
			filters={
				<Button
					variant={frozenOnly ? 'contained' : 'outlined'}
					aria-pressed={frozenOnly}
					onClick={() => filterFrozen(!frozenOnly)}
				>
					Frozen ({frozenCount})
				</Button>
			}
			emptyText={frozenOnly ? 'No frozen nodes.' : 'No nodes available.'}
		/>
	)
}
