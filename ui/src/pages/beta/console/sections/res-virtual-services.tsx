import { Alert, Chip, LinearProgress, Link, Stack, Tooltip, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { virtualServiceClient } from '../../../../api/grpc/client'
import { env } from '../../../../env'
import { usePermissionsStore } from '../../../../store/permissionsStore'
import { PermissionAction } from '../../../../utils/helpers/permissionsActions'
import type { VirtualServiceListItem } from '../../../../gen/virtual_service/v1/virtual_service_pb'
import { apiError, manualQueryOptions, nodePath } from '../../api'
import SectionTable, { type Column } from '../SectionTable'
import InventorySummary from '../InventorySummary'
import { appliesToAllNodes, loadVirtualServices, serviceStatus, virtualServicesKey } from '../resourceData'
import { ConfigView, ResourceLink } from '../grpcResources'

function configurationWarnings(raw: string) {
	if (!raw) return []
	try {
		const spec = JSON.parse(raw)
		if (!spec || typeof spec !== 'object') return []
		return ['accessLog', 'accessLogConfig'].filter(key => spec[key] != null)
	} catch {
		return []
	}
}

function ServiceDetail({ row }: { row: VirtualServiceListItem }) {
	const canReadGroup = usePermissionsStore(state =>
		state
			.getGroupPermissions(row.accessGroup)
			?.some(
				permission =>
					permission.action === PermissionAction.GetVirtualService && permission.objects.includes('*')
			)
	)
	const enabled = env.VITE_OIDC_ENABLED !== 'true' || Boolean(canReadGroup)
	const query = useQuery({
		...manualQueryOptions,
		queryKey: ['beta', 'virtual-service', row.uid],
		enabled,
		gcTime: 0,
		queryFn: ({ signal }) => virtualServiceClient.getVirtualService({ uid: row.uid }, { signal })
	})
	const vs = query.isSuccess && enabled ? query.data : undefined
	const references = vs && [
		...(vs.listener ? [{ section: 'res-listeners', ref: vs.listener }] : []),
		...vs.additionalRoutes.map(ref => ({ section: 'res-routes', ref })),
		...vs.additionalHttpFilters.map(ref => ({ section: 'res-http-filters', ref })),
		...(vs.accessLog.case === 'accessLogConfigs'
			? vs.accessLog.value.refs.map(ref => ({ section: 'res-access-logs', ref }))
			: [])
	]
	return (
		<Stack spacing={1.5}>
			<Typography variant='body2'>
				UID <code>{row.uid}</code>
			</Typography>
			{row.description && <Typography variant='body2'>{row.description}</Typography>}
			<Typography variant='subtitle2'>Status: {serviceStatus(row)}</Typography>
			{row.status?.message && (
				<Alert severity={row.status.invalid ? 'error' : 'warning'}>{row.status.message}</Alert>
			)}
			{row.template && (
				<Typography variant='body2'>
					Template <ResourceLink section='res-templates' uid={row.template.uid} name={row.template.name} />
				</Typography>
			)}
			{!enabled && (
				<Typography variant='body2'>
					The list response does not report whether you can read this manifest.
				</Typography>
			)}
			{query.isFetching && <LinearProgress aria-label='Loading virtual service' />}
			{query.error && <Alert severity='error'>{apiError(query.error)}</Alert>}
			{vs && (
				<>
					{vs.virtualHost?.domains.length ? (
						<Typography variant='body2'>Domains: {vs.virtualHost.domains.join(', ')}</Typography>
					) : null}
					{references?.length ? (
						<Stack gap={0.75}>
							<Typography variant='subtitle2'>Referenced resources</Typography>
							{references.map(({ section, ref }) => (
								<ResourceLink key={section + ref.uid} section={section} uid={ref.uid} name={ref.name} />
							))}
						</Stack>
					) : null}
					{configurationWarnings(vs.raw).map(key => (
						<Alert key={key} severity='warning'>
							Configuration: <code>{key}</code> is deprecated. Use <code>{key}s</code>.
						</Alert>
					))}
					{vs.raw && <ConfigView raw={vs.raw} />}
				</>
			)}
		</Stack>
	)
}

const columns: Column<VirtualServiceListItem>[] = [
	{ key: 'name', label: 'Name', value: r => r.name },
	{
		key: 'group',
		label: 'Access group',
		value: r => r.accessGroup,
		render: r => <ResourceLink section='access-groups' uid={r.accessGroup} name={r.accessGroup} />
	},
	{
		key: 'node',
		identifier: true,
		label: 'Nodes',
		value: r => r.nodeIds,
		appliesToAllNodes: r => appliesToAllNodes(r.nodeIds),
		render: r => (
			<Stack direction='row' flexWrap='wrap' gap={0.75}>
				{r.nodeIds.map(id =>
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
	},
	{
		key: 'template',
		label: 'Template',
		value: r => (r.template ? [r.template.name, r.template.uid] : []),
		render: r => r.template && <ResourceLink section='res-templates' uid={r.template.uid} name={r.template.name} />
	},
	{
		key: 'status',
		filterType: 'enum',
		filterValues: ['Accepted', 'Accepted with warnings', 'Rejected', 'Unknown'],
		label: 'Status',
		value: serviceStatus,
		render: r => (
			<Tooltip title={r.status?.message || ''} describeChild>
				<Chip
					data-status
					tabIndex={r.status?.message ? 0 : undefined}
					size='small'
					variant='outlined'
					label={serviceStatus(r)}
					color={
						r.status?.invalid ? 'error' : r.status?.message ? 'warning' : r.status ? 'success' : 'default'
					}
				/>
			</Tooltip>
		)
	}
]

export default function VirtualServices() {
	const query = useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: virtualServicesKey,
		queryFn: ({ signal }) => loadVirtualServices(signal)
	})
	return (
		<SectionTable
			title='Virtual services'
			tableHeading='Virtual service status'
			summary={
				query.data && (
					<InventorySummary
						fields={[
							{ label: 'Visible services', value: query.data.length },
							{
								label: 'Rejected',
								value: query.data.filter(service => serviceStatus(service) === 'Rejected').length
							},
							{
								label: 'Accepted with warnings',
								value: query.data.filter(service => serviceStatus(service) === 'Accepted with warnings')
									.length
							}
						]}
					/>
				)
			}
			rows={query.data || []}
			columns={columns}
			rowId={r => r.uid}
			loading={query.isFetching}
			error={query.error}
			updatedAt={query.dataUpdatedAt}
			onRefresh={() => void query.refetch()}
			detail={r => <ServiceDetail row={r} />}
		/>
	)
}
