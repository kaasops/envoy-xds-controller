import { ConsoleTime } from '../../time/ConsoleTime'
import { Button, Link, Stack, Typography } from '@mui/material'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { nodeOverviewPath, nodePath, fleetQueryKey, httpStatus, type SnapshotState } from '../../api'
import StatusHelp from '../StatusHelp'
import NodeListAction from '../../NodeListAction'
import { getNodeInventory, nodeInventoryKey, nodeSection, useNodeInventory, type NodeEntry } from '../nodeInventory'
import SectionTable, { type Column } from '../SectionTable'
import InventorySummary from '../InventorySummary'
import StreamResponseChip from '../StreamResponseChip'
import { streamClientLabel, streamClientSearchValues, streamId, streamRowId, type StreamStatus } from '../streams'

const statuses: StreamStatus[] = ['NACK', 'Awaiting ACK', 'ACK', 'Subscribed', 'Unknown']

const nodeHref = (row: NodeEntry) => nodeOverviewPath(row.nodeId)

const columns: Column<NodeEntry>[] = [
	{
		key: 'node',
		identifier: true,
		label: 'Node ID',
		value: row => row.nodeId,
		render: row => (
			<Link component={RouterLink} to={nodeHref(row)}>
				{row.nodeId}
			</Link>
		)
	},
	{
		key: 'client',
		label: 'Envoy client',
		value: row => row.streams.flatMap(streamClientSearchValues),
		filterOnly: true
	},
	{
		key: 'status',
		label: 'Sync status',
		value: row => (row.streams.length ? statuses.filter(status => row.responses[status]) : ['No streams']),
		render: row => {
			if (!row.streams.length)
				return (
					<StatusHelp status='No streams' to={nodeSection('clients', row.nodeId)}>
						No streams
					</StatusHelp>
				)
			return (
				<Stack direction='row' flexWrap='wrap' gap={0.75}>
					{statuses
						.filter(status => row.responses[status])
						.map(status => (
							<StreamResponseChip
								key={status}
								status={status}
								count={row.responses[status]}
								to={nodeSection('clients', row.nodeId, status)}
							/>
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
	...(['listeners', 'routes', 'clusters'] as const).map(
		(section): Column<NodeEntry> => ({
			key: section,
			label: section[0].toUpperCase() + section.slice(1),
			align: 'right',
			value: row => row.counts?.[section] ?? (row.cached ? 'Browse' : 'No snapshot'),
			render: row =>
				row.cached ? (
					<Link
						component={RouterLink}
						to={nodeSection(section, row.nodeId)}
						aria-label={section + ' for ' + row.nodeId}
					>
						{row.counts?.[section] ?? 'Browse'}
					</Link>
				) : (
					'No snapshot'
				)
		})
	),
	{
		key: 'pause',
		label: 'Updates',
		value: row => (row.pause ? 'Paused' : 'Active'),
		render: row =>
			row.pause ? (
				<StatusHelp status='Paused' to={nodePath(row.nodeId)}>
					Paused
				</StatusHelp>
			) : row.cached ? (
				<Link component={RouterLink} to={nodePath(row.nodeId)}>
					Active
				</Link>
			) : (
				'Active'
			)
	}
]

function NodeDetail({ row, controllerPod }: { row: NodeEntry; controllerPod: string }) {
	return (
		<Stack spacing={1.5}>
			<Typography variant='body2'>
				{row.cached ? 'Snapshot present in the controller cache.' : 'No snapshot in the controller cache.'}
			</Typography>
			{row.pause && (
				<Typography variant='body2'>
					Paused <ConsoleTime value={row.pause.at} /> · {row.pause.reason}
				</Typography>
			)}
			{(row.cached || row.pause) && (
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
						{streamClientLabel(stream)} · {streamId(stream)}
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
	async function onPaused(next: SnapshotState) {
		await cache.cancelQueries({ queryKey: nodeInventoryKey, exact: true })
		if ([401, 403].includes(httpStatus(cache.getQueryState(nodeInventoryKey)?.error) ?? 0)) return
		cache.setQueryData<Awaited<ReturnType<typeof getNodeInventory>>>(
			nodeInventoryKey,
			current =>
				current && {
					...current,
					nodes: current.nodes.map(node =>
						node.nodeId === next.nodeId ? { ...node, pause: next.pause ?? undefined } : node
					)
				}
		)
		await cache.invalidateQueries({ queryKey: fleetQueryKey, refetchType: 'none' })
	}
	const actions: Column<NodeEntry> = {
		key: 'actions',
		label: 'Actions',
		value: row => (row.pause ? 'Resume' : 'Pause'),
		render: row => (
			<NodeListAction
				nodeId={row.nodeId}
				paused={Boolean(row.pause)}
				disabled={query.isFetching || query.isError}
				onPaused={onPaused}
			/>
		)
	}
	const pausedOnly = params.get('paused') === 'true'
	const nodes = query.data?.nodes ?? []
	const pausedCount = nodes.filter(node => node.pause).length
	const pod = query.data?.clients.controllerPod
	function togglePaused() {
		setParams(current => {
			const next = new URLSearchParams(current)
			if (next.get('paused') === 'true') next.delete('paused')
			else next.set('paused', 'true')
			next.delete('selected')
			next.delete('page')
			return next
		})
	}
	return (
		<SectionTable
			title='Nodes'
			tableHeading='Stream and snapshot state'
			rows={pausedOnly ? nodes.filter(node => node.pause) : nodes}
			columns={[...columns, actions]}
			loading={query.isFetching}
			error={query.error}
			updatedAt={query.dataUpdatedAt}
			onRefresh={() => void query.refetch()}
			summary={
				query.data && (
					<InventorySummary
						fields={[
							{ label: 'Known nodes', value: nodes.length },
							{
								label: 'Updates',
								value: (
									<Button
										size='small'
										variant={pausedOnly ? 'contained' : 'outlined'}
										aria-pressed={pausedOnly}
										onClick={togglePaused}
									>
										{pausedCount} paused
									</Button>
								)
							},
							...(pod ? [{ label: 'xDS controller', value: <code>{pod}</code> }] : [])
						]}
					/>
				)
			}
			rowId={node => node.nodeId}
			rowHref={nodeHref}
			detail={row => <NodeDetail row={row} controllerPod={pod ?? ''} />}
			filters={
				<Button
					variant={pausedOnly ? 'contained' : 'outlined'}
					aria-pressed={pausedOnly}
					onClick={togglePaused}
				>
					Paused ({pausedCount})
				</Button>
			}
			emptyText={pausedOnly ? 'No paused nodes.' : 'No nodes available.'}
		/>
	)
}
