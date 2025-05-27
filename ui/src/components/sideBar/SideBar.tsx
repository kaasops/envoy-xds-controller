// MUI icons
import ChevronLeftOutlined from '@mui/icons-material/ChevronLeftOutlined'
import LogoutOutlined from '@mui/icons-material/LogoutOutlined'

// MUI components
import Box from '@mui/material/Box'
import Divider from '@mui/material/Divider'
import Drawer from '@mui/material/Drawer'
import IconButton from '@mui/material/IconButton'
import List from '@mui/material/List'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import Typography from '@mui/material/Typography'
import useTheme from '@mui/material/styles/useTheme'

import { useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import Logo from '../../assets/icons/envoy-logo.png'
import useSideBarState from '../../store/sideBarStore'
import { useColors } from '../../utils/hooks/useColors'
import navMenuItems from './navigateItems'
import { DrawerHeader, DrawerLogo, ListItemButtonNav } from './style'
import { useAuth } from 'react-oidc-context'
import { usePermissionsStore } from '../../store/permissionsStore.ts'

interface ISideBarProps {
	isSmallScreen: boolean
}

function SideBar({ isSmallScreen }: ISideBarProps) {
	const theme = useTheme()
	const { colors } = useColors()
	const auth = useAuth()

	const toggleSideBar = useSideBarState(state => state.toggleSideBar)
	const isOpenSideBar = useSideBarState(state => state.isOpenSideBar)
	const hasAccess = usePermissionsStore(state => state.hasAccess)

	const { pathname } = useLocation()
	const navigate = useNavigate()

	const [activePage, setActivePage] = useState('')

	useEffect(() => {
		setActivePage(pathname)
	}, [pathname])

	useEffect(() => {
		toggleSideBar(isSmallScreen)
	}, [isSmallScreen, toggleSideBar])

	const renderNavMenu = navMenuItems
		.filter(item => !item.requiresAccess || hasAccess)
		.map(menuItem => (
			<ListItemButtonNav
				key={menuItem.id}
				onClick={() => navigate(menuItem.path)}
				className={activePage.includes(menuItem.path) ? 'active' : ''}
			>
				<ListItemIcon sx={{ color: colors.gray[400] }}>{menuItem.icon}</ListItemIcon>
				<ListItemText primary={menuItem.name} />
			</ListItemButtonNav>
		))

	return (
		<Box component={'nav'}>
			{isOpenSideBar && (
				<Drawer
					variant='persistent'
					anchor='left'
					open={isOpenSideBar}
					onClose={() => toggleSideBar(false)}
					sx={{
						width: '240px',
						'& .MuiDrawer-paper': {
							color: theme.palette.secondary.main,
							backgroundColor: colors.secondary.DEFAULT,
							boxSizing: 'border-box',
							width: '240px'
						}
					}}
				>
					<Box className='drawerHeader' sx={{ borderBottom: `1px solid ${colors.primary[100]}` }}>
						<DrawerHeader>
							<Box display='flex' gap={2} sx={{ cursor: 'pointer' }} onClick={() => navigate('/nodeIDs')}>
								<img src={Logo} alt='logo' width={'35%'} />
								<DrawerLogo>
									<Typography variant='h4'>envoy</Typography>
									<Typography fontSize={11} sx={{ color: 'white', lineHeight: '1' }}>
										xDS controller
									</Typography>
								</DrawerLogo>
							</Box>

							{isOpenSideBar && (
								<IconButton
									onClick={() => toggleSideBar(!isOpenSideBar)}
									color={theme.palette.mode === 'light' ? 'info' : 'default'}
								>
									<ChevronLeftOutlined sx={{ color: '#DBDCDD' }} />
								</IconButton>
							)}
						</DrawerHeader>

						<List>{renderNavMenu}</List>
						<Divider sx={{ mt: 5 }} />
						<List>
							<ListItemButtonNav key={1} onClick={() => void auth.removeUser()}>
								<ListItemIcon sx={{ color: colors.secondary.DEFAULT }}>
									<LogoutOutlined />
								</ListItemIcon>
								<ListItemText>LogOut</ListItemText>
							</ListItemButtonNav>
						</List>
					</Box>
				</Drawer>
			)}
		</Box>
	)
}

export default SideBar
