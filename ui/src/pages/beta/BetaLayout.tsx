import { useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { Link as RouterLink, Outlet, useMatch, useLocation, useNavigate } from 'react-router-dom'
import {
	Box,
	Breadcrumbs,
	IconButton,
	Link,
	List,
	Drawer,
	Autocomplete,
	TextField,
	InputAdornment,
	ListItemButton,
	Stack,
	Typography,
	ThemeProvider,
	useTheme
} from '@mui/material'
import AccountTreeOutlined from '@mui/icons-material/AccountTreeOutlined'
import MenuOutlined from '@mui/icons-material/MenuOutlined'
import SearchOutlined from '@mui/icons-material/SearchOutlined'
import { sections, groups } from './console/sections'
import ContrastOutlined from '@mui/icons-material/ContrastOutlined'
import { ColorModeContext } from '../../theme'
import { nodePath } from './api'
import { createConsoleTheme } from './theme'

import DashboardOutlined from '@mui/icons-material/DashboardOutlined'
import DnsOutlined from '@mui/icons-material/DnsOutlined'
import SyncAltOutlined from '@mui/icons-material/SyncAltOutlined'
import LayersOutlined from '@mui/icons-material/LayersOutlined'
import HearingOutlined from '@mui/icons-material/HearingOutlined'
import AltRouteOutlined from '@mui/icons-material/AltRouteOutlined'
import HubOutlined from '@mui/icons-material/HubOutlined'
import KeyOutlined from '@mui/icons-material/KeyOutlined'
import DescriptionOutlined from '@mui/icons-material/DescriptionOutlined'
import WebAssetOutlined from '@mui/icons-material/WebAssetOutlined'
import LibraryBooksOutlined from '@mui/icons-material/LibraryBooksOutlined'
import FilterListOutlined from '@mui/icons-material/FilterListOutlined'
import PolicyOutlined from '@mui/icons-material/PolicyOutlined'
import ListAltOutlined from '@mui/icons-material/ListAltOutlined'
import TimelineOutlined from '@mui/icons-material/TimelineOutlined'
import GroupOutlined from '@mui/icons-material/GroupOutlined'

const navigationIcons: Partial<Record<string, typeof DescriptionOutlined>> = {
	Overview: DashboardOutlined,
	Nodes: DnsOutlined,
	'xDS streams': SyncAltOutlined,
	Snapshots: LayersOutlined,
	Listeners: HearingOutlined,
	Routes: AltRouteOutlined,
	Clusters: HubOutlined,
	Secrets: KeyOutlined,
	'Config dump': DescriptionOutlined,
	'Virtual services': WebAssetOutlined,
	Templates: LibraryBooksOutlined,
	'HTTP filters': FilterListOutlined,
	Policies: PolicyOutlined,
	'Access log configs': ListAltOutlined,
	Events: TimelineOutlined,
	'Access groups': GroupOutlined
}

export default function BetaLayout() {
	const { toggleColorMode } = useContext(ColorModeContext)
	const mode = useTheme().palette.mode
	const dark = mode === 'dark'
	const theme = useMemo(() => createConsoleTheme(mode), [mode])
	const headerBackground = dark ? '#101925' : '#142134'
	const headerText = dark ? '#edf3fc' : '#f2f5fa'
	const nodeId = useMatch('/beta/nodes/:nodeId/*')?.params.nodeId
	const nodeOverview = useMatch('/beta/nodes/:nodeId/overview')
	const comparing = useMatch('/beta/nodes/:nodeId/compare')
	const location = useLocation()
	const navigate = useNavigate()
	const [menuOpen, setMenuOpen] = useState(false)
	const search = useRef<HTMLInputElement>(null)
	const active = nodeOverview ? 'nodes' : nodeId ? 'snapshots' : location.pathname.split('/')[2] || 'nodes'
	const current = sections.find(section => section.slug === active)
	useEffect(() => {
		function shortcut(event: KeyboardEvent) {
			if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
				event.preventDefault()
				search.current?.focus()
			}
		}
		window.addEventListener('keydown', shortcut)
		return () => window.removeEventListener('keydown', shortcut)
	}, [])
	const [count, setCount] = useState<{ section: string; count: number | undefined }>()
	const reportCount = useCallback(
		(value: number | undefined) =>
			setCount(last =>
				last?.section === active && last.count === value ? last : { section: active, count: value }
			),
		[active]
	)
	const navigation = (
		<Box component='nav' aria-label='Console navigation' sx={{ px: 1.25, pb: 2 }}>
			{groups.map(group => (
				<Box key={group} sx={{ pt: 1 }}>
					<Typography
						variant='overline'
						sx={{
							px: 1.25,
							color: 'text.secondary',
							fontSize: 10,
							fontWeight: 650,
							letterSpacing: '.11em'
						}}
					>
						{group}
					</Typography>
					<List dense disablePadding>
						{sections
							.filter(item => item.group === group)
							.map(item => {
								const Icon = navigationIcons[item.title] ?? DescriptionOutlined
								return (
									<ListItemButton
										key={item.slug}
										component={RouterLink}
										to={'/beta/' + item.slug}
										selected={item.slug === active}
										aria-current={item.slug === active ? 'page' : undefined}
										onClick={() => setMenuOpen(false)}
										sx={{
											py: 0.55,
											px: 1.25,
											borderRadius: 1,
											fontSize: 13,
											fontWeight: item.slug === active ? 700 : 400
										}}
									>
										<Icon fontSize='small' sx={{ mr: 1.25, color: 'inherit', fontSize: 16 }} />
										{item.title}
										{item.slug === active &&
											item.slug !== 'overview' &&
											!nodeId &&
											count?.section === active &&
											count.count !== undefined && (
												<Box
													component='span'
													aria-hidden='true'
													title={count.count + ' rows in this view'}
													sx={{
														ml: 'auto',
														pl: 1,
														fontVariantNumeric: 'tabular-nums',
														fontSize: 12,
														color: 'text.secondary'
													}}
												>
													{count.count}
												</Box>
											)}
									</ListItemButton>
								)
							})}
					</List>
				</Box>
			))}
			<Link
				component={RouterLink}
				to='/nodeIDs'
				sx={{ display: { xs: 'block', sm: 'none' }, mt: 2, px: 1.25 }}
				onClick={() => setMenuOpen(false)}
			>
				Classic UI
			</Link>
		</Box>
	)
	return (
		<ThemeProvider theme={theme}>
			<Box sx={{ minHeight: '100vh', color: 'text.primary', bgcolor: 'background.default', typography: 'body1' }}>
				<Stack
					component='header'
					direction='row'
					alignItems='center'
					spacing={1.25}
					sx={{ bgcolor: headerBackground, color: headerText, px: { xs: 2, md: 3 }, height: 56 }}
				>
					<IconButton
						aria-label='Open navigation'
						color='inherit'
						sx={{ display: { md: 'none' } }}
						onClick={() => setMenuOpen(true)}
					>
						<MenuOutlined />
					</IconButton>
					<AccountTreeOutlined
						sx={{
							display: { xs: 'none', sm: 'block' },
							bgcolor: '#9eb9ff',
							color: '#142134',
							borderRadius: '7px',
							p: '4px',
							fontSize: 27
						}}
					/>
					<Link
						component={RouterLink}
						to='/beta/'
						color='inherit'
						underline='none'
						sx={{ fontSize: 17, fontWeight: 650, letterSpacing: '-.04em', whiteSpace: 'nowrap' }}
					>
						Envoy
						<Box component='span' sx={{ display: { xs: 'none', sm: 'inline' } }}>
							{' '}
							Console
						</Box>
					</Link>
					<Box sx={{ flex: 1 }} />
					<Autocomplete
						size='small'
						openOnFocus
						forcePopupIcon={false}
						options={sections}
						value={null}
						groupBy={item => item.group}
						getOptionKey={item => item.slug}
						getOptionLabel={item => item.title}
						onChange={(_, item) => {
							if (item) navigate('/beta/' + item.slug)
						}}
						sx={{
							width: { xs: 155, sm: 240 },
							minWidth: 0,
							'& .MuiOutlinedInput-root': { bgcolor: 'background.paper', color: 'text.primary' },
							'& input::placeholder': { color: 'text.secondary', opacity: 1 },
							'& .MuiSvgIcon-root': { color: 'text.secondary' }
						}}
						renderInput={params => (
							<TextField
								{...params}
								inputRef={search}
								placeholder='Find section'
								inputProps={{ ...params.inputProps, 'aria-label': 'Find section' }}
								InputProps={{
									...params.InputProps,
									startAdornment: (
										<InputAdornment position='start'>
											<SearchOutlined fontSize='small' />
										</InputAdornment>
									)
								}}
							/>
						)}
					/>

					<Link
						component={RouterLink}
						to='/nodeIDs'
						color='inherit'
						sx={{ fontSize: 13, whiteSpace: 'nowrap', display: { xs: 'none', sm: 'inline' } }}
					>
						Classic UI
					</Link>
					<IconButton color='inherit' aria-label='Switch color theme' onClick={toggleColorMode}>
						<ContrastOutlined fontSize='small' />
					</IconButton>
				</Stack>
				<Box
					sx={{
						display: { md: 'grid' },
						gridTemplateColumns: '190px minmax(0, 1fr)',
						minHeight: 'calc(100vh - 56px)'
					}}
				>
					<Box
						sx={{
							display: { xs: 'none', md: 'block' },
							borderRight: 1,
							borderColor: 'divider',
							bgcolor: 'background.paper'
						}}
					>
						{navigation}
					</Box>
					<Drawer open={menuOpen} onClose={() => setMenuOpen(false)} PaperProps={{ sx: { width: 240 } }}>
						{navigation}
					</Drawer>
					<Box
						component='main'
						sx={{
							minWidth: 0,
							width: '100%',
							maxWidth: 1540,
							mx: 'auto',
							px: { xs: 2, md: 'clamp(1.5rem, 3.1vw, 3.25rem)' },
							py: { xs: 2, md: 3 },
							overflowWrap: 'anywhere'
						}}
					>
						<Breadcrumbs aria-label='Breadcrumb' sx={{ mb: 2, fontSize: 13 }}>
							{nodeId && (
								<Link
									component={RouterLink}
									to={nodeOverview ? '/beta/nodes' : '/beta/snapshots'}
									color='inherit'
								>
									{nodeOverview ? 'Nodes' : 'Snapshots'}
								</Link>
							)}
							{comparing && nodeId ? (
								<Link component={RouterLink} to={nodePath(nodeId)} color='inherit'>
									{nodeId}
								</Link>
							) : (
								<span>{nodeId || current?.title}</span>
							)}
							{comparing && <span>Compare</span>}
							{nodeOverview && <span>Overview</span>}
						</Breadcrumbs>
						<Outlet context={reportCount} />
					</Box>
				</Box>
			</Box>
		</ThemeProvider>
	)
}
