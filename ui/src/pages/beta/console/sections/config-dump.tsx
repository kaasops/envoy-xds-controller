import { ConsoleTime } from '../../time/ConsoleTime'
import { useRef, useState } from 'react'
import { Alert, Box, Button, LinearProgress, Paper, Stack, Tab, Tabs, Typography } from '@mui/material'
import { readConfigDump, type DumpReport } from '../configDump'
import CompareServed from '../../configDump/CompareServed'
import DomainTrace from '../../configDump/DomainTrace'
import Resources from '../../configDump/Resources'
import Summary from '../../configDump/Summary'

interface LoadedFile {
	name: string
	modified: number
	report: DumpReport
}
const tabs = ['Summary', 'Domain trace', 'Resources', 'Compare with served']

function Report({ file }: { file: LoadedFile }) {
	const { report } = file
	const [tab, setTab] = useState(0)
	const [selected, setSelected] = useState('')
	const [details, setDetails] = useState(false)
	return (
		<Stack spacing={2}>
			<Stack direction='row' justifyContent='space-between' alignItems='center' flexWrap='wrap' gap={1}>
				<Typography variant='body2' color='text.secondary' sx={{ overflowWrap: 'anywhere' }}>
					{file.name} · File modified <ConsoleTime value={file.modified} fallback='Not reported' />
				</Typography>
				<Button size='small' variant='outlined' aria-expanded={details} onClick={() => setDetails(!details)}>
					File details
				</Button>
			</Stack>
			<Paper
				component='dl'
				variant='outlined'
				sx={{
					m: 0,
					display: 'grid',
					gridTemplateColumns: { xs: 'repeat(2,minmax(0,1fr))', md: 'repeat(4,minmax(0,1fr))' },
					gap: 1,
					p: 2,
					overflowWrap: 'anywhere'
				}}
			>
				{[
					['Node ID', report.node.id],
					['Envoy', report.node.version],
					['Admin API', report.apis.join(', ')],
					['Source', 'Local JSON file'],
					...(details
						? [
								['Node cluster', report.node.cluster],
								['Build revision', report.node.build],
								['ADS API type', report.node.apiType],
								['xDS server', report.node.servers.join(', ')],
								['ADS transport API', report.node.transportApi],
								[
									'Scope',
									report.filtered
										? 'Individual resource entries are included'
										: 'Sections included in the file'
								]
							]
						: [])
				].map(([label, value]) => (
					<Box key={label}>
						<Typography component='dt' variant='caption' color='text.secondary'>
							{label}
						</Typography>
						<Typography component='dd' variant='body2' fontWeight={600} sx={{ m: 0 }}>
							{value || 'Not reported'}
						</Typography>
					</Box>
				))}
			</Paper>
			<Tabs
				value={tab}
				onChange={(_, value: number) => setTab(value)}
				variant='scrollable'
				scrollButtons='auto'
				aria-label='Config dump views'
				sx={{ borderBottom: 1, borderColor: 'divider' }}
			>
				{tabs.map((label, i) => (
					<Tab key={label} id={'dump-tab-' + i} aria-controls={'dump-panel-' + i} label={label} />
				))}
			</Tabs>
			{tabs.map((label, i) => (
				<Box
					key={label}
					role='tabpanel'
					aria-labelledby={'dump-tab-' + i}
					id={'dump-panel-' + i}
					hidden={tab !== i}
					sx={{ minWidth: 0 }}
				>
					{i === 0 && tab === i && (
						<Summary
							report={report}
							openTrace={id => {
								setSelected(id)
								setTab(1)
							}}
						/>
					)}
					{i === 1 && tab === i && <DomainTrace report={report} initialListener={selected} />}
					{i === 2 && tab === i && <Resources report={report} />}
					{i === 3 && <CompareServed report={report} modified={file.modified} />}
				</Box>
			))}
		</Stack>
	)
}

export default function ConfigDump() {
	const [file, setFile] = useState<LoadedFile | null>(null)
	const [loading, setLoading] = useState(false)
	const [error, setError] = useState('')
	const generation = useRef(0)
	const fileInput = useRef<HTMLInputElement>(null)
	function clear() {
		generation.current++
		setFile(null)
		setError('')
		setLoading(false)
	}
	async function load(loadingFile: File) {
		clear()
		const sequence = generation.current
		setLoading(true)
		try {
			const report = readConfigDump(JSON.parse(await loadingFile.text()))
			if (generation.current === sequence)
				setFile({ name: loadingFile.name, modified: loadingFile.lastModified, report })
		} catch (e) {
			if (generation.current === sequence)
				setError(
					e instanceof SyntaxError
						? 'The file is not valid JSON.'
						: 'Could not read this file as an Envoy config dump.'
				)
		} finally {
			if (generation.current === sequence) setLoading(false)
		}
	}
	return (
		<Stack spacing={2}>
			<input
				ref={fileInput}
				hidden
				type='file'
				accept='.json,application/json'
				aria-label='Config dump file'
				onChange={e => {
					const next = e.target.files?.[0]
					e.target.value = ''
					if (next) void load(next)
				}}
			/>
			<Stack direction='row' gap={1} justifyContent='space-between' alignItems='center' flexWrap='wrap'>
				<Typography component='h1' variant='h4'>
					Config dump
				</Typography>
				<Stack direction='row' gap={1}>
					{(file || loading) && <Button onClick={clear}>Clear</Button>}
					<Button variant='outlined' onClick={() => fileInput.current?.click()}>
						Open file
					</Button>
				</Stack>
			</Stack>
			{loading && <LinearProgress aria-label='Reading config dump' />}
			{error && <Alert severity='error'>{error}</Alert>}
			{file ? (
				<Report key={generation.current} file={file} />
			) : (
				!loading &&
				!error && (
					<Paper variant='outlined' sx={{ p: 2 }}>
						<Typography>No config dump loaded.</Typography>
						<Typography variant='body2' color='text.secondary'>
							Open an Envoy admin config_dump JSON file to inspect it in this browser.
						</Typography>
					</Paper>
				)
			)}
		</Stack>
	)
}
