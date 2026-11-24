import { useMemo, useState } from 'react'
import { Alert, Box, MenuItem, Paper, Stack, TextField, Typography } from '@mui/material'
import ConfigurationTable from '../nodeOverview/ConfigurationTable'
import { ResourceState } from './Summary'
import { displayName, type DumpReport, type DumpResource } from '../console/configDump'

function OrderedFilters({ label, filters }: { label: string; filters: string[] }) {
	return (
		<Box sx={{ minWidth: 0 }}>
			<Typography component='h3' variant='subtitle2'>
				{label}
			</Typography>
			{filters.length ? (
				<Box component='ol' sx={{ m: 0, pl: 2.5, overflowWrap: 'anywhere' }}>
					{filters.map((name, i) => (
						<li key={i}>{name || 'Name not reported'}</li>
					))}
				</Box>
			) : (
				<Typography variant='body2' color='text.secondary'>
					None shown
				</Typography>
			)}
		</Box>
	)
}

export default function DomainTrace({ report, initialListener }: { report: DumpReport; initialListener: string }) {
	const paths = useMemo(() => {
		const routes = new Map<string, DumpResource[]>()
		for (const row of report.rows) {
			if (row.type !== 'Route configuration' || row.state !== 'Dynamic') continue
			const sameName = routes.get(row.name) || []
			sameName.push(row)
			routes.set(row.name, sameName)
		}
		return report.rows.flatMap(
			row =>
				row.listener?.chains.flatMap((chain, index) =>
					(chain.managers.length ? chain.managers : [undefined]).map((manager, managerIndex) => {
						const matches = manager?.routeName ? routes.get(manager.routeName) || [] : []
						const hosts = manager?.inlineRoutes ?? (matches.length === 1 ? matches[0].routes : undefined)
						return {
							key: [row.id, index, managerIndex].join('/'),
							row,
							chain,
							manager,
							managerIndex,
							matches,
							hosts
						}
					})
				) || []
		)
	}, [report])
	const [key, setKey] = useState(() => paths.find(path => path.row.id === initialListener)?.key || '')
	const [domain, setDomain] = useState('')
	const domains = useMemo(
		() => [...new Set(paths.flatMap(path => path.hosts?.flatMap(host => host.domains) || []))].sort(),
		[paths]
	)
	const candidates = domain ? paths.filter(path => path.hosts?.some(host => host.domains.includes(domain))) : paths
	const selectedPath = candidates.find(path => path.key === key) || candidates[0]
	const selected = selectedPath?.row
	const info = selected?.listener
	const chain = selectedPath?.chain
	const manager = selectedPath?.manager
	const matches = selectedPath?.matches || []
	const hosts = selectedPath?.hosts?.filter(host => !domain || host.domains.includes(domain)) || []
	function states(type: DumpResource['type'], name: string) {
		const rows = report.rows.filter(row => row.type === type && row.name === name)
		return rows.length ? [...new Set(rows.map(row => row.state))].join(', ') : 'Not listed in file'
	}
	if (!selectedPath || !selected || !info || !chain)
		return <Alert severity='info'>No listener filter chain is available to trace in this file.</Alert>
	return (
		<Stack spacing={2}>
			<Paper variant='outlined' component='section' sx={{ p: 2 }}>
				<Typography component='h2' variant='subtitle1' fontWeight={650} mb={1.5}>
					Domain trace
				</Typography>
				<Stack direction='row' flexWrap='wrap' gap={1.5}>
					{domains.length > 0 && (
						<TextField
							select
							size='small'
							label='HTTP domain pattern'
							value={domain}
							onChange={e => setDomain(e.target.value)}
							sx={{ minWidth: '13em', maxWidth: '100%' }}
						>
							<MenuItem value=''>All domains</MenuItem>
							{domains.map(item => (
								<MenuItem key={item} value={item}>
									{item}
								</MenuItem>
							))}
						</TextField>
					)}
					<TextField
						select
						size='small'
						label='Listener path'
						value={selectedPath.key}
						onChange={e => setKey(e.target.value)}
						sx={{ minWidth: '15em', maxWidth: '100%' }}
					>
						{candidates.map(path => (
							<MenuItem key={path.key} value={path.key}>
								{displayName(path.row)} · {path.row.state} ·{' '}
								{path.row.details.Address || 'Address not reported'} · {path.chain.name}
								{path.chain.managers.length > 1 && ' · HTTP manager ' + (path.managerIndex + 1)}
							</MenuItem>
						))}
					</TextField>
				</Stack>
				<Box
					sx={{
						display: 'grid',
						gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr', lg: 'repeat(3,minmax(0,1fr))' },
						gap: 2,
						mt: 2,
						overflowWrap: 'anywhere'
					}}
				>
					{[
						[
							'Listener',
							<>
								<b>{displayName(selected)}</b> <ResourceState state={selected.state} />
							</>
						],
						[
							'Filter chain · SNI',
							chain.name + ' · ' + (chain.domains.join(', ') || 'No server name constraint shown')
						],
						[
							'TLS certificate · SDS',
							chain.secrets.map(name => name + ' (' + states('Secret', name) + ')').join(', ') ||
								'No SDS certificate reference shown'
						],
						[
							'Route configuration',
							manager?.inlineRoutes
								? 'Inline route configuration'
								: manager?.routeName
									? manager.routeName +
										(matches.length === 1
											? ' · RDS'
											: matches.length
												? ' · Ambiguous in file'
												: ' · Not listed in file')
									: manager?.scoped
										? 'Scoped routing'
										: 'No RDS or inline routes shown'
						],
						[
							'Virtual host · HTTP Host',
							hosts.map(host => (host.name || 'Unnamed') + ' · ' + host.domains.join(', ')).join('; ') ||
								'Not resolved from this file'
						]
					].map(([key, value]) => (
						<Box key={String(key)}>
							<Typography variant='caption' color='text.secondary'>
								{key}
							</Typography>
							<Typography component='div' variant='body2'>
								{value}
							</Typography>
						</Box>
					))}
				</Box>
				{selected.state === 'Rejected' && (
					<Alert severity='warning' sx={{ mt: 2 }}>
						This path belongs to the rejected update.
					</Alert>
				)}
			</Paper>
			<Paper component='section' variant='outlined' sx={{ overflow: 'hidden' }}>
				<Typography component='h2' variant='subtitle1' fontWeight={650} sx={{ p: 2 }}>
					Routes and upstream clusters
				</Typography>
				<Typography variant='body2' color='text.secondary' sx={{ px: 2, pb: 1 }}>
					Routes can also match on headers and other request attributes.
				</Typography>
				<ConfigurationTable
					label='Routes'
					rows={hosts.flatMap((host, h) =>
						host.routes.map((route, i) => ({ ...route, id: h + '/' + i, vhost: host.name }))
					)}
					rowId={row => row.id}
					columns={[
						{ label: 'Virtual host', value: r => r.vhost || 'Unnamed' },
						{ label: 'Path match', value: r => r.match },
						{ label: 'Action', value: r => r.action },
						{
							label: 'Upstream cluster',
							value: r =>
								r.targets.map(name => name + ' (' + states('Cluster', name) + ')').join(', ') ||
								'None named'
						}
					]}
				/>
			</Paper>
			<Paper component='section' variant='outlined' sx={{ p: 2 }}>
				<Typography component='h2' variant='subtitle1' fontWeight={650} mb={1.5}>
					Filter order
				</Typography>
				<Box
					sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: 'repeat(3,minmax(0,1fr))' }, gap: 2 }}
				>
					<OrderedFilters label='Listener filters' filters={info.filters} />
					<OrderedFilters label='Network filters' filters={chain.networkFilters} />
					<OrderedFilters label='HTTP filters' filters={manager?.httpFilters || []} />
				</Box>
			</Paper>
		</Stack>
	)
}
