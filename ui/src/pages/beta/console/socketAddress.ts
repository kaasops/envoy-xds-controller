export function socketAddress(address?: {
	Address?: {
		SocketAddress?: { address?: string; PortSpecifier?: { PortValue?: number; NamedPort?: string } }
		Pipe?: { path?: string }
		EnvoyInternalAddress?: unknown
	}
}) {
	const value = address?.Address
	const socket = value?.SocketAddress
	if (!socket) return value?.Pipe?.path || (value?.EnvoyInternalAddress ? 'Internal address' : 'Unavailable')
	const port = socket.PortSpecifier?.PortValue ?? socket.PortSpecifier?.NamedPort
	const host = socket.address || ''
	if (port === undefined || port === '') return host || 'Unavailable'
	return (host.includes(':') ? '[' + host + ']' : host) + ':' + port
}
