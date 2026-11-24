type Json = Record<string, unknown>

export const object = (value: unknown): Json =>
	value && typeof value === 'object' && !Array.isArray(value) ? (value as Json) : {}

export function field(value: unknown, key: string): unknown {
	const obj = object(value)
	return obj[key] ?? obj[key.replace(/_([a-z])/g, (_, char: string) => char.toUpperCase())]
}

export const text = (value: unknown) => (typeof value === 'string' || typeof value === 'number' ? String(value) : '')
export function list(value: unknown, key: string): Json[] {
	const raw = field(value, key)
	return Array.isArray(raw) ? raw.map(object) : []
}

const strings = (value: unknown): string[] => (Array.isArray(value) ? value.filter(v => typeof v === 'string') : [])
const nameOf = (value: unknown) => text(field(value, 'name'))
const extension = (value: unknown) => text(field(value, '@type')).split('/').pop()
const unique = (values: string[]) => [...new Set(values.filter(Boolean))]

export function address(value: unknown) {
	const socket = field(value, 'socket_address')
	const host = text(field(socket, 'address'))
	const port = text(field(socket, 'port_value') ?? field(socket, 'named_port'))
	const label = host && port ? (host.includes(':') ? '[' + host + ']' : host) + ':' + port : host
	return {
		label: label || (port ? 'Port ' + port + ' (address not shown)' : text(field(field(value, 'pipe'), 'path'))),
		host,
		port,
		protocol: socket ? text(field(socket, 'protocol')) || 'TCP' : ''
	}
}

function routes(value: unknown) {
	return list(value, 'virtual_hosts').map(vhost => ({
		name: nameOf(vhost),
		domains: strings(field(vhost, 'domains')),
		routes: list(vhost, 'routes').map(entry => {
			const route = field(entry, 'route')
			const match = field(entry, 'match')
			const rules = ['path', 'prefix', 'path_separated_prefix', 'safe_regex', 'connect_matcher'] as const
			const rule = rules.find(key => field(match, key) !== undefined)
			const expression =
				rule === 'safe_regex' ? text(field(field(match, rule), 'regex')) : text(rule && field(match, rule))
			const targets = unique([
				text(field(route, 'cluster')),
				...list(field(route, 'weighted_clusters'), 'clusters').map(nameOf)
			])
			let action = targets.length ? 'Cluster' : 'Not reported'
			if (field(entry, 'redirect')) action = 'Redirect'
			else if (field(entry, 'direct_response')) action = 'Direct response'
			else if (field(route, 'cluster_header')) action = 'Cluster selected by request header'
			else if (field(route, 'cluster_specifier_plugin') || field(route, 'inline_cluster_specifier_plugin'))
				action = 'Cluster selected by plugin'
			return {
				match: rule ? [rule.replace(/_/g, ' '), expression].filter(Boolean).join(' ') : 'Match not reported',
				targets,
				action
			}
		})
	}))
}

export type DumpRoutes = ReturnType<typeof routes>
export { routes }

function hasExtension(wrapper: unknown, typeName: string, canonicalName: string) {
	const config = field(wrapper, 'typed_config')
	const type = extension(config)
	return type ? type === typeName : nameOf(wrapper) === canonicalName
}

function chain(value: Json, isDefault: boolean) {
	const match = field(value, 'filter_chain_match')
	const socket = field(value, 'transport_socket')
	const tls = hasExtension(
		socket,
		'envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext',
		'envoy.transport_sockets.tls'
	)
		? field(field(socket, 'typed_config'), 'common_tls_context')
		: undefined
	const network = list(value, 'filters')
	const managers = network.flatMap(filter => {
		const config = field(filter, 'typed_config')
		if (
			!hasExtension(
				filter,
				'envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager',
				'envoy.filters.network.http_connection_manager'
			)
		)
			return []
		const inline = field(config, 'route_config')
		return [
			{
				routeName: text(field(field(config, 'rds'), 'route_config_name')),
				inlineRoutes: inline ? routes(inline) : undefined,
				scoped: Boolean(field(config, 'scoped_routes')),
				httpFilters: list(config, 'http_filters').map(nameOf)
			}
		]
	})
	return {
		name: nameOf(value) || (isDefault ? 'Default filter chain' : 'Unnamed filter chain'),
		domains: strings(field(match, 'server_names')),
		transportSocket: nameOf(socket),
		secrets: unique(list(tls, 'tls_certificate_sds_secret_configs').map(nameOf)),
		networkFilters: network.map(nameOf),
		managers
	}
}

function moduleFile(filter: Json) {
	const config = field(filter, 'typed_config')
	if (extension(config) !== 'envoy.extensions.filters.listener.dynamic_modules.v3.DynamicModuleListenerFilter')
		return ''
	const local = field(field(field(config, 'dynamic_module_config'), 'module'), 'local')
	return safeModulePath(text(field(local, 'filename')))
}

function safeModulePath(value: string) {
	return /^\/[A-Za-z0-9._/-]{1,240}\.(?:so(?:\.\d+)*|dylib)$/.test(value) ? value : ''
}

export function listener(value: Json) {
	const addresses = [
		field(value, 'address'),
		...list(value, 'additional_addresses').map(item => field(item, 'address'))
	]
		.map(address)
		.filter(item => item.label)
	const filters = list(value, 'listener_filters')
	const defaultChain = field(value, 'default_filter_chain')
	return {
		addresses,
		filters: filters.map(nameOf),
		modules: filters.flatMap((filter, index) => {
			const path = moduleFile(filter)
			return path ? [{ index, name: nameOf(filter), path }] : []
		}),
		chains: [
			...list(value, 'filter_chains').map(item => chain(item, false)),
			...(defaultChain ? [chain(object(defaultChain), true)] : [])
		]
	}
}

// UpdateFailureState.details is unstructured and may include the configuration Envoy rejected.
// Render a recognized diagnostic and a validated module filename, never the original message.
export function rejection(value: unknown, modules: ReturnType<typeof listener>['modules'] = []) {
	const details = text(field(value, 'details'))
	if (!details) return 'Reason not reported'
	if (/validation (?:error|failed)|constraint validation/i.test(details)) return 'Configuration validation failed'
	const diagnostic = details.match(/failed to load dynamic module\b([^\r\n]*)/i)?.[1]
	if (diagnostic !== undefined && (modules.length || /^failed to load dynamic module\b/i.test(details.trim()))) {
		const filename = diagnostic.match(
			/^\s*(?:(?:at|from)\s+)?[:'"]?\s*(\/[A-Za-z0-9._/-]+\.(?:so(?:\.\d+)*|dylib))(?=[\s:,'")\]]|$)/i
		)?.[1]
		const path = modules.find(module => module.path === filename)?.path
		return (
			['Failed to load dynamic module', path].filter(Boolean).join(' ') +
			(/no such file/i.test(diagnostic) ? ': No such file' : '')
		)
	}
	if (/didn.t find a registered implementation|unknown filter/i.test(details))
		return 'Filter implementation not found'
	return 'Envoy reported a rejection. Unstructured error text is withheld.'
}

// Expose network host and port only. Opaque targets may contain credentials, query parameters or paths.
function grpcTarget(value: string) {
	const target = value.replace(/^dns:\/\/\//i, '').replace(/^https?:\/\//i, '')
	if (/^[a-z][a-z0-9+.-]*:\/\//i.test(target)) return ''
	const authority = target.split(/[/?#]/, 1)[0]
	const endpoint = authority.slice(authority.lastIndexOf('@') + 1)
	return /^(?:[A-Za-z0-9.-]+|\[[0-9a-f:]+\])(?::[0-9]{1,5})?$/i.test(endpoint) ? endpoint : ''
}

export function nodeInfo(value: unknown) {
	const node = field(value, 'node')
	const build = field(node, 'user_agent_build_version')
	const version = field(build, 'version')
	const semantic = version ? ['major_number', 'minor_number', 'patch'].map(k => Number(field(version, k) ?? 0)) : []
	const revision = text(field(field(build, 'metadata'), 'revision.sha'))
	const bootstrapApi = field(field(value, 'dynamic_resources'), 'ads_config')
	return {
		id: text(field(node, 'id')),
		cluster: text(field(node, 'cluster')),
		apiType: text(field(bootstrapApi, 'api_type')),
		servers: unique(
			list(bootstrapApi, 'grpc_services').map(service => {
				const cluster = text(field(field(service, 'envoy_grpc'), 'cluster_name'))
				if (cluster) return 'Envoy gRPC cluster ' + cluster
				const target = grpcTarget(text(field(field(service, 'google_grpc'), 'target_uri')))
				return target ? 'Google gRPC target ' + target : ''
			})
		),
		version:
			text(field(node, 'user_agent_version')) ||
			(semantic.length && semantic.every(Number.isSafeInteger) ? semantic.join('.') : ''),
		build: /^[0-9a-f]{7,64}$/i.test(revision) ? revision : '',
		transportApi: text(field(bootstrapApi, 'transport_api_version'))
	}
}
