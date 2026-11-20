import { useEffect } from 'react'
import {
	Alert,
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	LinearProgress,
	Paper,
	Stack,
	Typography
} from '@mui/material'
import { useQuery } from '@tanstack/react-query'
import { apiError, getSnapshotResource, manualQueryOptions, httpStatus, typeLabel, type ResourceChange } from './api'

interface ResourceSideProps {
	nodeId: string
	selector: string
	snapshotId: string
	row: ResourceChange
	expectedHash: string
	side: 'From' | 'To'
	onChanged: () => void
}

function ResourceSide({ nodeId, selector, snapshotId, row, expectedHash, side, onChanged }: ResourceSideProps) {
	const query = useQuery({
		...manualQueryOptions,
		queryKey: ['beta', 'snapshot-resource', nodeId, selector, row.typeUrl, row.name, expectedHash],
		queryFn: ({ signal }) => getSnapshotResource(nodeId, selector, row.typeUrl, row.name, signal),
		enabled: Boolean(expectedHash),
		gcTime: 0
	})
	const resource = query.data
	const matches =
		resource?.sha256 === expectedHash && resource?.name === row.name && resource?.typeUrl === row.typeUrl

	const changed = (query.isSuccess && !matches) || httpStatus(query.error) === 404
	useEffect(() => {
		if (changed) onChanged()
	}, [changed, onChanged])

	return (
		<Paper variant='outlined' sx={{ minWidth: 0, p: 2 }}>
			<Stack spacing={1.5}>
				<Typography variant='subtitle2'>
					{side} · {selector}
				</Typography>
				<Typography component='code' variant='body2' sx={{ fontFamily: 'monospace' }}>
					{snapshotId}
				</Typography>
				{!expectedHash ? (
					<Typography variant='body2' color='text.secondary'>
						Resource absent from this snapshot.
					</Typography>
				) : (
					<>
						<Typography component='div' variant='caption' color='text.secondary'>
							SHA-256
							<Box
								component='code'
								sx={{ display: 'block', color: 'text.primary', overflowWrap: 'anywhere' }}
							>
								{expectedHash}
							</Box>
						</Typography>
						{query.isPending && <LinearProgress aria-label={`Loading ${side.toLowerCase()} resource`} />}
						{query.isError && <Alert severity='error'>{apiError(query.error)}</Alert>}
						{query.isSuccess && !matches && (
							<Alert severity='warning'>
								The resource changed since this comparison. Close this view and refresh the comparison.
							</Alert>
						)}
						{query.isSuccess &&
							matches &&
							(row.typeUrl.endsWith('.Secret') ? (
								<Typography variant='body2' color='text.secondary'>
									Only the secret digest is available.
								</Typography>
							) : (
								<Box
									component='pre'
									tabIndex={0}
									aria-label={`${side} resource JSON`}
									sx={{
										m: 0,
										p: 1.5,
										bgcolor: 'action.hover',
										borderRadius: 1,
										overflow: 'auto',
										maxHeight: '50vh',
										fontSize: '0.8125rem'
									}}
								>
									{JSON.stringify(resource?.json, null, 2)}
								</Box>
							))}
					</>
				)}
			</Stack>
		</Paper>
	)
}

interface Props {
	nodeId: string
	from: string
	to: string
	fromId: string
	toId: string
	row: ResourceChange
	onClose: () => void
	onChanged: () => void
}

export default function SnapshotResourceDialog({ nodeId, from, to, fromId, toId, row, onClose, onChanged }: Props) {
	return (
		<Dialog open onClose={onClose} fullWidth maxWidth='lg' aria-labelledby='resource-dialog-title'>
			<DialogTitle id='resource-dialog-title' sx={{ overflowWrap: 'anywhere' }}>
				{row.name}
			</DialogTitle>
			<DialogContent>
				<Typography variant='body2' color='text.secondary' sx={{ mb: 2 }}>
					{typeLabel(row.typeUrl)}
				</Typography>
				<Box
					sx={{
						display: 'grid',
						gap: 2,
						gridTemplateColumns: { xs: '1fr', md: 'repeat(2, minmax(0, 1fr))' }
					}}
				>
					<ResourceSide
						nodeId={nodeId}
						row={row}
						selector={from}
						snapshotId={fromId}
						expectedHash={row.fromSha256}
						side='From'
						onChanged={onChanged}
					/>
					<ResourceSide
						nodeId={nodeId}
						row={row}
						selector={to}
						snapshotId={toId}
						expectedHash={row.toSha256}
						side='To'
						onChanged={onChanged}
					/>
				</Box>
			</DialogContent>
			<DialogActions>
				<Button onClick={onClose}>Close</Button>
			</DialogActions>
		</Dialog>
	)
}
