import { ChevronLeftOutlined } from '@mui/icons-material'
import { Box, Drawer, IconButton, List, ListItemIcon, ListItemText, Typography, useTheme } from '@mui/material'
import { useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import Logo from '../../assets/icons/envoy-logo.png'
import useSideBarState from '../../store/sideBarStore'
import useColors from '../../utils/hooks/useColors'
import navMenuItems from './navigateItems'
import { DrawerHeader, DrawerLogo, ListItemButtonNav } from './style'

interface ISideBarProps {
	isSmallScreen: boolean
}

function SideBar({ isSmallScreen }: ISideBarProps) {
	const theme = useTheme()
	const { colors } = useColors()

	const toggleSideBar = useSideBarState(state => state.toggleSideBar)
	const isOpenSideBar = useSideBarState(state => state.isOpenSideBar)

	const { pathname } = useLocation()
	const navigate = useNavigate()

	const [activePage, setActivePage] = useState('')

	useEffect(() => {
		setActivePage(pathname)
	}, [pathname])

	useEffect(() => {
		toggleSideBar(isSmallScreen)
	}, [isSmallScreen, toggleSideBar])

	const renderNavMenu = navMenuItems.map(menuItem => (
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
						width: '270px',
						'& .MuiDrawer-paper': {
							color: theme.palette.secondary.main,
							backgroundColor: colors.secondary.DEFAULT,
							boxSizing: 'border-box',
							width: '270px'
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
					</Box>
				</Drawer>
			)}
		</Box>
	)
}

export default SideBar
