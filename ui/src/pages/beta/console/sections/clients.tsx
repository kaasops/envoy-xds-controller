import {
	Box,
	Chip,
	Link,
	Stack,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Typography
} from '@mui/material'
import { useQuery } from '@tanstack/react-query'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'
import { manualQueryOptions, typeLabel } from '../../api'
import SectionTable, { type Column } from '../SectionTable'
import {
	clientsQueryKey,
	getClients,
	streamId,
	streamRowId,
	streamStatuses,
	type ClientStream,
	type StreamStatus
} from '../streams'

const resourceTypes: Record<string, { short: string; section: string }> = {
	'type.googleapis.com/envoy.config.listener.v3.Listener': { short: 'LDS', section: 'listeners' },
	'type.googleapis.com/envoy.config.route.v3.RouteConfiguration': { short: 'RDS', section: 'routes' },
	'type.googleapis.com/envoy.config.cluster.v3.Cluster': { short: 'CDS', section: 'clusters' },
	'type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret': { short: 'SDS', section: 'secrets' }
}
const listenerType = 'type.googleapis.com/envoy.config.listener.v3.Listener'

const statusColors = { NACK: 'error', 'Awaiting ACK': 'warning', ACK: 'success', Unknown: 'default' } as const

function StatusBadges({ statuses }: { statuses: StreamStatus[] }) {
	return (
		<Stack component='span' direction='row' gap={0.5} flexWrap='wrap'>
			{statuses.map(status => (
				<Chip
					key={status}
					component='span'
					size='small'
					variant='outlined'
					color={statusColors[status]}
					label={status}
				/>
			))}
		</Stack>
	)
}

function timeUtc(timestamp?: string) {
	if (!timestamp) return ''
	const date = new Date(timestamp)
	return Number.isNaN(date.getTime()) ? timestamp : date.toISOString().slice(0, 19).replace('T', ' ')
}

function StreamDetails({
	stream,
	pod,
	versions
}: {
	stream: ClientStream
	pod: string
	versions?: Record<string, string>
}) {
	const listener = stream.types.find(type => type.typeUrl === listenerType)
	const rejections = stream.types.flatMap(type => type.rejections.map(rejection => ({ type, rejection })))
	return (
		<Stack gap={2} sx={{ minWidth: 0 }}>
			<Box
				component='dl'
				sx={{
					m: 0,
					display: 'grid',
					gap: 0.65,
					gridTemplateColumns: 'max-content minmax(0,1fr)',
					'& dt': { color: 'text.secondary' },
					'& dd': { m: 0, overflowWrap: 'anywhere' }
				}}
			>
				<dt>Controller pod</dt>
				<dd>{pod || 'Unknown'}</dd>
				<dt>Node</dt>
				<dd>{stream.nodeId || 'Not identified'}</dd>
				<dt>Address</dt>
				<dd>{stream.address || 'Unknown'}</dd>
				<dt>Envoy version</dt>
				<dd>{stream.envoyVersion || 'Unknown'}</dd>
				<dt>Connected (UTC)</dt>
				<dd>{timeUtc(stream.connectedAt) || 'Unknown'}</dd>
				{versions?.[listenerType] && (
					<>
						<dt>LDS snapshot version</dt>
						<dd>{versions[listenerType]}</dd>
					</>
				)}
				{listener && (
					<>
						<dt>Last ACKed LDS version</dt>
						<dd>{listener.ackedVersion || 'None'}</dd>
					</>
				)}
			</Box>
			{Boolean(stream.unknownNacks) && (
				<Typography variant='body2' color='text.secondary'>
					NACKs with an unrecognized nonce: {stream.unknownNacks}
				</Typography>
			)}
			{rejections.length > 0 && (
				<Box>
					<Typography component='h3' variant='subtitle1' fontWeight={600}>
						Rejections
					</Typography>
					<Stack gap={1.5} sx={{ mt: 1 }}>
						{rejections.map(({ type, rejection }) => {
							const kind = resourceTypes[type.typeUrl]
							return (
								<Box
									key={type.typeUrl + '/' + rejection.nonce}
									sx={{ borderLeft: 3, borderColor: 'error.main', pl: 1.5 }}
								>
									<Typography variant='subtitle2'>
										{kind?.short || typeLabel(type.typeUrl)} NACK · version{' '}
										{rejection.responseVersion || 'unknown'}
									</Typography>
									<Typography variant='caption' color='text.secondary'>
										{timeUtc(rejection.rejectedAt)} UTC · nonce {rejection.nonce} ·{' '}
										{rejection.count} {rejection.count === 1 ? 'response' : 'responses'}
									</Typography>
									<Typography
										variant='body2'
										sx={{ my: 0.5, whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}
									>
										{rejection.error}
									</Typography>
									{rejection.resources?.length ? (
										<Box
											component='ul'
											aria-label='Resources in rejected response'
											sx={{ m: 0, pl: 2 }}
										>
											{rejection.resources.map(ref => (
												<li key={ref.name}>
													{kind && stream.nodeId ? (
														<Link
															component={RouterLink}
															to={
																'/beta/' +
																kind.section +
																'?' +
																new URLSearchParams({
																	node: stream.nodeId,
																	q: ref.name
																})
															}
														>
															{ref.name}
														</Link>
													) : (
														ref.name
													)}
													{ref.version && (
														<Typography
															component='span'
															variant='caption'
															color='text.secondary'
															sx={{ ml: 1, overflowWrap: 'anywhere' }}
														>
															{ref.version}
														</Typography>
													)}
												</li>
											))}
										</Box>
									) : (
										<Typography variant='body2' color='text.secondary'>
											{stream.protocol === 'sotw'
												? 'Whole response rejected.'
												: 'No resource names recorded.'}
										</Typography>
									)}
								</Box>
							)
						})}
					</Stack>
				</Box>
			)}
			<Box>
				<Typography component='h3' variant='subtitle1' fontWeight={600}>
					Resource types
				</Typography>
				{stream.types.length ? (
					<TableContainer>
						<Table size='small' aria-label='Stream resource types'>
							<TableHead>
								<TableRow>
									<TableCell>Type</TableCell>
									<TableCell>Response</TableCell>
								</TableRow>
							</TableHead>
							<TableBody>
								{stream.types.map(type => {
									const kind = resourceTypes[type.typeUrl]
									return (
										<TableRow key={type.typeUrl}>
											<TableCell
												title={type.typeUrl}
												sx={{ verticalAlign: 'top', width: 70, overflowWrap: 'normal' }}
											>
												{kind && stream.nodeId ? (
													<Link
														component={RouterLink}
														to={
															'/beta/' +
															kind.section +
															'?' +
															new URLSearchParams({ node: stream.nodeId })
														}
													>
														{kind.short}
													</Link>
												) : (
													typeLabel(type.typeUrl)
												)}
											</TableCell>
											<TableCell>
												<StatusBadges statuses={streamStatuses({ types: [type] })} />
												{(type.ackedVersion || type.lastAckAt) && (
													<Typography component='div' variant='caption' sx={{ mt: 0.5 }}>
														{type.ackedVersion
															? 'ACKed version ' + type.ackedVersion
															: 'Last ACK'}
														{type.lastAckAt && ' · ' + timeUtc(type.lastAckAt) + ' UTC'}
													</Typography>
												)}
												{type.awaitingAckSince && (
													<Typography component='div' variant='caption' sx={{ mt: 0.5 }}>
														Awaiting since {timeUtc(type.awaitingAckSince)} UTC
													</Typography>
												)}

												{!!type.droppedRejections && (
													<Typography variant='caption'>
														{type.droppedRejections} older rejections evicted
													</Typography>
												)}
											</TableCell>
										</TableRow>
									)
								})}
							</TableBody>
						</Table>
					</TableContainer>
				) : (
					<Typography variant='body2' color='text.secondary'>
						No resource responses recorded on this stream.
					</Typography>
				)}
			</Box>
		</Stack>
	)
}

export default function Clients() {
	const [params] = useSearchParams()
	const nodeId = params.get('node') || undefined
	const query = useQuery({
		...manualQueryOptions,
		queryKey: clientsQueryKey(nodeId),
		queryFn: ({ signal }) => getClients(signal, nodeId),
		refetchOnMount: 'always'
	})
	const pod = query.data?.controllerPod || ''
	const columns: Column<ClientStream>[] = [
		{ key: 'stream', label: 'Stream', value: streamId },
		{
			key: 'status',
			label: 'Status',
			value: streamStatuses,
			render: stream => {
				const selection = new URLSearchParams(params)
				const statuses = streamStatuses(stream)
				selection.set('selected', streamRowId(pod, stream))
				return (
					<Link
						component={RouterLink}
						to={'?' + selection}
						underline='none'
						onClick={event => event.stopPropagation()}
						aria-label={streamId(stream) + ' responses: ' + statuses.join(', ')}
					>
						<StatusBadges statuses={statuses} />
					</Link>
				)
			}
		},
		{
			key: 'node',
			label: 'Node',
			value: stream => stream.nodeId,
			render: stream =>
				stream.nodeId ? (
					<Link
						component={RouterLink}
						to={'/beta/nodes?' + new URLSearchParams({ selected: stream.nodeId })}
						onClick={event => event.stopPropagation()}
					>
						{stream.nodeId}
					</Link>
				) : (
					'Not identified'
				)
		},
		{ key: 'address', label: 'Address', value: stream => stream.address },
		{ key: 'envoy', label: 'Envoy', value: stream => stream.envoyVersion },
		{ key: 'connected', label: 'Connected (UTC)', value: stream => timeUtc(stream.connectedAt) }
	]
	return (
		<SectionTable
			title='xDS streams'
			description={pod ? 'Connections to ' + pod : undefined}
			rows={query.data?.clients ?? []}
			rowId={stream => streamRowId(pod, stream)}
			columns={columns}
			loading={query.isFetching}
			error={query.error}
			updatedAt={query.dataUpdatedAt}
			onRefresh={() => void query.refetch()}
			detail={stream => (
				<StreamDetails stream={stream} pod={pod} versions={query.data?.snapshotVersions[stream.nodeId]} />
			)}
			emptyText='No connected streams.'
		/>
	)
}
