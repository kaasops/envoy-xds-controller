import { consoleMonospace } from './theme'
import { useCallback, useRef, useState } from 'react'
import {
	Alert,
	Box,
	Button,
	Chip,
	Dialog,
	DialogActions,
	DialogContent,
	DialogContentText,
	DialogTitle,
	LinearProgress,
	MenuItem,
	Paper,
	Stack,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TablePagination,
	TableRow,
	TextField,
	Typography,
	type ChipProps
} from '@mui/material'
import RefreshIcon from '@mui/icons-material/Refresh'
import { useIsMutating, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import {
	apiError,
	fleetQueryKey,
	getSnapshotDiff,
	getSnapshotState,
	httpStatus,
	manualQueryOptions,
	nodePath,
	snapshotQueryKey,
	typeLabel,
	unfreezeNode,
	type ResourceChange,
	type SnapshotDiff
} from './api'
import SnapshotResourceDialog from './SnapshotResourceDialog'

const changes = {
	changed: { label: 'Changed', color: 'warning' },
	added: { label: 'Added', color: 'success' },
	removed: { label: 'Removed', color: 'error' },
	sameDecoded: { label: 'Encoding only', color: 'info' },
	same: { label: 'Unchanged', color: 'default' }
} satisfies Record<string, { label: string; color: ChipProps['color'] }>

type Change = keyof typeof changes
const changeTypes = Object.keys(changes) as Change[]

function ResourceTable({ diff, inspect }: { diff: SnapshotDiff; inspect: (row: ResourceChange) => void }) {
	const [search, setSearch] = useState('')
	const [filter, setFilter] = useState('differences')
	const [page, setPage] = useState(0)
	const [pageSize, setPageSize] = useState(25)
	const counts = { changed: 0, added: 0, removed: 0, sameDecoded: 0, same: 0 }
	for (const summary of Object.values(diff.summary)) {
		for (const type of changeTypes) counts[type] += summary[type]
	}
	const term = search.trim().toLowerCase()
	const rows = diff.resources.filter(row => {
		const matchesChange =
			filter === 'all' || (filter === 'differences' ? row.change !== 'same' : row.change === filter)
		return (
			matchesChange &&
			(row.name.toLowerCase().includes(term) || typeLabel(row.typeUrl).toLowerCase().includes(term))
		)
	})
	const visiblePage = Math.min(page, Math.max(0, Math.ceil(rows.length / pageSize) - 1))
	const start = visiblePage * pageSize
	return (
		<Paper variant='outlined'>
			<Stack spacing={1.5} sx={{ p: 2 }}>
				<Typography component='h2' variant='h6'>
					Resources
				</Typography>
				<Stack direction='row' flexWrap='wrap' gap={2}>
					{changeTypes.map(type => (
						<Typography key={type} variant='body2' color='text.secondary'>
							<Box component='strong' sx={{ color: 'text.primary' }}>
								{counts[type]}
							</Box>{' '}
							{changes[type].label.toLowerCase()}
						</Typography>
					))}
				</Stack>
				<Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5}>
					<TextField
						label='Filter by name or type'
						size='small'
						value={search}
						onChange={e => {
							setSearch(e.target.value)
							setPage(0)
						}}
						sx={{ flexGrow: 1 }}
					/>
					<TextField
						select
						label='Change'
						size='small'
						value={filter}
						onChange={e => {
							setFilter(e.target.value)
							setPage(0)
						}}
						sx={{ minWidth: 160 }}
					>
						<MenuItem value='differences'>Differences only</MenuItem>
						<MenuItem value='all'>All resources</MenuItem>
						{changeTypes.map(type => (
							<MenuItem key={type} value={type}>
								{changes[type].label}
							</MenuItem>
						))}
					</TextField>
				</Stack>
			</Stack>
			{rows.length > 0 ? (
				<>
					<TableContainer>
						<Table size='small' aria-label='Resource changes'>
							<TableHead>
								<TableRow>
									<TableCell>Resource</TableCell>
									<TableCell>Type</TableCell>
									<TableCell>Change</TableCell>
								</TableRow>
							</TableHead>
							<TableBody>
								{rows.slice(start, start + pageSize).map(row => (
									<TableRow key={JSON.stringify([row.typeUrl, row.name])} hover>
										<TableCell>
											<Button
												variant='text'
												size='small'
												onClick={() => inspect(row)}
												sx={{
													textTransform: 'none',
													textAlign: 'left',
													overflowWrap: 'anywhere',
													justifyContent: 'flex-start'
												}}
											>
												{row.name}
											</Button>
										</TableCell>
										<TableCell>{typeLabel(row.typeUrl)}</TableCell>
										<TableCell>
											<Chip
												data-status
												size='small'
												variant='outlined'
												label={changes[row.change].label}
												color={changes[row.change].color}
											/>
										</TableCell>
									</TableRow>
								))}
							</TableBody>
						</Table>
					</TableContainer>
					{rows.length > 10 && (
						<TablePagination
							component='div'
							count={rows.length}
							page={visiblePage}
							rowsPerPage={pageSize}
							rowsPerPageOptions={[10, 25, 50]}
							onPageChange={(_, next) => setPage(next)}
							onRowsPerPageChange={e => {
								setPageSize(Number(e.target.value))
								setPage(0)
							}}
						/>
					)}
				</>
			) : (
				<Typography color='text.secondary' variant='body2' sx={{ p: 2 }}>
					{diff.resources.length === 0
						? 'Both snapshots are empty.'
						: !term && filter === 'differences'
							? 'No resource differences. Select All resources to inspect unchanged resources.'
							: 'No resources match these filters.'}
				</Typography>
			)}
		</Paper>
	)
}

interface CompareProps {
	nodeId: string
	from: string
	to: string
	select: (from: string, to: string) => void
}

function Compare({ nodeId, from, to, select }: CompareProps) {
	const navigate = useNavigate()
	const client = useQueryClient()
	const [selected, setSelected] = useState<ResourceChange | null>(null)
	const [reviewedId, setReviewedId] = useState<string | null>(null)
	const [resourceChanged, setResourceChanged] = useState(false)
	const flagResourceChange = useCallback(() => setResourceChanged(true), [])
	const submitting = useRef(false)
	const mutationKey = [...snapshotQueryKey(nodeId), 'write']
	const writing = useIsMutating({ mutationKey }) > 0
	const state = useQuery({
		...manualQueryOptions,
		queryKey: snapshotQueryKey(nodeId),
		queryFn: ({ signal }) => getSnapshotState(nodeId, signal)
	})
	const diff = useQuery({
		...manualQueryOptions,
		queryKey: ['beta', 'snapshot-diff', nodeId, from, to],
		queryFn: ({ signal }) => getSnapshotDiff(nodeId, from, to, signal),
		refetchOnMount: 'always',
		gcTime: 0
	})
	const unfreeze = useMutation({
		mutationKey,
		mutationFn: (expected: string) => unfreezeNode(nodeId, expected),
		retry: false,
		networkMode: 'always',
		onSuccess: result => {
			client.setQueryData(snapshotQueryKey(nodeId), result)
			void client.invalidateQueries({ queryKey: fleetQueryKey })
		}
	})
	const compared = diff.data
	const info = state.data
	const publishing = from === 'served' && to === 'built' && info?.frozen && info.canControl
	const candidateMatches = Boolean(
		compared?.to.id && info?.built?.id === compared.to.id && (!info.served || info.served.id === compared.from.id)
	)
	const busy = state.isFetching || diff.isFetching || writing
	const mayUnfreeze =
		publishing &&
		candidateMatches &&
		!resourceChanged &&
		!info?.built?.error &&
		state.isSuccess &&
		diff.isSuccess &&
		!busy &&
		!unfreeze.isError
	const snapshots = [...new Set(['served', 'built', ...(info?.history.map(item => item.id) ?? []), from, to])]
	const label = (selector: string) => {
		if (selector === 'served') return 'Served'
		if (selector === 'built') return 'Built'
		return `History · ${selector}`
	}
	function refresh() {
		setReviewedId(null)
		setSelected(null)
		setResourceChanged(false)
		unfreeze.reset()
		void Promise.all([state.refetch(), diff.refetch()])
	}
	function confirm() {
		if (
			!mayUnfreeze ||
			!reviewedId ||
			reviewedId !== compared?.to.id ||
			submitting.current ||
			client.isMutating({ mutationKey })
		)
			return
		submitting.current = true
		unfreeze.mutate(reviewedId, {
			onSuccess: result =>
				navigate(nodePath(nodeId), {
					state: { unfrozenNode: nodeId, servedId: result.served?.id }
				}),
			onSettled: () => {
				submitting.current = false
				setReviewedId(null)
			}
		})
	}
	return (
		<Stack spacing={2.5}>
			<Stack direction='row' alignItems='center' justifyContent='space-between' gap={2}>
				<Typography variant='h4' component='h1'>
					Compare snapshots
				</Typography>
				<Button variant='outlined' onClick={refresh} disabled={busy} startIcon={<RefreshIcon />}>
					Refresh
				</Button>
			</Stack>
			<Paper variant='outlined' sx={{ p: 2 }}>
				<Stack direction={{ xs: 'column', md: 'row' }} gap={2} alignItems={{ xs: 'stretch', md: 'flex-start' }}>
					{(['From', 'To'] as const).map(side => {
						const value = side === 'From' ? from : to
						return (
							<TextField
								key={side}
								select
								label={side}
								value={value}
								size='small'
								disabled={unfreeze.isPending}
								helperText={
									compared && !diff.isError
										? side === 'From'
											? compared.from.id
											: compared.to.id
										: undefined
								}
								onChange={e =>
									side === 'From' ? select(e.target.value, to) : select(from, e.target.value)
								}
								sx={{
									minWidth: 170,
									flex: 1,
									'& .MuiFormHelperText-root': { fontFamily: consoleMonospace }
								}}
							>
								{snapshots.map(selector => (
									<MenuItem key={selector} value={selector}>
										{label(selector)}
									</MenuItem>
								))}
							</TextField>
						)
					})}
					{publishing && (
						<Button
							variant='contained'
							disabled={!mayUnfreeze}
							onClick={() => setReviewedId(compared?.to.id ?? null)}
						>
							Unfreeze node
						</Button>
					)}
				</Stack>
			</Paper>
			{(state.isFetching || diff.isFetching) && <LinearProgress aria-label='Loading snapshot comparison' />}
			{state.isError && <Alert severity='error'>Could not read node state. {apiError(state.error)}</Alert>}
			{diff.isError && <Alert severity='error'>Could not compare snapshots. {apiError(diff.error)}</Alert>}
			{info?.built?.error && (from === 'built' || to === 'built') && (
				<Alert severity='error'>Build failed. {info.built.error}</Alert>
			)}
			{(resourceChanged || (publishing && compared && !candidateMatches)) &&
				!busy &&
				!info?.built?.error &&
				!unfreeze.isError && (
					<Alert severity='warning'>
						The snapshots changed between reads. Refresh to review the current comparison.
					</Alert>
				)}
			{unfreeze.isError && (
				<Alert severity='error'>
					{httpStatus(unfreeze.error) === 409
						? 'The controller declined this change.'
						: 'Could not confirm that the node was unfrozen.'}{' '}
					{apiError(unfreeze.error)} Refresh and review the current snapshots before trying again.
				</Alert>
			)}
			{compared && !diff.isFetching && !diff.isError && !state.isError && (
				<ResourceTable diff={compared} inspect={setSelected} />
			)}
			{selected && compared && (
				<SnapshotResourceDialog
					row={selected}
					nodeId={nodeId}
					from={from}
					to={to}
					fromId={compared.from.id}
					toId={compared.to.id}
					onClose={() => setSelected(null)}
					onChanged={flagResourceChange}
				/>
			)}
			<Dialog
				open={Boolean(reviewedId)}
				onClose={() => {
					if (!unfreeze.isPending) setReviewedId(null)
				}}
				aria-labelledby='confirm-unfreeze-title'
			>
				<DialogTitle id='confirm-unfreeze-title'>Unfreeze {nodeId}?</DialogTitle>
				<DialogContent>
					<DialogContentText>
						The controller will serve snapshot <strong>{reviewedId}</strong> and resume publishing
						subsequent builds to this node.
					</DialogContentText>
				</DialogContent>
				<DialogActions>
					<Button onClick={() => setReviewedId(null)} disabled={unfreeze.isPending}>
						Cancel
					</Button>
					<Button
						variant='contained'
						onClick={confirm}
						disabled={!mayUnfreeze || reviewedId !== compared?.to.id}
					>
						{unfreeze.isPending ? 'Unfreezing…' : 'Unfreeze'}
					</Button>
				</DialogActions>
			</Dialog>
		</Stack>
	)
}

export default function CompareSnapshots() {
	const { nodeId } = useParams()
	const [params, setParams] = useSearchParams()
	if (!nodeId) return <Alert severity='error'>A node ID is required.</Alert>
	const from = params.get('from') || 'served'
	const to = params.get('to') || 'built'
	return (
		<Compare
			key={JSON.stringify([nodeId, from, to])}
			nodeId={nodeId}
			from={from}
			to={to}
			select={(nextFrom, nextTo) => setParams({ from: nextFrom, to: nextTo })}
		/>
	)
}
