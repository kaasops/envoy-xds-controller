import { Alert, Box, Button, Chip, Link, Paper, Stack, Typography } from '@mui/material'
import ConfigurationTable from '../nodeOverview/ConfigurationTable'
import { displayName, fileTime, type DumpReport, type DumpResource } from '../console/configDump'

export function ResourceState({ state }: { state: string }) {
	const color =
		state === 'Rejected' ? 'error' : state === 'Warming' ? 'warning' : state === 'Active' ? 'success' : 'default'
	return <Chip data-status label={state} color={color} variant='outlined' size='small' />
}

interface Props {
	report: DumpReport
	openTrace: (id: string) => void
}

export default function Summary({ report, openTrace }: Props) {
	const listeners = report.rows.filter(row => row.type === 'Listener')
	const serving = listeners.filter(row => row.state === 'Active' || row.state === 'Static')
	const rejected = listeners.filter(row => row.state === 'Rejected')
	const activeSockets = serving.flatMap(row => row.listener?.addresses || [])
	function unservedSocket(row: DumpResource) {
		return row.listener?.addresses.find(
			socket =>
				socket.host &&
				socket.port &&
				socket.port !== '0' &&
				!serving.some(active => !active.listener?.addresses.length) &&
				!activeSockets.some(
					active =>
						active.protocol === socket.protocol &&
						(!active.host || active.host === socket.host) &&
						(!active.port || active.port === socket.port)
				)
		)
	}
	const failed = rejected.find(row => unservedSocket(row)) || rejected[0]
	const socket = failed && unservedSocket(failed)
	const portElsewhere = socket && activeSockets.some(active => active.port === socket.port)
	const missingTitle = socket
		? portElsewhere
			? 'Rejected bind address ' + socket.protocol + ' ' + socket.label + ' is not listed among active sockets'
			: 'No active listener on :' + socket.port + ' shown in this file'
		: ''
	const warming = listeners.some(row => row.state === 'Warming')
	const traceable = serving.find(row => row.listener?.chains.length)
	const listenerDocs = 'https://www.envoyproxy.io/docs/envoy/latest/api-v3/admin/v3/config_dump_shared.proto'
	const adminDocs = 'https://www.envoyproxy.io/docs/envoy/latest/operations/admin'
	return (
		<Stack spacing={2}>
			<Box
				sx={{
					display: 'grid',
					gridTemplateColumns: { xs: '1fr', md: 'minmax(0,1.7fr) minmax(0,1fr)' },
					gap: 2
				}}
			>
				<Paper
					component='section'
					variant='outlined'
					sx={{ p: 2, borderLeft: 3, borderLeftColor: failed ? 'error.main' : 'divider' }}
				>
					<Typography variant='subtitle2' color={failed ? 'error.main' : 'text.secondary'}>
						{failed ? 'Rejected listener update' : 'Listener configuration'}
					</Typography>
					<Typography component='h2' variant='h6' fontWeight={650}>
						{missingTitle
							? missingTitle
							: failed
								? displayName(failed) + ' was rejected'
								: serving.length
									? serving.length +
										' active or static listener' +
										(serving.length === 1 ? '' : 's') +
										' in this file'
									: 'No active listener shown in this file'}
					</Typography>
					{failed ? (
						<>
							<Typography variant='body2'>
								{displayName(failed)} · Last failed update: {fileTime(failed.updated)}
							</Typography>
							<Alert severity='error' sx={{ my: 1, overflowWrap: 'anywhere' }}>
								{failed.reason}
							</Alert>
							<Typography variant='body2' color='text.secondary'>
								Envoy recorded the last failed update in <code>error_state</code>. Check the active
								state to see which listener still serves traffic.{' '}
								<Link href={listenerDocs} target='_blank' rel='noreferrer'>
									Learn more about listener states
								</Link>
							</Typography>
							{!failed.listener?.addresses.some(address => address.host && address.port) && (
								<Typography variant='body2' color='text.secondary'>
									Bind address or port not shown in this file.
								</Typography>
							)}
							{Boolean(failed.listener?.chains.length) && (
								<Button size='small' onClick={() => openTrace(failed.id)}>
									Inspect rejected configuration
								</Button>
							)}
						</>
					) : (
						<Typography variant='body2' color='text.secondary'>
							The summary covers listener states included in this file.
						</Typography>
					)}
				</Paper>
				<Paper component='section' variant='outlined' sx={{ p: 2 }}>
					<Typography component='h2' variant='subtitle1' fontWeight={650}>
						Active and static listeners
					</Typography>
					{serving.slice(0, 4).map(row => (
						<Stack
							key={row.id}
							direction='row'
							gap={1}
							justifyContent='space-between'
							sx={{ borderBottom: 1, borderColor: 'divider', py: 0.75, overflowWrap: 'anywhere' }}
						>
							<Box>
								<Typography variant='body2' fontWeight={600}>
									{displayName(row)}
								</Typography>
								<Typography variant='caption' color='text.secondary'>
									{row.state}
								</Typography>
							</Box>
							<Typography variant='body2' sx={{ textAlign: 'right' }}>
								{row.details.Address || 'Address not shown'}
							</Typography>
						</Stack>
					))}
					{traceable && (
						<Button size='small' onClick={() => openTrace(traceable.id)}>
							Trace active listener
						</Button>
					)}
					{serving.length > 4 && (
						<Typography variant='caption'>All {serving.length} are listed below.</Typography>
					)}
					{!serving.length && (
						<Typography variant='body2' color='text.secondary'>
							No entries in this file.
						</Typography>
					)}
				</Paper>
			</Box>
			<Paper component='section' variant='outlined' sx={{ overflow: 'hidden' }}>
				<Typography component='h2' variant='subtitle1' fontWeight={650} sx={{ p: 2 }}>
					Listener states
				</Typography>
				<ConfigurationTable
					label='Listener states'
					rows={listeners}
					rowId={row => row.id}
					columns={[
						{
							label: 'Listener',
							value: displayName,
							render: row =>
								row.listener?.chains.length ? (
									<Button
										size='small'
										sx={{ p: 0, minWidth: 0, textTransform: 'none' }}
										onClick={() => openTrace(row.id)}
									>
										{displayName(row)}
									</Button>
								) : (
									displayName(row)
								)
						},
						{ label: 'Address', value: row => String(row.details.Address || 'Not shown') },
						{ label: 'State', value: row => row.state, render: row => <ResourceState state={row.state} /> },
						{
							label: 'Evidence in file',
							value: row =>
								row.reason ||
								(row.state === 'Warming' ? 'Reason not reported' : row.state + ' listener state')
						},
						{ label: 'Last update', value: row => fileTime(row.updated) }
					]}
				/>
			</Paper>
			{(failed || warming) && (
				<Paper component='section' variant='outlined' sx={{ p: 2, overflowWrap: 'anywhere' }}>
					<Typography component='h2' variant='subtitle1' fontWeight={650}>
						Checks on the Envoy host
					</Typography>
					<Typography variant='body2' color='text.secondary'>
						Use the admin address configured for this Envoy. These examples use 127.0.0.1:9901.
					</Typography>
					<Stack gap={1.5} mt={1.5}>
						{failed && (
							<Box>
								<Typography variant='subtitle2'>Rejected update · error_state</Typography>
								<Typography variant='body2'>
									Check the Envoy log at {fileTime(failed.updated)} for the original diagnostic.
								</Typography>
								{failed.listener?.modules.map(module => (
									<Box key={module.index} sx={{ mt: 0.5 }}>
										<Typography variant='caption' color='text.secondary' display='block'>
											failed_configuration.listener_filters[{module.index}
											].typed_config.dynamic_module_config.module.local.filename
										</Typography>
										<Typography component='code' variant='body2'>
											ls -l '{module.path}'
										</Typography>
									</Box>
								))}
							</Box>
						)}
						{warming && (
							<Box>
								<Typography variant='subtitle2'>Warming · warming_state</Typography>
								<Typography variant='body2'>
									Envoy is preparing this listener. Inspect unready initialization targets to find
									outstanding dependencies.
								</Typography>
								<Typography component='code' variant='body2'>
									curl -fsS 'http://127.0.0.1:9901/init_dump?mask=listener'
								</Typography>
							</Box>
						)}
						{socket && (
							<Box>
								<Typography variant='subtitle2'>
									Rejected bind address · {socket.protocol} {socket.label}
								</Typography>
								<Typography variant='body2'>
									A file can omit listeners or fields. Check the addresses currently listed by this
									Envoy.
								</Typography>
								<Typography component='code' variant='body2'>
									curl -fsS 'http://127.0.0.1:9901/listeners?format=json'
								</Typography>
							</Box>
						)}
						{(warming || socket) && (
							<Link href={adminDocs} target='_blank' rel='noreferrer' variant='body2'>
								Learn more about the Envoy admin endpoints
							</Link>
						)}
					</Stack>
				</Paper>
			)}
		</Stack>
	)
}
