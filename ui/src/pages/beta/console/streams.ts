import client from '../../../api/axiosApiClient'

export type StreamStatus = 'ACK' | 'Awaiting ACK' | 'NACK' | 'Unknown'

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
	address: string
	envoyVersion: string
	connectedAt?: string
	types: StreamType[]
	unknownNacks?: number
}

export interface ClientsResponse {
	controllerPod: string
	snapshotVersions: Record<string, Record<string, string>>
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
// Stream counters restart with the process, including a container restart in the same pod.
export const streamRowId = (controllerPod: string, stream: ClientStream) =>
	controllerPod + '/' + streamId(stream) + (stream.connectedAt ? '/' + stream.connectedAt : '')

export function streamStatuses(stream: Pick<ClientStream, 'types'>): StreamStatus[] {
	const types = stream.types
	const statuses: StreamStatus[] = []
	if (types.some(type => type.rejections.length > 0)) statuses.push('NACK')
	if (types.some(type => type.awaitingAckSince)) statuses.push('Awaiting ACK')
	if (statuses.length) return statuses

	// After the controller evicts a rejection it can no longer track whether that resource recovered.
	const allAcknowledged =
		types.length > 0 && types.every(type => !type.droppedRejections && (type.ackedVersion || type.lastAckAt))
	return [allAcknowledged ? 'ACK' : 'Unknown']
}

export function streamCounts(streams: readonly ClientStream[]): Record<StreamStatus, number> {
	const counts = { NACK: 0, 'Awaiting ACK': 0, ACK: 0, Unknown: 0 }
	for (const stream of streams) {
		for (const status of streamStatuses(stream)) counts[status]++
	}
	return counts
}
