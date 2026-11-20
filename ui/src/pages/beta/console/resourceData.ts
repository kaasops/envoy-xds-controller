import {
	accessGroupsServiceClient,
	accessLogServiceClient,
	httpFilterServiceClient,
	listenerServiceClient,
	routeServiceClient,
	templateServiceClient,
	transport,
	virtualServiceClient
} from '../../../api/grpc/client'
import { createClient } from '@connectrpc/connect'
import { ClusterStoreService } from '../../../gen/cluster/v1/cluster_pb'
import { PolicyStoreService } from '../../../gen/policy/v1/policy_pb'
import { ListenerType } from '../../../gen/listener/v1/listener_pb'
import type { VirtualServiceListItem } from '../../../gen/virtual_service/v1/virtual_service_pb'

const policyClient = createClient(PolicyStoreService, transport)
const clusterClient = createClient(ClusterStoreService, transport)
export const virtualServicesKey = ['beta', 'virtual-services'] as const

export async function loadAccessGroups(signal: AbortSignal) {
	const { items } = await accessGroupsServiceClient.listAccessGroups({}, { signal })
	return [...new Set(items.map(item => item.name))]
}

async function forGroups<T>(groups: readonly string[], read: (group: string) => Promise<T[]>, signal: AbortSignal) {
	const results: { group: string; items: T[] }[] = []
	for (let i = 0; i < groups.length; i += 6) {
		signal.throwIfAborted()
		const batch = groups.slice(i, i + 6)
		results.push(...(await Promise.all(batch.map(async group => ({ group, items: await read(group) })))))
	}
	return results
}

export async function loadVirtualServices(signal: AbortSignal, groups?: string[]): Promise<VirtualServiceListItem[]> {
	const visible = groups ?? (await loadAccessGroups(signal))
	const batches = await forGroups(
		visible,
		async accessGroup => (await virtualServiceClient.listVirtualServices({ accessGroup }, { signal })).items,
		signal
	)
	return batches.flatMap(batch => batch.items)
}

export type SpecKind = 'templates' | 'listeners' | 'clusters' | 'routes' | 'http-filters' | 'policies' | 'access-logs'

export interface ResourceRow {
	uid: string
	name: string
	description: string
	raw: string
	groups: string[]
	type: string
}

type ResourceItem = { uid: string; name: string; description?: string; raw?: string; type?: ListenerType }

function row(item: ResourceItem): ResourceRow {
	return {
		uid: item.uid,
		name: item.name,
		description: item.description || '',
		raw: item.raw || '',
		type: item.type === undefined ? '' : ListenerType[item.type],
		groups: []
	}
}

export async function loadResources(kind: SpecKind, signal: AbortSignal): Promise<ResourceRow[]> {
	if (kind === 'clusters') return (await clusterClient.listCluster({}, { signal })).items.map(row)
	if (kind === 'policies') return (await policyClient.listPolicies({}, { signal })).items.map(row)
	const read = async (accessGroup: string): Promise<ResourceItem[]> => {
		const request = { accessGroup }
		const options = { signal }
		switch (kind) {
			case 'templates':
				return (await templateServiceClient.listVirtualServiceTemplates(request, options)).items
			case 'listeners':
				return (await listenerServiceClient.listListeners(request, options)).items
			case 'routes':
				return (await routeServiceClient.listRoutes(request, options)).items
			case 'http-filters':
				return (await httpFilterServiceClient.listHTTPFilters(request, options)).items
			case 'access-logs':
				return (await accessLogServiceClient.listAccessLogConfigs(request, options)).items
		}
	}
	const groups = await loadAccessGroups(signal)
	const available = await forGroups(groups, read, signal)
	const unique = new Map<string, ResourceRow>()
	for (const { group, items } of available) {
		for (const item of items) {
			const result = unique.get(item.uid) ?? row(item)
			result.groups.push(group)
			unique.set(item.uid, result)
		}
	}
	return Array.from(unique.values()).sort((a, b) => a.name.localeCompare(b.name))
}

export function publication(vs: VirtualServiceListItem) {
	if (!vs.status) return 'Unknown'
	if (vs.status.invalid) return 'Degraded'
	return vs.status.message ? 'Warning' : 'No failure reported'
}

export function appliesToAllNodes(nodes: readonly string[]) {
	return nodes.length === 1 && nodes[0] === '*'
}

export function sectionLink(section: string, filters: Record<string, string>) {
	return '/beta/' + section + '?' + new URLSearchParams(filters)
}
