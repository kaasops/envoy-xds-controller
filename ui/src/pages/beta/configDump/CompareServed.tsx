import { ConsoleTime } from '../time/ConsoleTime'
import { useEffect, useRef, useState } from 'react'
import { Alert, Box, Button, Paper, Stack, Typography } from '@mui/material'
import { apiError, getSnapshotDiff, httpStatus, type SnapshotDiff } from '../api'
import { dumpTypeUrl, type DumpReport, type ResourceType } from '../console/configDump'
import ConfigurationTable from '../nodeOverview/ConfigurationTable'

type Compared = { id: string; name: string; type: string; states: string[]; served?: string; listed: boolean }
const labels: Record<string, string> = Object.fromEntries(
	(
		[
			'Listener',
			'Cluster',
			'Route configuration',
			'Secret',
			'Endpoint',
			'Extension',
			'Scoped route'
		] as ResourceType[]
	).map(type => [dumpTypeUrl(type), type])
)

function compare(report: DumpReport, snapshot: SnapshotDiff) {
	const result = new Map<string, Compared>()
	for (const row of report.rows) {
		if (!row.name) continue
		const url = dumpTypeUrl(row.type)
		const id = JSON.stringify([url, row.name])
		const found = result.get(id)
		if (found) {
			if (!found.states.includes(row.state)) found.states.push(row.state)
		} else result.set(id, { id, name: row.name, type: row.type, states: [row.state], listed: false })
	}
	for (const entry of snapshot.resources) {
		if (entry.change === 'removed') continue
		const id = JSON.stringify([entry.typeUrl, entry.name])
		const row = result.get(id) || {
			id,
			name: entry.name,
			type: labels[entry.typeUrl] || entry.typeUrl,
			states: [],
			listed: false
		}
		row.listed = true
		row.served = entry.toSha256
		result.set(id, row)
	}
	return [...result.values()]
}

export default function CompareServed({ report, modified }: { report: DumpReport; modified: number }) {
	const [snapshot, setSnapshot] = useState<SnapshotDiff>()
	const [error, setError] = useState('')
	const [loading, setLoading] = useState(false)
	const pending = useRef<AbortController>()
	useEffect(() => () => pending.current?.abort(), [])
	const node = report.node.id
	const validNode = node && node !== '.' && node !== '..'
	async function load() {
		pending.current?.abort()
		const controller = new AbortController()
		pending.current = controller
		setLoading(true)
		setError('')
		setSnapshot(undefined)
		try {
			const result = await getSnapshotDiff(node, 'served', 'served', controller.signal)
			if (!controller.signal.aborted) setSnapshot(result)
		} catch (e) {
			if (!controller.signal.aborted) {
				const status = httpStatus(e)
				setError(
					status === 403 || status === 401
						? 'Access to the served snapshot was denied.'
						: status === 404
							? 'No served snapshot is available for this node.'
							: 'Could not load the served snapshot. ' + apiError(e)
				)
			}
		} finally {
			if (!controller.signal.aborted) setLoading(false)
		}
	}
	const published = snapshot?.to.publishedAt
	const newer = published && modified > 0 && Date.parse(published) > modified
	const versions = snapshot
		? [...new Set([...Object.keys(snapshot.to.versions || {}), ...Array.from(report.versions.keys(), dumpTypeUrl)])]
		: []
	return (
		<Stack spacing={2}>
			<Paper component='section' variant='outlined' sx={{ p: 2 }}>
				<Stack direction='row' flexWrap='wrap' gap={2} justifyContent='space-between'>
					<Box>
						<Typography component='h2' variant='subtitle1' fontWeight={650}>
							Compare with the served snapshot
						</Typography>
						<Typography variant='body2' color='text.secondary'>
							{validNode
								? 'Request the current controller’s inventory for ' +
									node +
									'. The file stays in this browser.'
								: 'A node ID from the bootstrap section is required.'}
						</Typography>
					</Box>
					<Button variant='outlined' disabled={!validNode || loading} onClick={() => void load()}>
						{loading ? 'Loading…' : snapshot ? 'Refresh comparison' : 'Compare with served'}
					</Button>
				</Stack>
				{snapshot && (
					<Stack direction='row' flexWrap='wrap' gap={3} mt={2}>
						<Box>
							<Typography variant='caption'>File modified</Typography>
							<Typography variant='body2'>
								<ConsoleTime value={modified} fallback='Not reported' />
							</Typography>
						</Box>
						<Box>
							<Typography variant='caption'>Served snapshot published</Typography>
							<Typography variant='body2'>
								<ConsoleTime value={published} fallback='Not reported' />
							</Typography>
						</Box>
						<Box sx={{ minWidth: 0, overflowWrap: 'anywhere' }}>
							<Typography variant='caption'>Served snapshot ID</Typography>
							<Typography variant='body2'>{snapshot.to.id}</Typography>
						</Box>
					</Stack>
				)}
			</Paper>
			{error && <Alert severity='error'>{error}</Alert>}
			{snapshot && (
				<>
					<Alert severity={newer ? 'warning' : 'info'}>
						{newer && 'The served snapshot was published after the file was modified. '}Resource names and
						states describe these two sources. Content equality is unverified.
					</Alert>
					<Paper variant='outlined' sx={{ overflow: 'hidden' }}>
						<Typography component='h2' variant='subtitle1' fontWeight={650} sx={{ p: 2 }}>
							Resources by name
						</Typography>
						<ConfigurationTable
							label='Resources by name'
							rows={compare(report, snapshot)}
							rowId={row => row.id}
							columns={[
								{ label: 'Resource', value: r => r.name },
								{ label: 'Type', value: r => r.type },
								{ label: 'State in file', value: r => r.states.join(', ') || 'Not listed' },
								{
									label: 'Served SHA-256',
									value: r => r.served || (r.listed ? 'Not reported' : 'Not listed'),
									render: r =>
										r.served ? (
											<Box component='details' sx={{ overflowWrap: 'anywhere' }}>
												<Box
													component='summary'
													sx={{ color: 'primary.main', cursor: 'pointer', minHeight: 24 }}
													aria-label={'Show full SHA-256 for ' + r.name}
												>
													{r.served.slice(0, 12)}…
												</Box>
												<code>{r.served}</code>
											</Box>
										) : r.listed ? (
											'Not reported'
										) : (
											'Not listed'
										)
								},
								{
									label: 'Observation',
									value: r =>
										r.states.length
											? r.listed
												? 'Present in both by name'
												: 'Only listed in file'
											: 'Only listed in served inventory'
								}
							]}
						/>
						<Typography variant='body2' color='text.secondary' sx={{ p: 2 }}>
							Per-resource wire versions are unavailable in this controller response. Served SHA-256
							identifies the controller’s resource bytes.
						</Typography>
					</Paper>
					<Paper variant='outlined' sx={{ overflow: 'hidden' }}>
						<Typography component='h2' variant='subtitle1' fontWeight={650} sx={{ p: 2 }}>
							xDS type versions
						</Typography>
						<ConfigurationTable
							label='xDS type versions'
							rows={versions}
							rowId={url => url}
							columns={[
								{ label: 'Type', value: url => labels[url] || url },
								{
									label: 'File section version',
									value: url => report.versions.get(labels[url] as ResourceType) || 'Not reported'
								},
								{
									label: 'Served snapshot version',
									value: url => snapshot.to.versions?.[url] || 'Not reported'
								}
							]}
						/>
					</Paper>
				</>
			)}
		</Stack>
	)
}
