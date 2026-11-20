import { useState } from 'react'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'
import {
	Alert,
	Box,
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
import { useQuery } from '@tanstack/react-query'
import { apiError, fleetQueryKey, getFleet, httpStatus, manualQueryOptions, nodePath } from './api'

export default function FleetNodes() {
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
	return (
		<Stack spacing={2}>
			<Stack direction='row' justifyContent='space-between' alignItems='center' gap={2}>
				<Typography component='h1' variant='h4' fontWeight={600}>
					{frozenOnly ? 'Frozen nodes' : 'Nodes'}
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
						{nodes.length} nodes · {frozenCount} frozen · Streams on{' '}
						{query.data.controllerPod || 'this controller'} · Updated{' '}
						{new Date(query.dataUpdatedAt).toLocaleTimeString()}
					</Typography>
					<Paper variant='outlined' sx={{ overflow: 'hidden' }}>
						<Box sx={{ p: 1.5 }}>
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
						</Box>
						<TableContainer>
							<Table size='small' aria-label='Nodes' sx={{ minWidth: 610 }}>
								<TableHead>
									<TableRow>
										<TableCell>Node ID</TableCell>
										<TableCell>Freeze</TableCell>
										<TableCell align='right'>xDS streams</TableCell>
										<TableCell>Frozen since</TableCell>
										<TableCell>Reason</TableCell>
									</TableRow>
								</TableHead>
								<TableBody>
									{visible.slice(current * perPage, (current + 1) * perPage).map(node => (
										<TableRow hover key={node.nodeId}>
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
										</TableRow>
									))}
									{!visible.length && (
										<TableRow>
											<TableCell colSpan={5} align='center' sx={{ py: 4 }}>
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
