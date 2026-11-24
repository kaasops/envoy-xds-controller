import StatusHelp from '../StatusHelp'
import VersionValue from '../../VersionValue'
import { Button, Link, Stack, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { nodePath } from '../../api'
import { nodeSection, useSnapshotInventory, type SnapshotEntry } from '../nodeInventory'
import SectionTable, { type Column } from '../SectionTable'

const columns: Column<SnapshotEntry>[] = [
	{
		key: 'node',
		identifier: true,
		label: 'Node ID',
		value: row => row.nodeId,
		render: row => (
			<Link component={RouterLink} to={nodePath(row.nodeId)}>
				{row.nodeId}
			</Link>
		)
	},
	...Object.entries({ listeners: 'LDS', routes: 'RDS', clusters: 'CDS', secrets: 'SDS' }).map(
		([section, label]): Column<SnapshotEntry> => ({
			key: section,
			identifier: true,
			label: label + ' version',
			value: row => row.versions?.[section] || 'Unavailable',
			render: row =>
				row.versions?.[section] ? (
					<VersionValue
						value={row.versions[section]}
						to={nodeSection(section, row.nodeId)}
						label={label + ' version ' + row.versions[section] + ' for ' + row.nodeId}
					/>
				) : (
					'Unavailable'
				)
		})
	),
	{
		key: 'pause',
		label: 'Updates',
		value: row => (row.pause ? 'Paused' : 'Active'),
		render: row =>
			row.pause ? (
				<StatusHelp status='Paused' to={nodePath(row.nodeId)}>
					Paused
				</StatusHelp>
			) : (
				'Active'
			)
	}
]

export default function Snapshots() {
	const query = useSnapshotInventory()
	return (
		<SectionTable
			title='Snapshots'
			tableHeading='Served versions by node'
			rows={query.data ?? []}
			columns={columns}
			rowId={row => row.nodeId}
			description='Served resource versions in the controller cache, by node.'
			loading={query.isFetching}
			error={query.error}
			updatedAt={query.dataUpdatedAt}
			onRefresh={() => void query.refetch()}
			emptyText='No cached snapshots or paused nodes.'
			detail={row => (
				<Stack spacing={1.5} alignItems='flex-start'>
					{!row.versions && <Typography variant='body2'>No served snapshot in this controller.</Typography>}
					{row.pause && (
						<Typography variant='body2'>
							Paused {new Date(row.pause.at).toLocaleString()} · {row.pause.reason}
						</Typography>
					)}
					<Button component={RouterLink} variant='outlined' to={nodePath(row.nodeId)}>
						Inspect snapshots
					</Button>
					{row.pause && (
						<Link component={RouterLink} to={nodePath(row.nodeId) + '/compare?from=served&to=pending'}>
							Compare served and pending
						</Link>
					)}
				</Stack>
			)}
		/>
	)
}
