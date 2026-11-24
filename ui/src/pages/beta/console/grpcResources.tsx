import { useQuery } from '@tanstack/react-query'
import { Link as RouterLink } from 'react-router-dom'
import { Box, Link, Stack, Typography } from '@mui/material'
import SectionTable, { type Column } from './SectionTable'
import { manualQueryOptions } from '../api'
import { loadResources, sectionLink, type ResourceRow, type SpecKind } from './resourceData'

const titles: Record<SpecKind, string> = {
	templates: 'Templates',
	listeners: 'Listeners',
	routes: 'Routes',
	clusters: 'Clusters',
	'http-filters': 'HTTP filters',
	policies: 'Policies',
	'access-logs': 'Access log configs'
}

export function ResourceLink({ section, uid, name }: { section: string; uid: string; name: string }) {
	return (
		<Link component={RouterLink} to={sectionLink(section, { selected: uid })}>
			{name}
		</Link>
	)
}

function object(value: unknown): Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value)
		? (value as Record<string, unknown>)
		: {}
}

function text(value: unknown) {
	return typeof value === 'string' || typeof value === 'number' ? String(value) : ''
}
function records(value: unknown) {
	return (Array.isArray(value) ? value : [value]).map(object)
}

function summarize(kind: SpecKind, spec: unknown): string[] {
	const fields = object(spec)
	switch (kind) {
		case 'templates':
			return records(fields.listener)
				.map(o => text(o.name))
				.filter(Boolean)
		case 'listeners': {
			const address = object(fields.address)
			const socket = object(address.socket_address ?? address.socketAddress)
			const host = text(socket.address)
			if (!host) return [text(object(address.pipe).path)].filter(Boolean)
			const port = text(socket.port_value ?? socket.portValue ?? socket.named_port ?? socket.namedPort)
			return [port ? (host.includes(':') ? '[' + host + ']' : host) + ':' + port : host]
		}
		case 'routes':
			return records(spec).flatMap(o => {
				const route = object(o.route)
				const weights = object(route.weighted_clusters ?? route.weightedClusters)
				return [text(route.cluster), ...records(weights.clusters).map(c => text(c.name))].filter(Boolean)
			})
		case 'http-filters':
		case 'access-logs':
			return records(spec)
				.map(o => text(o.name))
				.filter(Boolean)
		default:
			return []
	}
}

function parseSpec(raw: string): unknown {
	if (!raw) return undefined
	try {
		return JSON.parse(raw)
	} catch {
		return undefined
	}
}

export function ConfigView({ raw }: { raw: string }) {
	const spec = parseSpec(raw)
	return (
		<Box
			component='pre'
			tabIndex={0}
			aria-label='Configuration'
			sx={{
				m: 0,
				p: 1.5,
				bgcolor: 'action.hover',
				borderRadius: 1,
				overflow: 'auto',
				fontSize: '0.8125rem',
				maxHeight: '60vh'
			}}
		>
			{spec === undefined ? raw : JSON.stringify(spec, null, 2)}
		</Box>
	)
}

const summaryLabel: Partial<Record<SpecKind, string>> = {
	templates: 'Listener',
	listeners: 'Socket address',
	routes: 'Upstream clusters',
	'http-filters': 'Filters',
	'access-logs': 'Access logs'
}

export default function ResourceSection({ kind }: { kind: SpecKind }) {
	const title = titles[kind]
	const query = useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: ['beta', 'resources', kind],
		queryFn: async ({ signal }) =>
			(await loadResources(kind, signal)).map(row => ({ ...row, summary: summarize(kind, parseSpec(row.raw)) }))
	})
	type Row = ResourceRow & { summary: string[] }
	const grouped = kind !== 'clusters' && kind !== 'policies'
	const columns: Column<Row>[] = [{ key: 'name', label: 'Name', value: r => r.name }]
	if (grouped)
		columns.push({
			key: 'groups',
			label: 'Visible in groups',
			value: r => r.groups,
			render: r => (
				<Stack direction='row' flexWrap='wrap' gap={0.75}>
					{r.groups.map(g => (
						<ResourceLink key={g} section='access-groups' uid={g} name={g} />
					))}
				</Stack>
			)
		})
	if (kind === 'listeners') columns.push({ key: 'type', label: 'Type', value: r => r.type })
	if (summaryLabel[kind]) columns.push({ key: 'summary', label: summaryLabel[kind]!, value: r => r.summary })
	if (kind !== 'clusters') columns.push({ key: 'description', label: 'Description', value: r => r.description })
	else columns.push({ key: 'uid', identifier: true, label: 'UID', value: r => r.uid })
	return (
		<SectionTable
			title={title}
			tableHeading='Configured resources'
			rows={query.data || []}
			columns={columns}
			rowId={r => r.uid}
			loading={query.isFetching}
			error={query.error}
			onRefresh={() => void query.refetch()}
			updatedAt={query.dataUpdatedAt}
			detail={r => (
				<Stack spacing={1.5}>
					<Typography variant='body2'>
						UID <code>{r.uid}</code>
					</Typography>
					{r.description && <Typography variant='body2'>{r.description}</Typography>}
					{r.raw && <ConfigView raw={r.raw} />}
					{kind === 'templates' && (
						<Link
							component={RouterLink}
							to={sectionLink('res-virtual-services', { by: 'template', value: r.uid })}
						>
							Virtual services using this template
						</Link>
					)}
				</Stack>
			)}
		/>
	)
}
