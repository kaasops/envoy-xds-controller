type JsonObject = Record<string, unknown>
type ResourceType = 'Listener' | 'Route configuration' | 'Cluster' | 'Secret'

function object(value: unknown): JsonObject {
	return value && typeof value === 'object' && !Array.isArray(value) ? (value as JsonObject) : {}
}

function field(obj: JsonObject, key: string): unknown {
	return obj[key] ?? obj[key.replace(/_([a-z])/g, (_, letter: string) => letter.toUpperCase())]
}

function items(obj: JsonObject, key: string): JsonObject[] {
	const value = field(obj, key)
	return Array.isArray(value) ? value.map(object) : []
}

function text(value: unknown): string {
	return typeof value === 'string' || typeof value === 'number' ? String(value) : ''
}

export interface DumpResource {
	name: string
	type: ResourceType
	state: string
	version: string
	updated: string
	details: Record<string, string | number>
	components: { label: string; value: string }[]
}

// Project only the fields displayed by the console. Keep credentials, arbitrary Any payloads and metadata out of state.
function resource(
	type: ResourceType,
	state: string,
	wrapper: JsonObject,
	config: JsonObject,
	fallbackName?: unknown
): DumpResource {
	const details: DumpResource['details'] = {}
	const components: DumpResource['components'] = []
	function add(label: string, names: string[]) {
		for (const value of new Set(names.filter(Boolean))) components.push({ label, value })
	}
	if (type === 'Listener' && Object.keys(config).length > 0) {
		const address = object(config.address)
		const socket = object(field(address, 'socket_address'))
		details.Address = text(socket.address) || text(object(address.pipe).path)
		details.Port = text(field(socket, 'port_value'))
		const chains = items(config, 'filter_chains')
		details['Filter chains'] = chains.length
		add(
			'Listener filter',
			items(config, 'listener_filters').map(filter => text(filter.name))
		)
		for (const chain of chains) {
			add('Filter chain', [text(chain.name)])
			add(
				'Network filter',
				items(chain, 'filters').map(filter => text(filter.name))
			)
			add('Transport socket', [text(object(field(chain, 'transport_socket')).name)])
		}
	} else if (type === 'Route configuration') {
		const hosts = items(config, 'virtual_hosts')
		details['Virtual hosts'] = hosts.length
		for (const host of hosts) {
			add('Virtual host', [text(host.name)])
			if (Array.isArray(host.domains)) add('Domain', host.domains.map(text))
			add(
				'Upstream cluster',
				items(host, 'routes').flatMap(route => {
					const action = object(route.route)
					return [
						text(action.cluster),
						...items(object(field(action, 'weighted_clusters')), 'clusters').map(cluster =>
							text(cluster.name)
						)
					]
				})
			)
		}
		details.Routes = hosts.reduce((sum, host) => sum + items(host, 'routes').length, 0)
	} else if (type === 'Cluster') {
		details['Discovery type'] = text(config.type) || text(object(field(config, 'cluster_type')).name)
		details['Load balancing'] = text(field(config, 'lb_policy'))
		const endpoints = items(object(field(config, 'load_assignment')), 'endpoints').flatMap(locality =>
			items(locality, 'lb_endpoints')
		)
		details['Configured endpoints'] = endpoints.length
		add(
			'Endpoint',
			endpoints.map(item => {
				const address = object(object(item.endpoint).address)
				const socket = object(field(address, 'socket_address'))
				const host = text(socket.address)
				const port = text(field(socket, 'port_value'))
				return host && port
					? (host.includes(':') ? '[' + host + ']' : host) + ':' + port
					: host || text(object(address.pipe).path)
			})
		)
	}
	return {
		type,
		state,
		name: text(config.name) || text(fallbackName) || 'Unnamed',
		version: text(field(wrapper, 'version_info')),
		updated: text(field(wrapper, 'last_updated') ?? field(wrapper, 'last_update_attempt')),
		details,
		components
	}
}

const groups: Record<string, { key: string; type: ResourceType; field: string; state: string }[]> = {
	ListenersConfigDump: [{ key: 'static_listeners', type: 'Listener', field: 'listener', state: 'Static' }],
	RoutesConfigDump: [
		{ key: 'static_route_configs', type: 'Route configuration', field: 'route_config', state: 'Static' },
		{ key: 'dynamic_route_configs', type: 'Route configuration', field: 'route_config', state: 'Dynamic' }
	],
	ClustersConfigDump: [
		{ key: 'static_clusters', type: 'Cluster', field: 'cluster', state: 'Static' },
		{ key: 'dynamic_active_clusters', type: 'Cluster', field: 'cluster', state: 'Active' },
		{ key: 'dynamic_warming_clusters', type: 'Cluster', field: 'cluster', state: 'Warming' }
	],
	SecretsConfigDump: [
		{ key: 'static_secrets', type: 'Secret', field: 'secret', state: 'Static' },
		{ key: 'dynamic_active_secrets', type: 'Secret', field: 'secret', state: 'Active' },
		{ key: 'dynamic_warming_secrets', type: 'Secret', field: 'secret', state: 'Warming' }
	]
}

export function readConfigDump(raw: unknown) {
	const { configs } = object(raw)
	if (!Array.isArray(configs)) throw new Error('Expected an Envoy config dump with a configs array.')
	const rows: DumpResource[] = []
	let nodeId = ''
	let skipped = 0
	for (const item of configs) {
		const section = object(item)
		const kind = text(section['@type']).split('.').pop() || ''
		if (kind === 'BootstrapConfigDump') {
			nodeId = text(object(object(section.bootstrap).node).id)
			continue
		}
		if (!Object.prototype.hasOwnProperty.call(groups, kind)) {
			skipped++
			continue
		}
		for (const group of groups[kind]) {
			for (const entry of items(section, group.key)) {
				rows.push(resource(group.type, group.state, entry, object(field(entry, group.field)), entry.name))
			}
		}
		if (kind === 'ListenersConfigDump') {
			for (const entry of items(section, 'dynamic_listeners')) {
				for (const [state, key] of Object.entries({
					Active: 'active_state',
					Warming: 'warming_state',
					Draining: 'draining_state',
					Rejected: 'error_state'
				})) {
					const value = field(entry, key)
					if (value) {
						const wrapper = object(value)
						rows.push(resource('Listener', state, wrapper, object(wrapper.listener), entry.name))
					}
				}
			}
		}
	}
	return { nodeId, skipped, rows: rows.map((row, index) => ({ ...row, id: String(index) })) }
}
