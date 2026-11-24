import { ConsoleTime } from '../time/ConsoleTime'
import { useEffect, useRef } from 'react'
import { Link as RouterLink, useLocation, useParams, useSearchParams } from 'react-router-dom'
import { useIsFetching, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Box, Button, LinearProgress, Paper, Stack, Typography } from '@mui/material'
import PublicOutlined from '@mui/icons-material/PublicOutlined'
import HearingOutlined from '@mui/icons-material/HearingOutlined'
import KeyOutlined from '@mui/icons-material/KeyOutlined'
import SyncAltOutlined from '@mui/icons-material/SyncAltOutlined'
import RefreshOutlined from '@mui/icons-material/RefreshOutlined'
import LayersOutlined from '@mui/icons-material/LayersOutlined'
import GetOverviewService from '../../../api/services/getOverviewService'
import { apiError, fleetQueryKey, httpStatus, manualQueryOptions, nodeOverviewPath, nodePath } from '../api'
import { getPauses } from '../console/nodeInventory'
import { clientsQueryKey, getClients } from '../console/streams'
import ConfigurationTabs from './ConfigurationTabs'
import { resourceHashesKey } from './configurationData'
import { NodeAlerts, NodeStreams } from './NodeSignals'

const tabs = [
	{ id: 'endpoints', label: 'Listener endpoints' },
	{ id: 'certificates', label: 'Certificates' },
	{ id: 'streams', label: 'xDS streams' },
	{ id: 'hashes', label: 'Resource hashes' }
] as const

const onOpen = { ...manualQueryOptions, refetchOnMount: 'always', retryOnMount: true } as const

function NodePage({ nodeId }: { nodeId: string }) {
	const [params] = useSearchParams()
	const location = useLocation()
	const requestedTab = params.get('tab')
	const tab = tabs.find(t => t.id === requestedTab)?.id ?? 'endpoints'
	const selected = params.get('selected') || ''
	const queryClient = useQueryClient()
	const overview = useQuery({
		...onOpen,
		queryKey: ['beta', 'node-overview', nodeId],
		queryFn: ({ signal }) => GetOverviewService.getOverview(nodeId, signal)
	})
	const streams = useQuery({
		...onOpen,
		queryKey: clientsQueryKey(nodeId),
		queryFn: ({ signal }) => getClients(signal, nodeId)
	})
	const pauses = useQuery({
		...onOpen,
		queryKey: [...fleetQueryKey, 'node-overview-pauses'],
		queryFn: ({ signal }) => getPauses(signal)
	})

	const status = httpStatus(overview.error)
	const denied = status === 401 || status === 403
	const noConfig = status === 400 || status === 404
	const config = overview.isFetchedAfterMount && overview.isSuccess ? overview.data : undefined
	const clients = !denied && streams.isFetchedAfterMount && streams.isSuccess ? streams.data : undefined
	const pause = !denied && pauses.isFetchedAfterMount && pauses.isSuccess ? pauses.data.get(nodeId) : undefined
	const hashKey = resourceHashesKey(nodeId)
	const fetchingHashes = useIsFetching({ queryKey: hashKey, exact: true }) > 0
	const fetching = overview.isFetching || streams.isFetching || pauses.isFetching || fetchingHashes
	function refresh() {
		void overview.refetch()
		void streams.refetch()
		void pauses.refetch()
		if (tab === 'hashes') void queryClient.refetchQueries({ queryKey: hashKey, type: 'active', exact: true })
	}

	const panel = useRef<HTMLDivElement>(null)
	const lastFocus = useRef('')
	useEffect(() => {
		if (!requestedTab || lastFocus.current === location.key) return
		const target = panel.current?.querySelector<HTMLElement>(
			selected ? '[data-node-detail]' : 'nav [aria-current="page"]'
		)
		if (target) {
			lastFocus.current = location.key
			target.focus()
			target.scrollIntoView({ block: 'nearest' })
		}
	}, [location.key, requestedTab, selected, config, clients, fetchingHashes])

	const summary = config?.summary
	const expiring = summary && summary.certificatesWarning + summary.certificatesCritical + summary.certificatesExpired
	const metrics = [
		{
			label: 'Domains',
			value: summary?.totalDomains,
			note: 'In listener configuration',
			tab: 'endpoints',
			Icon: PublicOutlined
		},
		{
			label: 'Listener endpoints',
			value: summary?.totalEndpoints,
			note: 'Incoming domains and ports',
			tab: 'endpoints',
			Icon: HearingOutlined
		},
		{
			label: 'Certificates',
			value: summary?.totalCertificates,
			note: expiring == null ? '' : expiring + ' expiring or expired',
			tab: 'certificates',
			Icon: KeyOutlined
		},
		{
			label: 'xDS streams',
			value: clients?.clients.filter(s => s.nodeId === nodeId).length,
			note: clients?.controllerPod ? 'On ' + clients.controllerPod : 'On the responding controller',
			tab: 'streams',
			Icon: SyncAltOutlined
		}
	]
	return (
		<Stack spacing={2}>
			<Stack direction='row' alignItems='center' justifyContent='space-between' flexWrap='wrap' gap={1.5}>
				<Box>
					<Typography component='h1' variant='h4'>
						{nodeId}
					</Typography>
					<Typography variant='body2' color='text.secondary'>
						Node overview
						{config && (
							<>
								{' '}
								· Configuration fetched <ConsoleTime value={overview.dataUpdatedAt} />
							</>
						)}
					</Typography>
				</Box>
				<Stack direction='row' gap={1}>
					<Button
						component={RouterLink}
						to={nodePath(nodeId)}
						startIcon={<LayersOutlined />}
						variant='outlined'
					>
						Snapshots
					</Button>
					<Button onClick={refresh} disabled={fetching} startIcon={<RefreshOutlined />} variant='outlined'>
						Refresh
					</Button>
				</Stack>
			</Stack>
			{fetching && <LinearProgress aria-label='Loading node data' />}
			{overview.isError && (
				<Alert severity={noConfig ? 'info' : 'error'}>
					{denied
						? 'Access to the node configuration was denied.'
						: noConfig
							? 'No configuration is available for this node in the controller cache.'
							: 'Could not load node configuration. ' + apiError(overview.error)}
				</Alert>
			)}
			{!denied && (
				<>
					{streams.isError && (
						<Alert severity='warning'>Could not load xDS streams. {apiError(streams.error)}</Alert>
					)}
					{pauses.isError && (
						<Alert severity='warning'>Could not load update pause status. {apiError(pauses.error)}</Alert>
					)}
					<Paper
						variant='outlined'
						aria-label='Node summary'
						component='section'
						sx={{
							display: 'grid',
							gridTemplateColumns: { xs: 'repeat(2,minmax(0,1fr))', md: 'repeat(4,minmax(0,1fr))' },
							gap: '1px',
							bgcolor: 'divider',
							overflow: 'hidden'
						}}
					>
						{metrics.map(({ label, value, note, tab: target, Icon }) => (
							<Box
								key={label}
								component={RouterLink}
								to={nodeOverviewPath(nodeId, target)}
								sx={{
									color: 'inherit',
									textDecoration: 'none',
									p: 2,
									minWidth: 0,
									bgcolor: 'background.paper',
									'&:hover': { color: 'primary.main' },
									'&:focus-visible': {
										outline: '2px solid',
										outlineColor: 'primary.main',
										outlineOffset: -2
									}
								}}
							>
								<Stack direction='row' gap={0.75} alignItems='center' color='text.secondary'>
									<Icon fontSize='small' />
									<Typography variant='body2'>{label}</Typography>
								</Stack>
								<Typography
									component='div'
									variant={value == null ? 'subtitle1' : 'h5'}
									sx={{ my: 0.5, fontVariantNumeric: 'tabular-nums' }}
								>
									{value ?? 'Unavailable'}
								</Typography>
								<Typography variant='caption' color='text.secondary'>
									{note}
								</Typography>
							</Box>
						))}
					</Paper>
					{(config || clients || pause) && (
						<NodeAlerts nodeId={nodeId} overview={config} clients={clients} pause={pause} />
					)}
					<Paper variant='outlined' ref={panel} sx={{ minWidth: 0, overflow: 'hidden' }}>
						<Box
							component='nav'
							aria-label='Node data'
							sx={{
								display: 'flex',
								flexWrap: 'wrap',
								columnGap: 1,
								px: 1,
								borderBottom: 1,
								borderColor: 'divider'
							}}
						>
							{tabs.map(t => (
								<Button
									key={t.id}
									component={RouterLink}
									to={nodeOverviewPath(nodeId, t.id)}
									aria-current={t.id === tab ? 'page' : undefined}
									sx={{
										borderRadius: 0,
										py: 1,
										color: t.id === tab ? 'primary.main' : 'text.secondary',
										borderBottom: 2,
										borderBottomColor: t.id === tab ? 'primary.main' : 'transparent'
									}}
								>
									{t.label}
								</Button>
							))}
						</Box>
						{tab === 'streams'
							? clients && <NodeStreams nodeId={nodeId} selected={selected} clients={clients} />
							: (config || (tab === 'hashes' && overview.isError && !noConfig)) && (
									<ConfigurationTabs
										nodeId={nodeId}
										overview={config}
										selected={selected}
										tab={tab}
									/>
								)}
					</Paper>
				</>
			)}
		</Stack>
	)
}

export default function NodeOverview() {
	const { nodeId } = useParams()
	return nodeId ? <NodePage key={nodeId} nodeId={nodeId} /> : null
}
