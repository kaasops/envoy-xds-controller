import { Button, Link, Stack, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { nodePath } from '../../api'
import { nodeSection, useSnapshotInventory, type SnapshotEntry } from '../nodeInventory'
import SectionTable, { type Column } from '../SectionTable'

const columns: Column<SnapshotEntry>[] = [
	{
		key: 'node',
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
			label: label + ' version',
			value: row => row.versions?.[section] || 'Unavailable',
			render: row =>
				row.versions?.[section] ? (
					<Link
						component={RouterLink}
						to={nodeSection(section, row.nodeId)}
						aria-label={label + ' version ' + row.versions[section] + ' for ' + row.nodeId}
					>
						{row.versions[section]}
					</Link>
				) : (
					'Unavailable'
				)
		})
	),
	{ key: 'freeze', label: 'Publication', value: row => (row.freeze ? 'Frozen' : 'Not frozen') }
]

export default function Snapshots() {
	const query = useSnapshotInventory()
	return (
		<SectionTable
			title='Snapshots'
			rows={query.data ?? []}
			columns={columns}
			rowId={row => row.nodeId}
			description='Served resource versions in the controller cache, by node.'
			loading={query.isFetching}
			error={query.error}
			updatedAt={query.dataUpdatedAt}
			onRefresh={() => void query.refetch()}
			emptyText='No cached snapshots or frozen nodes.'
			detail={row => (
				<Stack spacing={1.5} alignItems='flex-start'>
					{!row.versions && <Typography variant='body2'>No served snapshot in this controller.</Typography>}
					{row.freeze && (
						<Typography variant='body2'>
							Frozen {new Date(row.freeze.at).toLocaleString()} · {row.freeze.reason}
						</Typography>
					)}
					<Button component={RouterLink} variant='outlined' to={nodePath(row.nodeId)}>
						Inspect snapshots
					</Button>
					<Link component={RouterLink} to={nodePath(row.nodeId) + '/compare?from=served&to=built'}>
						Compare served and built
					</Link>
				</Stack>
			)}
		/>
	)
}
