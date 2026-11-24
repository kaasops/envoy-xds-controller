import { Link, Stack, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import client from '../../../../api/axiosApiClient'
import type { IRouteConfigurationResponse, Route } from '../../../../common/types/getRouteConfigurationApiTypes'
import ConfigResourceTable, { Fields } from '../ConfigResourceTable'
import { nodeSection } from '../nodeInventory'
import type { Column } from '../SectionTable'

function describeRoute(route: Route) {
	const path = route.match?.PathSpecifier as
		| (Route['match']['PathSpecifier'] & { SafeRegex?: { regex?: string } })
		| undefined
	const cluster = route.Action?.Route?.ClusterSpecifier as
		| { Cluster?: string; ClusterHeader?: string; WeightedClusters?: { clusters?: { name?: string }[] } }
		| undefined
	const clusters = cluster?.Cluster
		? [cluster.Cluster]
		: (cluster?.WeightedClusters?.clusters ?? []).flatMap(c => (c.name ? [c.name] : []))
	const matcher =
		path?.Prefix !== undefined
			? 'Prefix ' + path.Prefix
			: path?.Path !== undefined
				? 'Path ' + path.Path
				: path?.SafeRegex
					? 'Regex ' + path.SafeRegex.regex
					: Object.keys(path ?? {})[0] || 'Unspecified match'
	const action = route.Action?.DirectResponse
		? 'Direct response ' + route.Action.DirectResponse.status
		: cluster?.ClusterHeader
			? 'Cluster from header ' + cluster.ClusterHeader
			: Object.keys(route.Action ?? {})[0] || 'Unspecified action'
	return { name: route.name, matcher, action, clusters }
}

async function loadRoutes(node: string, signal: AbortSignal) {
	const response = await client.get<IRouteConfigurationResponse>('/routeConfigurations', {
		params: { node_id: node },
		signal
	})
	return (response.data.routeConfigurations ?? []).map(config => ({
		name: config.name,
		hosts: (config.virtual_hosts ?? []).map(host => ({
			name: host.name,
			domains: host.domains ?? [],
			routes: (host.routes ?? []).map(describeRoute)
		}))
	}))
}

type RouteRow = Awaited<ReturnType<typeof loadRoutes>>[number]
const columns: Column<RouteRow>[] = [
	{ key: 'name', label: 'Route configuration', value: r => r.name },
	{ key: 'hosts', label: 'Virtual hosts', value: r => r.hosts.map(h => h.name) },
	{ key: 'domains', label: 'Domains', value: r => [...new Set(r.hosts.flatMap(h => h.domains))] },
	{
		key: 'routes',
		label: 'Routes',
		filterType: 'number',
		value: r => r.hosts.reduce((n, h) => n + h.routes.length, 0),
		align: 'right'
	}
]
export default function Routes() {
	return (
		<ConfigResourceTable
			title='Routes'
			columns={columns}
			load={loadRoutes}
			detail={(config, node) => (
				<Stack spacing={2}>
					{config.hosts.map((host, i) => (
						<Stack spacing={1} key={i}>
							<Typography variant='subtitle2'>Virtual host · {host.name}</Typography>
							<Fields entries={[['Domains', host.domains.join(', ') || 'None']]} />
							{host.routes.map((route, j) => (
								<Stack spacing={0.25} key={j}>
									<Typography variant='body2' fontWeight={600}>
										{route.name || 'Route ' + (j + 1)}
									</Typography>
									<Typography variant='body2'>{route.matcher}</Typography>
									{route.clusters.length ? (
										route.clusters.map(cluster => (
											<Link
												key={cluster}
												component={RouterLink}
												to={
													nodeSection('clusters', node) +
													'&selected=' +
													encodeURIComponent(cluster)
												}
											>
												{cluster}
											</Link>
										))
									) : (
										<Typography variant='body2'>{route.action}</Typography>
									)}
								</Stack>
							))}
						</Stack>
					))}
				</Stack>
			)}
		/>
	)
}
