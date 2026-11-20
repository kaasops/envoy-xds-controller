import { useContext, useMemo } from 'react'
import { Link as RouterLink, Outlet, useMatch } from 'react-router-dom'
import {
	Box,
	Breadcrumbs,
	IconButton,
	Link,
	List,
	ListItemButton,
	Stack,
	Typography,
	ThemeProvider,
	createTheme,
	useTheme
} from '@mui/material'
import AccountTreeOutlined from '@mui/icons-material/AccountTreeOutlined'
import StorageOutlined from '@mui/icons-material/StorageOutlined'
import ContrastOutlined from '@mui/icons-material/ContrastOutlined'
import { ColorModeContext } from '../../theme'
import { nodePath } from './api'

export default function BetaLayout() {
	const { toggleColorMode } = useContext(ColorModeContext)
	const mode = useTheme().palette.mode
	const dark = mode === 'dark'
	const theme = useMemo(
		() =>
			createTheme({
				palette: {
					mode,
					primary: { main: mode === 'dark' ? '#96c8ff' : '#1e5790' },
					background:
						mode === 'dark'
							? { default: '#141e2a', paper: '#202d3d' }
							: { default: '#f5f7fb', paper: '#ffffff' }
				},
				typography: {
					fontFamily: 'Roboto, sans-serif',
					h4: { fontSize: '1.75rem', fontWeight: 600 },
					button: { textTransform: 'none' }
				}
			}),
		[mode]
	)
	const nodeId = useMatch('/beta/nodes/:nodeId/*')?.params.nodeId
	const comparing = useMatch('/beta/nodes/:nodeId/compare')
	return (
		<ThemeProvider theme={theme}>
			<Box sx={{ minHeight: '100vh', color: 'text.primary', bgcolor: dark ? 'background.default' : '#f5f7fb' }}>
				<Stack
					component='header'
					direction='row'
					alignItems='center'
					spacing={1.25}
					sx={{ bgcolor: '#17283c', color: '#fff', px: { xs: 2, md: 3 }, height: 56 }}
				>
					<AccountTreeOutlined fontSize='small' />
					<Link
						component={RouterLink}
						to='/beta/'
						color='inherit'
						underline='none'
						sx={{ fontSize: 18, fontWeight: 600 }}
					>
						Envoy Console
					</Link>
					<Box sx={{ flex: 1 }} />
					<Link component={RouterLink} to='/nodeIDs' color='inherit' sx={{ fontSize: 13 }}>
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
						component='nav'
						aria-label='Nodes navigation'
						sx={{
							bgcolor: 'background.paper',
							borderRight: { md: 1 },
							borderBottom: { xs: 1, md: 0 },
							borderColor: { xs: 'divider', md: 'divider' },
							p: { xs: 0.75, md: 1.5 }
						}}
					>
						<Typography
							variant='overline'
							sx={{ display: { xs: 'none', md: 'block' }, px: 1.5, mb: 0.75, color: 'text.secondary' }}
						>
							Nodes
						</Typography>
						<List disablePadding sx={{ display: { xs: 'flex', md: 'block' } }}>
							<ListItemButton
								component={RouterLink}
								to='/beta/'
								selected={!nodeId}
								aria-current={!nodeId ? 'page' : undefined}
								sx={{ gap: 1, px: 1.5, borderRadius: 1, fontWeight: !nodeId ? 600 : 400, fontSize: 14 }}
							>
								<StorageOutlined fontSize='small' />
								All nodes
							</ListItemButton>
						</List>
					</Box>
					<Box
						component='main'
						sx={{
							minWidth: 0,
							width: '100%',
							maxWidth: 1640,
							mx: 'auto',
							p: { xs: 2, md: 3.5 },
							overflowWrap: 'anywhere'
						}}
					>
						<Breadcrumbs aria-label='Breadcrumb' sx={{ mb: 2, fontSize: 13 }}>
							{nodeId && (
								<Link component={RouterLink} to='/beta/' color='inherit'>
									Nodes
								</Link>
							)}
							{nodeId && comparing ? (
								<Link component={RouterLink} to={nodePath(nodeId)} color='inherit'>
									{nodeId}
								</Link>
							) : (
								<span>{nodeId || 'Nodes'}</span>
							)}
							{comparing && <span>Compare</span>}
						</Breadcrumbs>
						<Outlet />
					</Box>
				</Box>
			</Box>
		</ThemeProvider>
	)
}
