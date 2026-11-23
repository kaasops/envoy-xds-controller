import { useRef, useState } from 'react'
import { Alert, Box, Button, Stack, Typography } from '@mui/material'
import SectionTable, { type Column } from '../SectionTable'
import { readConfigDump } from '../configDump'

type Capture = ReturnType<typeof readConfigDump>
type Row = Capture['rows'][number]

const columns: Column<Row>[] = [
	{ key: 'name', label: 'Resource', value: row => row.name },
	{ key: 'type', label: 'Type', value: row => row.type },
	{ key: 'state', label: 'State', value: row => row.state },
	{ key: 'version', label: 'Version', value: row => row.version },
	{ key: 'updated', label: 'Last updated', value: row => row.updated }
]

function Details({ row }: { row: Row }) {
	return (
		<Stack gap={1}>
			<Typography variant='subtitle2'>
				{row.state} · {row.type}
			</Typography>
			{row.type === 'Secret' && <Typography variant='body2'>Only secret metadata is shown.</Typography>}
			<Box
				component='dl'
				sx={{
					m: 0,
					display: 'grid',
					gridTemplateColumns: 'max-content 1fr',
					gap: 1,
					'& dt': { color: 'text.secondary' },
					'& dd': { m: 0, overflowWrap: 'anywhere' }
				}}
			>
				{Object.entries({ Version: row.version, 'Last updated': row.updated, ...row.details })
					.filter(([, value]) => value !== '')
					.map(([key, value]) => (
						<Stack component='div' key={key} sx={{ display: 'contents' }}>
							<dt>{key}</dt>
							<dd>{value}</dd>
						</Stack>
					))}
			</Box>
			{row.components.length > 0 && (
				<Box component='ul' sx={{ m: 0, pl: 2.25, overflowWrap: 'anywhere' }}>
					{row.components.map((item, index) => (
						<li key={index}>
							<strong>{item.label}</strong> {item.value}
						</li>
					))}
				</Box>
			)}
		</Stack>
	)
}

export default function ConfigDump() {
	const [capture, setCapture] = useState<{ name: string; data: Capture } | null>(null)
	const [loading, setLoading] = useState(false)
	const [error, setError] = useState('')
	const sequence = useRef(0)
	const fileInput = useRef<HTMLInputElement>(null)
	async function load(file: File) {
		const generation = ++sequence.current
		setError('')
		setCapture(null)
		setLoading(true)
		try {
			const data = readConfigDump(JSON.parse(await file.text()))
			if (generation === sequence.current) setCapture({ name: file.name, data })
		} catch (failure) {
			// A JSON parse error may quote the input, including secret material.
			if (generation === sequence.current)
				setError(
					failure instanceof SyntaxError
						? 'The file is not valid JSON.'
						: 'Could not read this file as an Envoy config dump.'
				)
		} finally {
			if (generation === sequence.current) setLoading(false)
		}
	}
	function clear() {
		sequence.current++
		setCapture(null)
		setLoading(false)
		setError('')
	}
	const skipped = capture?.data.skipped ?? 0
	return (
		<Stack gap={1}>
			<input
				ref={fileInput}
				hidden
				type='file'
				accept='.json,application/json'
				aria-label='Config dump file'
				onChange={e => {
					const file = e.target.files?.[0]
					e.target.value = ''
					if (file) void load(file)
				}}
			/>
			{error && <Alert severity='error'>{error}</Alert>}
			{skipped > 0 && (
				<Alert severity='info'>
					{skipped} additional dump {skipped === 1 ? 'section is' : 'sections are'} outside this resource
					list.
				</Alert>
			)}
			<SectionTable
				key={sequence.current}
				localState
				title='Config dump'
				rows={capture?.data.rows || []}
				columns={columns}
				rowId={row => row.id}
				loading={loading}
				description={
					capture
						? [capture.name, capture.data.nodeId].filter(Boolean).join(' · ')
						: 'Open an Envoy admin config_dump JSON file in this browser.'
				}
				emptyText={
					capture ? 'No listeners, routes, clusters or secrets in this dump.' : 'No config dump loaded.'
				}
				actions={
					<Stack direction='row' gap={1}>
						{(capture || loading) && <Button onClick={clear}>Clear</Button>}
						<Button variant='outlined' onClick={() => fileInput.current?.click()}>
							Open file
						</Button>
					</Stack>
				}
				detail={row => <Details row={row} />}
			/>
		</Stack>
	)
}
