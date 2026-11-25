import { Stack, Typography } from '@mui/material'
import client from '../../../../api/axiosApiClient'
import type { IClustersResponse } from '../../../../common/types/getClustersApiTypes'
import ConfigResourceTable, { Fields, ResourceSelection } from '../ConfigResourceTable'
import { socketAddress } from '../socketAddress'
import type { Column } from '../SectionTable'

const discovery: Record<number, string> = {
	0: 'STATIC',
	1: 'STRICT_DNS',
	2: 'LOGICAL_DNS',
	3: 'EDS',
	4: 'ORIGINAL_DST'
}
const balancing: Record<number, string> = {
	0: 'ROUND_ROBIN',
	1: 'LEAST_REQUEST',
	2: 'RING_HASH',
	3: 'RANDOM',
	5: 'MAGLEV',
	6: 'CLUSTER_PROVIDED',
	7: 'LOAD_BALANCING_POLICY_CONFIG'
}

async function loadClusters(node: string, signal: AbortSignal) {
	const response = await client.get<IClustersResponse>('/clusters', { params: { node_id: node }, signal })
	return (response.data.clusters ?? []).map(cluster => {
		const discoveryType = cluster.ClusterDiscoveryType as { Type?: number; ClusterType?: { name?: string } } | null
		const type = discoveryType?.Type ?? 0
		return {
			name: cluster.name,
			discovery: discoveryType?.ClusterType?.name || (discovery[type] ?? String(type)),
			balancing: balancing[cluster.lb_policy ?? 0] ?? String(cluster.lb_policy),
			transport: cluster.transport_socket?.name,
			endpoints: (cluster.load_assignment?.endpoints ?? []).flatMap(group =>
				(group.lb_endpoints ?? []).map(member => socketAddress(member.HostIdentifier?.Endpoint?.address))
			)
		}
	})
}

type ClusterRow = Awaited<ReturnType<typeof loadClusters>>[number]
const columns: Column<ClusterRow>[] = [
	{ key: 'name', label: 'Cluster', value: r => r.name },
	{ key: 'discovery', label: 'Discovery type', filterType: 'enum', value: r => r.discovery },
	{ key: 'balancing', label: 'Load balancing', filterType: 'enum', value: r => r.balancing },
	{
		key: 'endpoints',
		filterType: 'number',
		label: 'Configured endpoints',
		value: r => r.endpoints.length,
		align: 'right',
		render: row => <ResourceSelection name={row.name}>{row.endpoints.length}</ResourceSelection>
	}
]
export default function Clusters() {
	return (
		<ConfigResourceTable
			title='Clusters'
			load={loadClusters}
			columns={columns}
			detail={cluster => (
				<Stack spacing={1.5}>
					<Fields
						entries={[
							['Discovery type', cluster.discovery],
							['Load balancing', cluster.balancing],
							['Transport socket', cluster.transport || 'Default']
						]}
					/>
					<Typography variant='subtitle2'>Configured endpoints</Typography>
					{cluster.endpoints.length ? (
						cluster.endpoints.map((endpoint, i) => (
							<Typography key={i} component='code' variant='body2'>
								{endpoint}
							</Typography>
						))
					) : (
						<Typography variant='body2'>No endpoints in the cluster load assignment.</Typography>
					)}
				</Stack>
			)}
		/>
	)
}
