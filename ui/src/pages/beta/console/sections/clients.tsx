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
import { manualQueryOptions, nodeOverviewPath, typeLabel } from '../../api'
import SectionTable, { type Column } from '../SectionTable'
import {
	clientsQueryKey,
	getClients,
	streamConfiguration,
	streamClient,
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

const statusColors = {
	NACK: 'error',
	'Awaiting ACK': 'warning',
	ACK: 'success',
	Subscribed: 'default',
	Unknown: 'default'
} as const

export function StatusBadges({ statuses }: { statuses: StreamStatus[] }) {
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

export function StreamClientLink({ stream, to }: { stream: ClientStream; to: string }) {
	const { value, source } = streamClient(stream)
	return (
		<Box sx={{ minWidth: '20ch', overflowWrap: 'anywhere' }}>
			<Link component={RouterLink} to={to}>
				{value}
			</Link>
			{source && (
				<Typography component='span' display='block' variant='caption' color='text.secondary'>
					{source}
				</Typography>
			)}
		</Box>
	)
}

function timeUtc(timestamp?: string) {
	if (!timestamp) return ''
	const date = new Date(timestamp)
	return Number.isNaN(date.getTime()) ? timestamp : date.toISOString().slice(0, 19).replace('T', ' ')
}

export function StreamDetails({
	stream,
	pod,
	versions,
	resourcePath
}: {
	stream: ClientStream
	pod: string
	versions?: Record<string, string>
	resourcePath?: (typeKey: string, name: string) => string
}) {
	const listener = stream.types.find(type => type.typeUrl === listenerType)
	const rejections = stream.types.flatMap(type => type.rejections.map(rejection => ({ type, rejection })))
	const { region, zone, subZone } = stream.locality ?? {}
	const locality = [region && 'Region ' + region, zone && 'Zone ' + zone, subZone && 'Sub-zone ' + subZone]
		.filter(Boolean)
		.join(' · ')
	return (
		<Stack gap={2} sx={{ minWidth: 0 }}>
			<Box
				component='dl'
				sx={{
					m: 0,
					display: 'grid',
					gap: 0.65,
					gridTemplateColumns: {
						xs: 'minmax(0, 0.85fr) minmax(0, 1.15fr)',
						sm: 'max-content minmax(0, 1fr)'
					},
					'& dt': { color: 'text.secondary' },
					'& dd': { m: 0, overflowWrap: 'anywhere' }
				}}
			>
				<dt>Stream</dt>
				<dd>{streamId(stream)}</dd>
				<dt>Identity source</dt>
				<dd>{streamClient(stream).source || 'Unavailable'}</dd>
				<dt>Envoy host</dt>
				<dd>{stream.host || 'Host not reported'}</dd>
				<dt>Forwarded IP</dt>
				<dd>
					{stream.forwardedAddress || 'Not reported'}
					{stream.forwardedAddress && (
						<Typography component='span' display='block' variant='caption' color='text.secondary'>
							First IP reported in X-Forwarded-For. Clients or proxies can supply it.
						</Typography>
					)}
				</dd>
				<dt>Connection peer</dt>
				<dd>{stream.address || 'Unavailable'}</dd>
				{locality && (
					<>
						<dt>Envoy locality</dt>
						<dd>{locality}</dd>
					</>
				)}
				<dt>Node ID</dt>
				<dd>{stream.nodeId || 'Not identified'}</dd>
				<dt>Controller pod</dt>
				<dd>{pod || 'Unknown'}</dd>
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
											{rejection.resources.map(ref => {
												const href =
													ref.name &&
													stream.nodeId &&
													(resourcePath
														? resourcePath(kind?.short || type.typeUrl, ref.name)
														: kind &&
															'/beta/' +
																kind.section +
																'?' +
																new URLSearchParams({
																	node: stream.nodeId,
																	q: ref.name
																}))
												return (
													<li key={ref.name}>
														{href ? (
															<Link
																component={RouterLink}
																to={href}
																aria-label={
																	resourcePath
																		? 'Find ' + ref.name + ' in the served snapshot'
																		: undefined
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
												)
											})}
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
				{stream.types.some(type => type.awaitingFirstResponse) && (
					<Typography variant='body2' color='text.secondary' sx={{ mb: 1 }}>
						For a type marked Subscribed, Envoy has requested resources and the controller has yet to send a
						response on this stream. Envoy can ACK only after it receives a response.
						{stream.protocol === 'delta' &&
							' After a Delta reconnect, Envoy may already have the current versions.'}
					</Typography>
				)}
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
	function streamHref(stream: ClientStream) {
		const selection = new URLSearchParams(params)
		selection.set('selected', streamRowId(pod, stream))
		return '?' + selection
	}
	const columns: Column<ClientStream>[] = [
		{
			key: 'client',
			label: 'Envoy client',
			value: stream => streamClient(stream).value,
			render: stream => <StreamClientLink stream={stream} to={streamHref(stream)} />
		},
		{ key: 'stream', label: 'Stream', value: streamId },
		{
			key: 'status',
			label: 'Status',
			value: streamStatuses,
			render: stream => {
				const statuses = streamStatuses(stream)
				return (
					<Link
						component={RouterLink}
						to={streamHref(stream)}
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
						to={nodeOverviewPath(stream.nodeId)}
						onClick={event => event.stopPropagation()}
					>
						{stream.nodeId}
					</Link>
				) : (
					'Not identified'
				)
		},
		{
			key: 'configuration',
			label: 'Configuration',
			value: stream => streamConfiguration(stream, query.data?.snapshotVersions)
		},
		{
			key: 'source',
			label: 'Identity source',
			value: stream => streamClient(stream).source || 'Unavailable',
			filterOnly: true
		},
		{ key: 'host', label: 'Envoy host', value: stream => stream.host || 'Host not reported', filterOnly: true },
		{
			key: 'forwardedAddress',
			label: 'Forwarded IP',
			value: stream => stream.forwardedAddress || '',
			filterOnly: true
		},
		{ key: 'address', label: 'Connection peer', value: stream => stream.address, filterOnly: true },
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
				<StreamDetails stream={stream} pod={pod} versions={query.data?.snapshotVersions?.[stream.nodeId]} />
			)}
			emptyText='No connected streams.'
		/>
	)
}
