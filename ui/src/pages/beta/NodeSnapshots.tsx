import { useState } from 'react'
import { Link as RouterLink, useLocation, useNavigationType, useParams } from 'react-router-dom'
import { useIsMutating, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
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
	Paper,
	Stack,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Typography
} from '@mui/material'
import RefreshOutlined from '@mui/icons-material/RefreshOutlined'
import {
	apiError,
	fleetQueryKey,
	getSnapshotState,
	httpStatus,
	manualQueryOptions,
	nodePath,
	rollbackNode,
	snapshotQueryKey,
	typeLabel
} from './api'
import type { SnapshotInfo, SnapshotState } from './api'
import FreezeDialog from './FreezeDialog'

type Action = { kind: 'freeze' } | { kind: 'rollback'; to: string; expectServed: string }

const timeFormat = new Intl.DateTimeFormat('en', {
	year: 'numeric',
	month: 'short',
	day: 'numeric',
	hour: '2-digit',
	minute: '2-digit',
	second: '2-digit',
	timeZoneName: 'short'
})

function timestamp(value?: string) {
	return value ? <time dateTime={value}>{timeFormat.format(new Date(value))}</time> : 'Unavailable'
}

function SnapshotSummary({ title, snapshot }: { title: 'Served' | 'Built'; snapshot: SnapshotInfo | null }) {
	const time = title === 'Served' ? snapshot?.publishedAt : snapshot?.builtAt
	return (
		<Box sx={{ p: 2.5, minWidth: 0 }}>
			<Typography variant='subtitle2' component='h2'>
				{title} snapshot
			</Typography>
			<Typography sx={{ mt: 0.75, fontFamily: 'monospace', fontSize: '1.2rem', overflowWrap: 'anywhere' }}>
				{snapshot?.id || 'Unavailable'}
			</Typography>
			{time && (
				<Typography variant='body2' color='text.secondary' sx={{ mt: 0.5 }}>
					{title === 'Served' ? 'Published' : 'Built'} {timestamp(time)}
				</Typography>
			)}
		</Box>
	)
}

function NodePage({ nodeId }: { nodeId: string }) {
	const location = useLocation()
	const navType = useNavigationType()
	const client = useQueryClient()
	const key = snapshotQueryKey(nodeId)
	const mutationKey = [...key, 'write']
	const writing = useIsMutating({ mutationKey }) > 0
	const [justUnfrozen] = useState(() => {
		const last = client.getQueryData<SnapshotState>(key)
		return (
			navType === 'PUSH' &&
			location.state?.unfrozenNode === nodeId &&
			last?.nodeId === nodeId &&
			!last.frozen &&
			Boolean(last.served?.id) &&
			location.state?.servedId === last.served?.id
		)
	})
	const query = useQuery({
		...manualQueryOptions,
		queryKey: key,
		retryOnMount: true,
		networkMode: 'always',
		queryFn: ({ signal }) => getSnapshotState(nodeId, signal),
		refetchOnMount: justUnfrozen ? false : 'always'
	})
	const [action, setAction] = useState<Action | null>(null)
	const [notice, setNotice] = useState(justUnfrozen ? 'Node unfrozen.' : '')
	const [writeError, setWriteError] = useState('')
	const mutation = useMutation({
		mutationKey,
		retry: false,
		networkMode: 'always',
		mutationFn: (request: Extract<Action, { kind: 'rollback' }>) =>
			rollbackNode(nodeId, request.to, request.expectServed),
		onSuccess: async next => {
			await client.cancelQueries({ queryKey: key, exact: true })
			client.setQueryData(key, next)
			void client.invalidateQueries({ queryKey: fleetQueryKey, refetchType: 'none' })
			setAction(null)
			setNotice('Snapshot restored. The node is frozen.')
		},
		onError: error => {
			setAction(null)
			const status = httpStatus(error)
			const context =
				status === 409
					? 'The node state changed.'
					: status === 403
						? 'The controller denied this action.'
						: status === 404
							? 'The node or snapshot is no longer available.'
							: 'The operation could not be confirmed.'
			setWriteError(context + ' ' + apiError(error))
		}
	})
	const refresh = async () => {
		setNotice('')
		const latest = await query.refetch()
		if (latest.isSuccess) setWriteError('')
	}

	const data = query.data
	const servedId = data?.served?.id
	const readStatus = httpStatus(query.error)
	const canShow = data && readStatus !== 403 && readStatus !== 404
	const blocked = query.isFetching || query.isError || writing || Boolean(writeError)
	const open = (next: Action) => {
		setNotice('')
		setAction(next)
	}
	const compare = nodePath(nodeId) + '/compare'
	const compareBuilt = compare + '?from=served&to=built'
	const types = Array.from(
		new Set([
			...Object.keys(data?.served?.counts ?? {}),
			...Object.keys(data?.built?.counts ?? {}),
			...Object.keys(data?.served?.versions ?? {})
		])
	).sort((a, b) => typeLabel(a).localeCompare(typeLabel(b)))
	const canCompare = Boolean(data?.built?.id)
	return (
		<Stack spacing={2.25}>
			<Stack direction='row' alignItems='flex-start' justifyContent='space-between' gap={2} flexWrap='wrap'>
				<Box sx={{ minWidth: 0 }}>
					<Typography variant='h4' component='h1' sx={{ overflowWrap: 'anywhere' }}>
						{nodeId}
					</Typography>
					{canShow && (
						<Stack direction='row' alignItems='center' gap={1.25} flexWrap='wrap' sx={{ mt: 1 }}>
							<Chip
								size='small'
								color={data.frozen ? 'warning' : 'default'}
								label={data.frozen ? 'Frozen' : 'Following builds'}
								variant='outlined'
							/>
							{!data.canControl && (
								<Typography variant='body2' color='text.secondary'>
									Read only
								</Typography>
							)}
							{data.controllerPod && (
								<Typography variant='body2' color='text.secondary' sx={{ overflowWrap: 'anywhere' }}>
									Controller {data.controllerPod}
								</Typography>
							)}
						</Stack>
					)}
				</Box>
				<Stack direction='row' gap={1} flexWrap='wrap'>
					<Button
						startIcon={<RefreshOutlined />}
						variant='outlined'
						disabled={query.isFetching || writing}
						onClick={() => void refresh()}
					>
						Refresh
					</Button>
					{canShow && data.canControl && data.frozen && (
						<Button
							component={RouterLink}
							to={compareBuilt}
							variant='contained'
							aria-describedby={data.built?.error ? 'unfreeze-build-error' : undefined}
							disabled={blocked || !canCompare || Boolean(data.built?.error)}
						>
							Unfreeze…
						</Button>
					)}
					{canShow && data.canControl && !data.frozen && (
						<Button variant='outlined' disabled={blocked} onClick={() => open({ kind: 'freeze' })}>
							Freeze node
						</Button>
					)}
				</Stack>
			</Stack>
			{canShow && data.built?.error && (
				<Alert severity='error' id='unfreeze-build-error'>
					<Typography variant='subtitle2'>Build failed</Typography>
					<Typography variant='body2' sx={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>
						{data.built.error}
					</Typography>
					{data.frozen && <Typography variant='body2'>Fix the objects before unfreezing.</Typography>}
				</Alert>
			)}
			{query.isFetching && <LinearProgress aria-label='Loading node snapshots' />}
			{query.isPending && !data && <Typography role='status'>Loading snapshots…</Typography>}
			{query.isError && (
				<Alert severity='error'>
					{readStatus === 403
						? 'You do not have access to this node.'
						: readStatus === 404
							? 'This node could not be found.'
							: canShow
								? 'Refresh failed. Showing the last snapshots received.'
								: 'Could not load snapshots.'}{' '}
					{apiError(query.error)}
				</Alert>
			)}
			{notice && (
				<Alert severity='success' onClose={() => setNotice('')}>
					{notice}
				</Alert>
			)}
			{writeError && (
				<Alert severity='error'>{writeError} Refresh the node and review its state before trying again.</Alert>
			)}
			{canShow && (
				<>
					{data.frozen && (
						<Alert severity='warning'>
							<Typography variant='subtitle2'>Publication is frozen</Typography>
							{data.freeze && (
								<Typography variant='body2' sx={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>
									Frozen since {timestamp(data.freeze.at)}
									{data.freeze.by && ' by ' + data.freeze.by}.
									<br />
									Reason: {data.freeze.reason}
								</Typography>
							)}
							<Typography variant='body2' sx={{ mt: 0.5 }}>
								{data.canControl
									? 'Choose Unfreeze… to compare the served and built snapshots before publishing.'
									: 'An operator with write access can review and unfreeze this node.'}
							</Typography>
							<Typography variant='body2' sx={{ mt: 0.5 }}>
								The webhook accepts listener address conflicts on this node with a warning.
							</Typography>
							{!data.served && (
								<Typography variant='body2' sx={{ mt: 0.5 }}>
									This controller has no served snapshot. Running Envoys keep their configuration. A
									newly started Envoy waits for the node to be unfrozen.
								</Typography>
							)}
						</Alert>
					)}
					<Paper variant='outlined'>
						<Box
							sx={{
								display: 'grid',
								gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, minmax(0, 1fr))' }
							}}
						>
							<SnapshotSummary title='Served' snapshot={data.served} />
							<Box
								sx={{
									borderColor: 'divider',
									borderLeft: { sm: 1 },
									borderTop: { xs: 1, sm: 0 },
									borderLeftColor: 'divider',
									borderTopColor: 'divider'
								}}
							>
								<SnapshotSummary title='Built' snapshot={data.built} />
							</Box>
						</Box>
						{canCompare && (
							<Box sx={{ borderTop: 1, borderColor: 'divider', px: 2.5, py: 1.25 }}>
								<Button
									component={RouterLink}
									to={compareBuilt}
									disabled={blocked}
									variant='outlined'
									size='small'
								>
									Compare snapshots
								</Button>
							</Box>
						)}
					</Paper>
					<Box>
						<Typography variant='h6' component='h2' sx={{ mb: 1 }}>
							Resources
						</Typography>
						{types.length ? (
							<TableContainer component={Paper} variant='outlined'>
								<Table size='small' aria-label='Snapshot resources by type'>
									<TableHead>
										<TableRow>
											<TableCell>Resource type</TableCell>
											<TableCell align='right'>Served count</TableCell>
											<TableCell align='right'>Built count</TableCell>
											<TableCell>Served version</TableCell>
										</TableRow>
									</TableHead>
									<TableBody>
										{types.map(type => (
											<TableRow key={type}>
												<TableCell component='th' scope='row' title={type}>
													{typeLabel(type)}
												</TableCell>
												<TableCell align='right'>
													{data.served ? data.served.counts[type] ?? 0 : 'Unavailable'}
												</TableCell>
												<TableCell align='right'>
													{data.built?.id ? data.built.counts[type] ?? 0 : 'Unavailable'}
												</TableCell>
												<TableCell sx={{ fontFamily: 'monospace' }}>
													{data.served?.versions[type] || 'Unavailable'}
												</TableCell>
											</TableRow>
										))}
									</TableBody>
								</Table>
							</TableContainer>
						) : (
							<Typography variant='body2' color='text.secondary'>
								No resource counts available.
							</Typography>
						)}
					</Box>
					<Box>
						<Typography variant='h6' component='h2' sx={{ mb: 1 }}>
							Snapshot history
						</Typography>
						{data.history.length ? (
							<TableContainer component={Paper} variant='outlined'>
								<Table size='small' aria-label='Snapshot history'>
									<TableHead>
										<TableRow>
											<TableCell>Snapshot ID</TableCell>
											<TableCell>Published</TableCell>
											<TableCell align='right'>Actions</TableCell>
										</TableRow>
									</TableHead>
									<TableBody>
										{data.history.map((item, index) => (
											<TableRow key={item.id + ':' + index}>
												<TableCell component='th' scope='row' sx={{ fontFamily: 'monospace' }}>
													{item.id}
												</TableCell>
												<TableCell>{timestamp(item.publishedAt)}</TableCell>
												<TableCell align='right'>
													<Stack
														direction='row'
														gap={0.5}
														justifyContent='flex-end'
														flexWrap='wrap'
													>
														<Button
															component={RouterLink}
															size='small'
															to={
																compare +
																'?from=' +
																encodeURIComponent(item.id) +
																'&to=served'
															}
															aria-label={'Compare ' + item.id + ' with served'}
														>
															Compare
														</Button>
														{data.canControl && servedId && item.id !== servedId && (
															<Button
																size='small'
																disabled={blocked}
																aria-label={'Roll back to ' + item.id}
																onClick={() =>
																	open({
																		kind: 'rollback',
																		to: item.id,
																		expectServed: servedId
																	})
																}
															>
																Roll back
															</Button>
														)}
													</Stack>
												</TableCell>
											</TableRow>
										))}
									</TableBody>
								</Table>
							</TableContainer>
						) : (
							<Typography variant='body2' color='text.secondary'>
								This controller has no previous snapshots for this node.
							</Typography>
						)}
					</Box>
				</>
			)}
			{action?.kind === 'freeze' && (
				<FreezeDialog
					nodeId={nodeId}
					hasServed={Boolean(servedId)}
					disabled={blocked || !data?.canControl || data.frozen}
					onClose={() => setAction(null)}
					onFailure={message => {
						setAction(null)
						setWriteError(message)
					}}
					onFrozen={() => {
						void client.invalidateQueries({ queryKey: fleetQueryKey, refetchType: 'none' })
						setNotice('Node frozen.')
					}}
				/>
			)}
			<Dialog
				open={action?.kind === 'rollback'}
				fullWidth
				maxWidth='xs'
				aria-labelledby='snapshot-action-title'
				onClose={() => {
					if (!writing) setAction(null)
				}}
			>
				{action?.kind === 'rollback' && (
					<form
						onSubmit={event => {
							event.preventDefault()
							if (blocked || !data?.canControl || client.isMutating({ mutationKey })) return
							mutation.mutate(action)
						}}
					>
						<DialogTitle id='snapshot-action-title'>Roll back snapshot</DialogTitle>
						<DialogContent>
							<DialogContentText sx={{ overflowWrap: 'anywhere' }}>
								Serve <code>{action.to}</code> on {nodeId} and freeze publication. The current served
								snapshot is <code>{action.expectServed}</code>. Unfreezing will publish the
								configuration built from the current objects.
							</DialogContentText>
						</DialogContent>
						<DialogActions>
							<Button disabled={writing} onClick={() => setAction(null)}>
								Cancel
							</Button>
							<Button variant='contained' type='submit' disabled={blocked}>
								Roll back
							</Button>
						</DialogActions>
					</form>
				)}
			</Dialog>
		</Stack>
	)
}

export default function NodeSnapshots() {
	const { nodeId } = useParams()
	if (!nodeId) return <Alert severity='error'>Missing node ID.</Alert>
	return <NodePage key={nodeId} nodeId={nodeId} />
}
