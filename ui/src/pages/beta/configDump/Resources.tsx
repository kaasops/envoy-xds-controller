import { ConsoleTime } from '../time/ConsoleTime'
import { useConsoleTime } from '../time/useConsoleTime'
import { Box, Paper, Stack, Typography } from '@mui/material'
import SectionTable, { type Column } from '../console/SectionTable'
import ConfigurationTable from '../nodeOverview/ConfigurationTable'
import { type DumpReport, type DumpResource as Row } from '../console/configDump'
import { ResourceState } from './Summary'

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
				{Object.entries({
					Version: row.version,
					'Last updated': row.updated ? <ConsoleTime value={row.updated} fallback='Not reported' /> : '',
					Reason: row.reason || '',
					...row.details
				})
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

export default function Resources({ report }: { report: DumpReport }) {
	const { formatTime } = useConsoleTime()
	const columns: Column<Row>[] = [
		{ key: 'name', label: 'Resource', value: row => row.name || 'Name not reported' },
		{ key: 'type', label: 'Type', value: row => row.type },
		{ key: 'state', label: 'State', value: row => row.state, render: row => <ResourceState state={row.state} /> },
		{ key: 'version', label: 'Version', value: row => row.version },
		{
			key: 'updated',
			label: 'Last updated',
			value: row => formatTime(row.updated) || 'Not reported',
			sortValue: row => Date.parse(row.updated) || 0,
			render: row => <ConsoleTime value={row.updated} fallback='Not reported' />
		}
	]
	return (
		<Stack spacing={2}>
			<Paper variant='outlined' sx={{ overflow: 'hidden' }}>
				<Typography component='h2' variant='subtitle1' fontWeight={650} sx={{ p: 2 }}>
					Sections in this file
				</Typography>
				<ConfigurationTable
					label='Dump sections'
					rows={report.sections}
					rowId={s => s.name}
					columns={[
						{ label: 'Section', value: s => s.name },
						{
							label: 'Contents',
							value: s =>
								s.projected
									? s.name === 'BootstrapConfigDump'
										? 'Node and bootstrap information'
										: s.count + ' resource states'
									: 'Additional section (contents omitted)'
						},
						{ label: 'Scope', value: s => (s.filtered ? 'Individual entries' : 'Section') }
					]}
				/>
			</Paper>
			<SectionTable
				localState
				title='Resources'
				tableTitle='Config dump'
				rows={report.rows}
				rowId={row => row.id}
				columns={columns}
				detail={row => <Details row={row} />}
				emptyText='No supported resource entries in this file.'
			/>
		</Stack>
	)
}
