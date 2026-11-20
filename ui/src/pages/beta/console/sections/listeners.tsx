import { Link, Stack, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import client from '../../../../api/axiosApiClient'
import type { Listener, IListenersResponse } from '../../../../common/types/getListenerDomainApiTypes'
import type { NodeOverviewResponse } from '../../../../common/types/overviewApiTypes'
import ConfigResourceTable, { Fields, ResourceSelection } from '../ConfigResourceTable'
import { socketAddress } from '../socketAddress'
import { nodeSection } from '../nodeInventory'
import type { Column } from '../SectionTable'

async function loadListeners(node: string, signal: AbortSignal) {
	const options = { params: { node_id: node }, signal }
	const [listeners, overview] = await Promise.all([
		client.get<IListenersResponse>('/listeners', options),
		client.get<NodeOverviewResponse>('/overview', options)
	])
	const routes = new Map<string, Set<string>>()
	for (const endpoint of overview.data.endpoints ?? []) {
		if (!endpoint.routeConfigName) continue
		const names = routes.get(endpoint.listenerName) ?? new Set<string>()
		names.add(endpoint.routeConfigName)
		routes.set(endpoint.listenerName, names)
	}
	return (listeners.data.listeners ?? []).map(
		(listener: Listener & { default_filter_chain?: Listener['filter_chains'][number] }) => {
			const chains = [
				...(listener.filter_chains ?? []),
				...(listener.default_filter_chain ? [listener.default_filter_chain] : [])
			]
			return {
				name: listener.name,
				address: socketAddress(listener.address),
				filters: (listener.listener_filters ?? []).map(f => f.name),
				chains: chains.map((chain, index) => ({
					name: chain.name || 'Filter chain ' + (index + 1),
					domains: chain.filter_chain_match?.server_names ?? [],
					filters: (chain.filters ?? []).map(f => f.name),
					transport: chain.transport_socket?.name
				})),
				routes: [...(routes.get(listener.name) ?? [])]
			}
		}
	)
}

type ListenerRow = Awaited<ReturnType<typeof loadListeners>>[number]
const columns: Column<ListenerRow>[] = [
	{ key: 'name', label: 'Listener', value: r => r.name },
	{ key: 'address', label: 'Address', value: r => r.address },
	{
		key: 'chains',
		label: 'Filter chains',
		align: 'right',
		value: r => r.chains.length,
		render: row => <ResourceSelection name={row.name}>{row.chains.length}</ResourceSelection>
	},
	{ key: 'domains', label: 'Server names', value: r => [...new Set(r.chains.flatMap(c => c.domains))] }
]
export default function Listeners() {
	return (
		<ConfigResourceTable
			title='Listeners'
			load={loadListeners}
			columns={columns}
			detail={(row, node) => (
				<Stack spacing={1.5}>
					<Fields
						entries={[
							['Address', row.address],
							['Listener filters', row.filters.join(', ') || 'None']
						]}
					/>
					{row.routes.map(route => (
						<Link
							key={route}
							component={RouterLink}
							to={nodeSection('routes', node) + '&selected=' + encodeURIComponent(route)}
						>
							{route}
						</Link>
					))}
					{row.chains.map((chain, index) => (
						<Stack key={index} spacing={0.5}>
							<Typography variant='subtitle2'>{chain.name}</Typography>
							<Fields
								entries={[
									['Server names', chain.domains.join(', ') || 'Any'],
									['Network filters', chain.filters.join(', ') || 'None'],
									['Transport socket', chain.transport || 'Default']
								]}
							/>
						</Stack>
					))}
				</Stack>
			)}
		/>
	)
}
