import { useState } from 'react'
import { Link as RouterLink, useNavigate, useSearchParams } from 'react-router-dom'
import {
	Alert,
	Button,
	Chip,
	InputAdornment,
	LinearProgress,
	Link,
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
	Typography
} from '@mui/material'
import SearchOutlined from '@mui/icons-material/SearchOutlined'
import RefreshOutlined from '@mui/icons-material/RefreshOutlined'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { apiError, fleetQueryKey, getFleet, httpStatus, manualQueryOptions, nodePath, type SnapshotState } from './api'

import NodeListAction from './NodeListAction'

export default function FleetNodes() {
	const client = useQueryClient()
	const navigate = useNavigate()
	const [params] = useSearchParams()
	const frozenOnly = params.get('frozen') === 'true'
	const [filter, setFilter] = useState('')
	const [page, setPage] = useState(0)
	const [perPage, setPerPage] = useState(25)
	const query = useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: fleetQueryKey,
		queryFn: ({ signal }) => getFleet(signal)
	})
	const nodes = httpStatus(query.error) === 403 ? [] : query.data?.nodes || []
	const visible = nodes.filter(
		node => (!frozenOnly || node.freeze) && node.nodeId.toLowerCase().includes(filter.trim().toLowerCase())
	)
	const current = Math.min(page, Math.max(0, Math.ceil(visible.length / perPage) - 1))
	const frozenCount = nodes.filter(node => node.freeze).length
	const filterLink = (frozen: boolean) => {
		const next = new URLSearchParams(params)
		if (frozen) next.set('frozen', 'true')
		else next.delete('frozen')
		return { search: next.toString() }
	}
	const updateFrozen = async (next: SnapshotState) => {
		await client.cancelQueries({ queryKey: fleetQueryKey, exact: true })
		if (httpStatus(client.getQueryState(fleetQueryKey)?.error) === 403) return
		client.setQueryData<Awaited<ReturnType<typeof getFleet>>>(
			fleetQueryKey,
			previous =>
				previous && {
					...previous,
					nodes: previous.nodes.map(node =>
						node.nodeId === next.nodeId ? { ...node, freeze: next.freeze ?? undefined } : node
					)
				}
		)
	}
	return (
		<Stack spacing={2}>
			<Stack direction='row' justifyContent='space-between' alignItems='center' gap={2}>
				<Typography component='h1' variant='h4' fontWeight={600}>
					Nodes
				</Typography>
				<Button
					variant='outlined'
					startIcon={<RefreshOutlined />}
					disabled={query.isFetching}
					onClick={() => void query.refetch()}
				>
					Refresh
				</Button>
			</Stack>
			{query.isFetching && <LinearProgress aria-label='Loading nodes' />}
			{query.isError && (
				<Alert severity={nodes.length ? 'warning' : 'error'}>
					{nodes.length ? 'Refresh failed. Showing the last node list. ' : 'Could not load nodes. '}
					{apiError(query.error)}
				</Alert>
			)}
			{query.data && httpStatus(query.error) !== 403 && (
				<>
					<Typography variant='body2' color='text.secondary'>
						{nodes.length} nodes ·{' '}
						<Link component={RouterLink} to={filterLink(!frozenOnly)}>
							{frozenCount} frozen
						</Link>{' '}
						· Streams on {query.data.controllerPod || 'this controller'} · Updated{' '}
						{new Date(query.dataUpdatedAt).toLocaleTimeString()}
					</Typography>
					<Paper variant='outlined' sx={{ overflow: 'hidden' }}>
						<Stack direction='row' alignItems='center' flexWrap='wrap' gap={1} sx={{ p: 1.5 }}>
							<TextField
								size='small'
								placeholder='Filter by node ID'
								value={filter}
								onChange={event => {
									setFilter(event.target.value)
									setPage(0)
								}}
								inputProps={{ 'aria-label': 'Filter by node ID' }}
								InputProps={{
									startAdornment: (
										<InputAdornment position='start'>
											<SearchOutlined fontSize='small' />
										</InputAdornment>
									)
								}}
							/>
							<Stack direction='row' role='group' aria-label='Freeze filter'>
								{[false, true].map(frozen => (
									<Button
										key={String(frozen)}
										component={RouterLink}
										size='small'
										to={filterLink(frozen)}
										onClick={() => setPage(0)}
										aria-current={frozenOnly === frozen ? 'true' : undefined}
										variant={frozenOnly === frozen ? 'contained' : 'text'}
									>
										{frozen ? 'Frozen (' + frozenCount + ')' : 'All'}
									</Button>
								))}
							</Stack>
						</Stack>
						<TableContainer>
							<Table size='small' aria-label='Nodes' sx={{ minWidth: 610 }}>
								<TableHead>
									<TableRow>
										<TableCell>Node ID</TableCell>
										<TableCell>Freeze</TableCell>
										<TableCell align='right'>xDS streams</TableCell>
										<TableCell>Frozen since</TableCell>
										<TableCell>Reason</TableCell>
										<TableCell align='right'>Actions</TableCell>
									</TableRow>
								</TableHead>
								<TableBody>
									{visible.slice(current * perPage, (current + 1) * perPage).map(node => (
										<TableRow
											hover
											key={node.nodeId}
											tabIndex={0}
											sx={{
												cursor: 'pointer',
												'&:focus-visible': {
													outline: '2px solid',
													outlineOffset: '-2px',
													outlineColor: 'primary.main'
												}
											}}
											onClick={event => {
												const target = event.target
												if (
													target instanceof Element &&
													event.currentTarget.contains(target) &&
													!target.closest('a, button')
												)
													navigate(nodePath(node.nodeId))
											}}
											onKeyDown={event => {
												if (event.target === event.currentTarget && event.key === 'Enter') {
													event.preventDefault()
													navigate(nodePath(node.nodeId))
												}
											}}
										>
											<TableCell component='th' scope='row'>
												<Link
													component={RouterLink}
													to={nodePath(node.nodeId)}
													fontWeight={500}
												>
													{node.nodeId}
												</Link>
											</TableCell>
											<TableCell>
												{node.freeze ? (
													<Chip label='Frozen' variant='outlined' color='info' size='small' />
												) : (
													'Not frozen'
												)}
											</TableCell>
											<TableCell align='right'>{node.streams}</TableCell>
											<TableCell>
												{node.freeze ? new Date(node.freeze.at).toLocaleString() : '–'}
											</TableCell>
											<TableCell sx={{ maxWidth: 300, overflowWrap: 'anywhere' }}>
												{node.freeze?.reason || '–'}
											</TableCell>
											<TableCell align='right'>
												<NodeListAction
													nodeId={node.nodeId}
													frozen={Boolean(node.freeze)}
													disabled={query.isError || query.isFetching}
													onFrozen={updateFrozen}
												/>
											</TableCell>
										</TableRow>
									))}
									{!visible.length && (
										<TableRow>
											<TableCell colSpan={6} align='center' sx={{ py: 4 }}>
												{filter
													? 'No nodes match this filter.'
													: frozenOnly
														? 'No frozen nodes.'
														: 'No nodes available.'}
											</TableCell>
										</TableRow>
									)}
								</TableBody>
							</Table>
						</TableContainer>
						{visible.length > 25 && (
							<TablePagination
								component='div'
								rowsPerPageOptions={[25, 50, 100]}
								count={visible.length}
								rowsPerPage={perPage}
								page={current}
								onPageChange={(_, next) => setPage(next)}
								onRowsPerPageChange={event => {
									setPerPage(Number(event.target.value))
									setPage(0)
								}}
							/>
						)}
					</Paper>
				</>
			)}
		</Stack>
	)
}
