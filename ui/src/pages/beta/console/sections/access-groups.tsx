import { Link as RouterLink } from 'react-router-dom'
import { Link, Stack, Typography } from '@mui/material'
import { useQuery } from '@tanstack/react-query'
import type { VirtualServiceListItem } from '../../../../gen/virtual_service/v1/virtual_service_pb'
import SectionTable, { type Column } from '../SectionTable'
import { loadAccessGroups, loadVirtualServices, sectionLink } from '../resourceData'
import { manualQueryOptions, nodePath } from '../../api'
import { ResourceLink } from '../grpcResources'

interface Group {
	name: string
	services: VirtualServiceListItem[]
	nodes: string[]
}

async function readGroups(signal: AbortSignal): Promise<Group[]> {
	const names = await loadAccessGroups(signal)
	const services = await loadVirtualServices(signal, names)
	const byGroup = new Map(names.map(name => [name, [] as VirtualServiceListItem[]]))
	for (const vs of services) byGroup.get(vs.accessGroup)?.push(vs)
	return names.map(name => {
		const members = byGroup.get(name) || []
		return { name, services: members, nodes: [...new Set(members.flatMap(v => v.nodeIds))].sort() }
	})
}

const columns: Column<Group>[] = [
	{ key: 'name', label: 'Access group', value: r => r.name },
	{
		key: 'services',
		label: 'Virtual services',
		align: 'right',
		value: r => r.services.length,
		render: r => (
			<Link component={RouterLink} to={sectionLink('res-virtual-services', { by: 'group', value: r.name })}>
				{r.services.length}
			</Link>
		)
	},
	{
		key: 'node',
		label: 'Nodes',
		value: r => r.nodes,
		render: r => (
			<Stack direction='row' flexWrap='wrap' gap={0.75}>
				{r.nodes.map(id =>
					id === '*' ? (
						<span key={id}>All nodes</span>
					) : (
						<Link key={id} component={RouterLink} to={nodePath(id)}>
							{id}
						</Link>
					)
				)}
			</Stack>
		)
	}
]

export default function AccessGroups() {
	const query = useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: ['beta', 'access-group-services'],
		queryFn: ({ signal }) => readGroups(signal)
	})
	return (
		<SectionTable
			title='Access groups'
			description='Virtual services and nodes visible to your account.'
			rows={query.data || []}
			columns={columns}
			rowId={r => r.name}
			loading={query.isFetching}
			error={query.error}
			updatedAt={query.dataUpdatedAt}
			onRefresh={() => void query.refetch()}
			detail={r => (
				<Stack spacing={1}>
					<Typography variant='subtitle2'>Virtual services</Typography>
					{!r.services.length && <Typography variant='body2'>No visible virtual services.</Typography>}
					{r.services.map(v => (
						<ResourceLink key={v.uid} section='res-virtual-services' uid={v.uid} name={v.name} />
					))}
				</Stack>
			)}
		/>
	)
}
