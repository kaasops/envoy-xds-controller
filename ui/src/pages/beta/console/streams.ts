import client from '../../../api/axiosApiClient'

export type StreamStatus = 'ACK' | 'Awaiting ACK' | 'NACK' | 'Subscribed' | 'Unknown'

export interface Rejection {
	nonce: string
	responseVersion: string
	rejectedAt: string
	count: number
	error: string
	resources: { name: string; version?: string }[] | null
}

export interface StreamType {
	typeUrl: string
	awaitingFirstResponse?: boolean
	ackedVersion?: string
	lastAckAt?: string
	awaitingAckSince?: string
	rejections: Rejection[]
	droppedRejections?: number
}

export interface ClientStream {
	protocol: 'sotw' | 'delta'
	streamId: number
	nodeId: string
	host?: string
	forwardedAddress?: string
	locality?: { region?: string; zone?: string; subZone?: string }
	address: string
	envoyVersion: string
	connectedAt?: string
	types: StreamType[]
	unknownNacks?: number
}

export interface ClientsResponse {
	controllerPod: string
	snapshotVersions?: Record<string, Record<string, string>> | null
	clients: ClientStream[]
}

export const clientsQueryKey = (nodeId?: string) => ['beta', 'xds-streams', nodeId || ''] as const

export async function getClients(signal?: AbortSignal, nodeId?: string) {
	const { data } = await client.get<ClientsResponse>('/clients', {
		signal,
		params: nodeId ? { node_id: nodeId } : undefined
	})
	return data
}

export const streamId = (stream: ClientStream) => stream.protocol + '/' + stream.streamId
export function streamClient(stream: ClientStream) {
	if (stream.host) return { value: stream.host, source: 'Envoy host' }
	if (stream.forwardedAddress) return { value: stream.forwardedAddress, source: 'Forwarded IP' }
	if (stream.address) return { value: stream.address, source: 'Connection peer' }
	return { value: 'Identity unavailable', source: '' }
}

export function streamClientLabel(stream: ClientStream) {
	const { value, source } = streamClient(stream)
	return source && !stream.host ? source + ' · ' + value : value
}

export const streamClientSearchValues = (stream: ClientStream) =>
	[streamClientLabel(stream), stream.host || '', stream.forwardedAddress || '', stream.address].filter(Boolean)

// Stream counters restart with the process, including a container restart in the same pod.
export const streamRowId = (controllerPod: string, stream: ClientStream) =>
	controllerPod + '/' + streamId(stream) + (stream.connectedAt ? '/' + stream.connectedAt : '')

export const noConfiguration = 'No configuration on this controller'

export function streamConfiguration(
	stream: Pick<ClientStream, 'nodeId'>,
	versions: ClientsResponse['snapshotVersions']
) {
	if (!stream.nodeId || !versions) return 'Unknown'
	return Object.prototype.hasOwnProperty.call(versions, stream.nodeId) ? 'Snapshot available' : noConfiguration
}

export function nodesWithoutSnapshot(response?: ClientsResponse): Set<string> {
	const missing = new Set<string>()
	for (const stream of response?.clients ?? []) {
		if (streamConfiguration(stream, response?.snapshotVersions) === noConfiguration) missing.add(stream.nodeId)
	}
	return missing
}

export function streamStatuses(stream: Pick<ClientStream, 'types'>): StreamStatus[] {
	const types = stream.types
	const statuses: StreamStatus[] = []
	if (types.some(type => type.rejections.length > 0)) statuses.push('NACK')
	if (types.some(type => type.awaitingAckSince)) statuses.push('Awaiting ACK')
	if (statuses.length) return statuses

	// After the controller evicts a rejection it can no longer track whether that resource recovered.
	const responses = types.filter(type => !type.awaitingFirstResponse)
	if (!responses.length) return [types.length ? 'Subscribed' : 'Unknown']
	const allAcknowledged = responses.every(type => !type.droppedRejections && (type.ackedVersion || type.lastAckAt))
	return [allAcknowledged ? 'ACK' : 'Unknown']
}

export function streamCounts(streams: readonly ClientStream[]): Record<StreamStatus, number> {
	const counts = { NACK: 0, 'Awaiting ACK': 0, ACK: 0, Subscribed: 0, Unknown: 0 }
	for (const stream of streams) {
		for (const status of streamStatuses(stream)) counts[status]++
	}
	return counts
}
