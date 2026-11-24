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

const columns: Column<Observation>[] = [
	{ key: 'event', label: 'Event', value: row => row.title },
	{
		key: 'at',
		label: 'Time (UTC)',
		value: row => row.at || '',
		render: row => (row.at ? new Date(row.at).toISOString().slice(0, 19).replace('T', ' ') : 'Not reported')
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
	const [params] = useSearchParams()
	const node = params.get('node') || undefined
	const clients = useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: clientsQueryKey(node),
		queryFn: ({ signal }) => getClients(signal, node)
	})
	const freezes = useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: [...fleetQueryKey, 'freeze-records'],
		queryFn: async ({ signal }) => {
			const { data } = await client.get<{
				frozenNodes: Record<string, NonNullable<SnapshotState['freeze']>> | null
			}>('/frozenNodes', { signal })
			return data.frozenNodes || {}
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
	if (readable(freezes.error)) {
		for (const [nodeId, freeze] of Object.entries(freezes.data || {})) {
			observations.push({
				id: JSON.stringify(['freeze', nodeId]),
				title: 'Snapshot frozen',
				source: 'Snapshot',
				nodes: [nodeId],
				subject: nodeId,
				at: freeze.at,
				message: freeze.reason,
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
					title: 'Publication failed',
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
		{ label: 'Snapshot freezes', query: freezes },
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
					'Current rejections, frozen snapshots and publication failures.',
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
