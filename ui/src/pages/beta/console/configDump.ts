import {
	address,
	field,
	list,
	listener,
	nodeInfo,
	object,
	rejection,
	routes,
	text,
	type DumpRoutes
} from './dumpProjection'

const resourceTypes = {
	Listener: 'envoy.config.listener.v3.Listener',
	'Route configuration': 'envoy.config.route.v3.RouteConfiguration',
	Cluster: 'envoy.config.cluster.v3.Cluster',
	Secret: 'envoy.extensions.transport_sockets.tls.v3.Secret',
	Endpoint: 'envoy.config.endpoint.v3.ClusterLoadAssignment',
	'Scoped route': 'envoy.config.route.v3.ScopedRouteConfiguration',
	Extension: 'envoy.config.core.v3.TypedExtensionConfig'
}
export type ResourceType = keyof typeof resourceTypes
export const dumpTypeUrl = (type: ResourceType) => 'type.googleapis.com/' + resourceTypes[type]

export interface DumpResource {
	id: string
	name: string
	type: ResourceType
	state: string
	version: string
	updated: string
	details: Record<string, string | number>
	components: { label: string; value: string }[]
	listener?: ReturnType<typeof listener>
	routes?: DumpRoutes
	reason?: string
}

interface Group {
	key: string
	wrapper: string
	type: ResourceType
	config: string
	state: string
}
const groups: Record<string, Group[]> = {
	ListenersConfigDump: [
		{ key: 'static_listeners', wrapper: 'StaticListener', type: 'Listener', config: 'listener', state: 'Static' }
	],
	RoutesConfigDump: [
		{
			key: 'static_route_configs',
			wrapper: 'StaticRouteConfig',
			type: 'Route configuration',
			config: 'route_config',
			state: 'Static'
		},
		{
			key: 'dynamic_route_configs',
			wrapper: 'DynamicRouteConfig',
			type: 'Route configuration',
			config: 'route_config',
			state: 'Dynamic'
		}
	],
	ClustersConfigDump: [
		{ key: 'static_clusters', wrapper: 'StaticCluster', type: 'Cluster', config: 'cluster', state: 'Static' },
		{
			key: 'dynamic_active_clusters',
			wrapper: 'DynamicCluster',
			type: 'Cluster',
			config: 'cluster',
			state: 'Active'
		},
		{
			key: 'dynamic_warming_clusters',
			wrapper: 'DynamicCluster',
			type: 'Cluster',
			config: 'cluster',
			state: 'Warming'
		}
	],
	SecretsConfigDump: [
		{ key: 'static_secrets', wrapper: 'StaticSecret', type: 'Secret', config: 'secret', state: 'Static' },
		{ key: 'dynamic_active_secrets', wrapper: 'DynamicSecret', type: 'Secret', config: 'secret', state: 'Active' },
		{ key: 'dynamic_warming_secrets', wrapper: 'DynamicSecret', type: 'Secret', config: 'secret', state: 'Warming' }
	],
	EndpointsConfigDump: [
		{
			key: 'static_endpoint_configs',
			wrapper: 'StaticEndpointConfig',
			type: 'Endpoint',
			config: 'endpoint_config',
			state: 'Static'
		},
		{
			key: 'dynamic_endpoint_configs',
			wrapper: 'DynamicEndpointConfig',
			type: 'Endpoint',
			config: 'endpoint_config',
			state: 'Dynamic'
		}
	],
	ScopedRoutesConfigDump: [
		{
			key: 'inline_scoped_route_configs',
			wrapper: 'InlineScopedRouteConfigs',
			type: 'Scoped route',
			config: 'scoped_route_configs',
			state: 'Inline'
		},
		{
			key: 'dynamic_scoped_route_configs',
			wrapper: 'DynamicScopedRouteConfigs',
			type: 'Scoped route',
			config: 'scoped_route_configs',
			state: 'Dynamic'
		}
	],
	EcdsConfigDump: [
		{ key: 'ecds_filters', wrapper: 'EcdsFilterConfig', type: 'Extension', config: 'ecds_filter', state: 'Dynamic' }
	]
}

export function readConfigDump(raw: unknown) {
	const configs = field(raw, 'configs')
	if (!Array.isArray(configs)) throw new Error('Expected a configs array.')
	const rows: DumpResource[] = []
	const sections = new Map<string, { name: string; count: number; projected: boolean; filtered: boolean }>()
	const versions = new Map<ResourceType, string>()
	const apis = new Set<string>()
	let node = nodeInfo({})
	let filtered = false

	function add(type: ResourceType, state: string, wrapper: unknown, value: unknown, fallback?: unknown) {
		const config = object(value)
		const name = text(field(config, type === 'Endpoint' ? 'cluster_name' : 'name')) || text(fallback)
		const row: DumpResource = {
			id: String(rows.length),
			type,
			name,
			state,
			version: text(field(wrapper, 'version_info')),
			updated: text(field(wrapper, 'last_updated') ?? field(wrapper, 'last_update_attempt')),
			details: {},
			components: []
		}
		if (type === 'Listener') {
			row.listener = listener(config)
			row.details.Address = row.listener.addresses.map(item => item.label).join(', ')
			row.details['Filter chains'] = row.listener.chains.length
			row.components.push(
				...row.listener.filters.map((value, index) => ({ label: 'Listener filter ' + (index + 1), value }))
			)
			for (const chain of row.listener.chains) {
				row.components.push({ label: 'Filter chain', value: chain.name })
				if (chain.transportSocket)
					row.components.push({ label: 'Transport socket', value: chain.transportSocket })
				row.components.push(
					...chain.networkFilters.map((value, index) => ({ label: 'Network filter ' + (index + 1), value }))
				)
			}
		} else if (type === 'Route configuration') {
			row.routes = routes(config)
			row.details['Virtual hosts'] = row.routes.length
			row.details.Routes = row.routes.reduce((count, host) => count + host.routes.length, 0)
			for (const host of row.routes) {
				row.components.push(
					{ label: 'Virtual host', value: host.name },
					...host.domains.map(value => ({ label: 'Domain', value }))
				)
				for (const route of host.routes)
					row.components.push(...route.targets.map(value => ({ label: 'Upstream cluster', value })))
			}
		} else if (type === 'Cluster' || type === 'Endpoint') {
			if (type === 'Cluster') {
				row.details['Discovery type'] =
					text(field(config, 'type')) || text(field(field(config, 'cluster_type'), 'name'))
				row.details['Load balancing'] = text(field(config, 'lb_policy'))
			}
			const assignment = type === 'Cluster' ? field(config, 'load_assignment') : config
			const endpoints = list(assignment, 'endpoints').flatMap(item => list(item, 'lb_endpoints'))
			row.details['Configured endpoints'] = endpoints.length
			row.components.push(
				...endpoints
					.map(entry => ({
						label: 'Endpoint',
						value: address(field(field(entry, 'endpoint'), 'address')).label
					}))
					.filter(item => item.value)
			)
		}
		if (state === 'Rejected') row.reason = rejection(wrapper, row.listener?.modules)
		rows.push(row)
	}

	function groupEntry(group: Group, entry: unknown, state = group.state) {
		const config = field(entry, group.config)
		if (group.type === 'Scoped route' && Array.isArray(config)) {
			for (const scope of config) add(group.type, state, entry, scope)
		} else add(group.type, state, entry, config, field(entry, 'name'))
		const error = field(entry, 'error_state')
		if (error)
			add(
				group.type,
				'Rejected',
				error,
				field(error, 'failed_configuration'),
				field(entry, 'name') ?? field(config, 'name') ?? field(config, 'cluster_name')
			)
	}
	function dynamicListener(entry: unknown) {
		const before = rows.length
		for (const [key, state] of [
			['active_state', 'Active'],
			['warming_state', 'Warming'],
			['draining_state', 'Draining'],
			['error_state', 'Rejected']
		]) {
			const wrapper = field(entry, key)
			if (wrapper)
				add(
					'Listener',
					state,
					wrapper,
					field(wrapper, state === 'Rejected' ? 'failed_configuration' : 'listener'),
					field(entry, 'name')
				)
		}
		if (rows.length === before) add('Listener', 'State not reported', entry, undefined, field(entry, 'name'))
	}
	for (const section of configs) {
		const typeName = text(field(section, '@type')).split('/').pop() || ''
		const match = /^envoy\.admin\.(v\d+(?:alpha\d*)?)\.(\w+)(?:\.(\w+))?$/.exec(typeName)
		const kind = match?.[2] || typeName || 'Unknown section'
		const member = match?.[3] || ''
		const known =
			match?.[1] === 'v3' &&
			(kind === 'BootstrapConfigDump' || Object.prototype.hasOwnProperty.call(groups, kind))
		const before = rows.length
		if (match) apis.add('envoy.admin.' + match[1])
		let projected = Boolean(known)
		if (known && kind === 'BootstrapConfigDump' && !member) node = nodeInfo(field(section, 'bootstrap'))
		else if (known && member) {
			filtered = true
			if (kind === 'ListenersConfigDump' && member === 'DynamicListener') dynamicListener(section)
			else {
				const variants = groups[kind]?.filter(group => group.wrapper === member) || []
				if (!variants.length) projected = false
				else groupEntry(variants[0], section, variants.length === 1 ? variants[0].state : 'State not reported')
			}
		} else if (known && groups[kind]) {
			const version = text(field(section, 'version_info'))
			if (version) versions.set(groups[kind][0].type, version)
			for (const group of groups[kind]) for (const entry of list(section, group.key)) groupEntry(group, entry)
			if (kind === 'ListenersConfigDump')
				for (const entry of list(section, 'dynamic_listeners')) dynamicListener(entry)
		}
		const item = sections.get(kind)
		sections.set(kind, {
			name: kind,
			count: (item?.count || 0) + rows.length - before,
			projected: (item?.projected ?? true) && projected,
			filtered: Boolean(member || item?.filtered)
		})
	}
	return { node, rows, sections: [...sections.values()], versions, apis: [...apis], filtered }
}
export type DumpReport = ReturnType<typeof readConfigDump>

export const displayName = (row: DumpResource) => row.name || 'Name not reported'
