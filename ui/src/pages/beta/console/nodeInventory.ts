import { useQuery } from '@tanstack/react-query'
import client from '../../../api/axiosApiClient'
import { fleetQueryKey, manualQueryOptions, type SnapshotState } from '../api'
import { getClients, streamCounts, type ClientStream } from './streams'

type Freeze = NonNullable<SnapshotState['freeze']>
type Counts = Record<'listeners' | 'routes' | 'clusters', number>
type CachedNode = string | { node_id: string; counts?: Counts }

export async function getFreezes(signal: AbortSignal) {
	const { data } = await client.get<{ frozenNodes: Record<string, Freeze> | null }>('/frozenNodes', { signal })
	return new Map(Object.entries(data.frozenNodes ?? {}))
}

export async function getNodeInventory(signal: AbortSignal) {
	const [cached, frozen, clients] = await Promise.all([
		client.get<CachedNode[] | null>('/nodeIDs', { signal, params: { include: 'counts' } }),
		getFreezes(signal),
		getClients(signal)
	])
	const byId = new Map((cached.data ?? []).map(entry => [typeof entry === 'string' ? entry : entry.node_id, entry]))
	const groups = new Map<string, ClientStream[]>()
	for (const stream of clients.clients) {
		if (!stream.nodeId) continue
		const group = groups.get(stream.nodeId)
		if (group) group.push(stream)
		else groups.set(stream.nodeId, [stream])
	}
	const ids = Array.from(new Set([...byId.keys(), ...frozen.keys(), ...groups.keys()])).sort()
	const nodes = ids.map(nodeId => {
		const streams = groups.get(nodeId) ?? []
		const entry = byId.get(nodeId)
		const counts = typeof entry === 'object' ? entry.counts : undefined
		return {
			nodeId,
			streams,
			counts,
			responses: streamCounts(streams),
			freeze: frozen.get(nodeId),
			cached: typeof entry === 'string' || Boolean(counts)
		}
	})
	return { nodes, clients }
}

export type NodeEntry = Awaited<ReturnType<typeof getNodeInventory>>['nodes'][number]

export const nodeInventoryKey = [...fleetQueryKey, 'console-nodes'] as const

export function useNodeInventory() {
	return useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: nodeInventoryKey,
		queryFn: ({ signal }) => getNodeInventory(signal)
	})
}

interface SnapshotVersion {
	node_id: string
	versions: Record<string, string>
}

async function getSnapshotInventory(signal: AbortSignal) {
	const [snapshots, frozen] = await Promise.all([
		client.get<SnapshotVersion[] | null>('/nodeIDs/versions', { signal }),
		getFreezes(signal)
	])
	const byNode = new Map((snapshots.data ?? []).map(snapshot => [snapshot.node_id, snapshot.versions]))
	const ids = Array.from(new Set([...byNode.keys(), ...frozen.keys()])).sort()
	return ids.map(nodeId => ({ nodeId, versions: byNode.get(nodeId), freeze: frozen.get(nodeId) }))
}

export type SnapshotEntry = Awaited<ReturnType<typeof getSnapshotInventory>>[number]

export function useSnapshotInventory() {
	return useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: [...fleetQueryKey, 'snapshot-versions'],
		queryFn: ({ signal }) => getSnapshotInventory(signal)
	})
}

export function nodeSection(section: string, nodeId: string, status?: string) {
	const params = new URLSearchParams({ node: nodeId })
	if (status) params.set('status', status)
	return '/beta/' + section + '?' + params.toString()
}
