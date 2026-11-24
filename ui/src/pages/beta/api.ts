import axios from 'axios'
import client from '../../api/axiosApiClient'

export interface SnapshotInfo {
	id: string
	publishedAt?: string
	builtAt?: string
	versions: Record<string, string>
	counts: Record<string, number>
	error?: string
}

interface Pause {
	by: string
	at: string
	reason: string
}

export interface SnapshotState {
	nodeId: string
	controllerPod: string
	canControl: boolean
	paused: boolean
	pause: Pause | null
	served: SnapshotInfo | null
	pending: SnapshotInfo | null
	history: SnapshotInfo[]
}

export interface ResourceChange {
	name: string
	typeUrl: string
	change: 'same' | 'sameDecoded' | 'changed' | 'added' | 'removed'
	fromSha256: string
	toSha256: string
}

export interface SnapshotDiff {
	from: SnapshotInfo
	to: SnapshotInfo
	summary: Record<string, Record<ResourceChange['change'], number>>
	resources: ResourceChange[]
}

export interface SnapshotResource {
	name: string
	typeUrl: string
	sha256: string
	json?: unknown
}

export const fleetQueryKey = ['beta', 'fleet'] as const
export const snapshotQueryKey = (nodeId: string) => ['beta', 'snapshot-state', nodeId] as const

export const manualQueryOptions = {
	retry: false,
	retryOnMount: false,
	networkMode: 'always',
	refetchOnWindowFocus: false,
	refetchOnReconnect: false,
	refetchOnMount: false,
	refetchInterval: false
} as const

const nodeUrl = (nodeId: string) => '/nodes/' + encodeURIComponent(nodeId)
export const nodePath = (nodeId: string) => '/beta' + nodeUrl(nodeId)
export function nodeOverviewPath(nodeId: string, tab?: string, selected?: string) {
	const params = new URLSearchParams()
	if (tab) params.set('tab', tab)
	if (selected) params.set('selected', selected)
	return nodePath(nodeId) + '/overview' + (params.size ? '?' + params : '')
}
export const typeLabel = (url: string) => url.split('.').pop() || url

export function httpStatus(error: unknown) {
	return axios.isAxiosError(error) ? error.response?.status : undefined
}

export function apiError(error: unknown): string {
	if (axios.isAxiosError(error)) {
		const detail = error.response?.data?.error
		if (typeof detail === 'string' && detail) return detail
	}
	return error instanceof Error ? error.message : 'The request failed.'
}

export async function getSnapshotState(nodeId: string, signal?: AbortSignal) {
	return (await client.get<SnapshotState>(nodeUrl(nodeId) + '/snapshots', { signal })).data
}

export async function getSnapshotDiff(nodeId: string, from: string, to: string, signal?: AbortSignal) {
	return (
		await client.get<SnapshotDiff>(nodeUrl(nodeId) + '/snapshots/diff', { signal, params: { from, to, all: true } })
	).data
}

export async function getSnapshotResource(
	nodeId: string,
	snapshot: string,
	typeUrl: string,
	name: string,
	signal?: AbortSignal
) {
	return (
		await client.get<SnapshotResource>(nodeUrl(nodeId) + '/snapshots/resource', {
			signal,
			params: { snapshot, type: typeUrl, name }
		})
	).data
}

export async function pauseNode(nodeId: string, reason: string) {
	return (await client.post<SnapshotState>(nodeUrl(nodeId) + '/pause', { reason })).data
}

export async function resumeNode(nodeId: string, expectPending: string) {
	return (await client.post<SnapshotState>(nodeUrl(nodeId) + '/resume', { expectPending })).data
}

export async function rollbackNode(nodeId: string, to: string, expectServed: string) {
	return (await client.post<SnapshotState>(nodeUrl(nodeId) + '/rollback', { to, expectServed })).data
}
