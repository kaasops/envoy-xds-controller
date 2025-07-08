import MenuOutlined from '@mui/icons-material/MenuOutlined'
import AppBar from '@mui/material/AppBar'
import Box from '@mui/material/Box'
import { useLocation } from 'react-router-dom'
import useSideBarState from '../../store/sideBarStore'
import RouterBreadcrumbs from '../routerBreadcrumbs/RouterBreadcrumbs'
import ThemeSwitcher from '../themeSwitcher/ThemeSwitcher'
import { CustomToolBar } from './style'

function TopBar() {
	const toggleSideBar = useSideBarState(state => state.toggleSideBar)
	const isOpenSideBar = useSideBarState(state => state.isOpenSideBar)
	const location = useLocation()

	return (
		<AppBar
			position='fixed'
			sx={{
				width: isOpenSideBar ? 'calc(100% - 40px)' : '100%',
				top: 0,
				left: 'auto',
				right: 0,
				zIndex: 5
			}}
		>
			<CustomToolBar>
				<Box display='flex' justifyContent='space-between' alignItems='center' width='100%'>
					<Box display='flex' alignItems='center' gap='15px'>
						<MenuOutlined
							onClick={() => toggleSideBar(!isOpenSideBar)}
							sx={{ cursor: 'pointer', ...(isOpenSideBar && { display: 'none' }) }}
						/>
					</Box>

					<Box
						display='flex'
						justifyContent='flex-start'
						flexGrow={1}
						marginLeft={isOpenSideBar ? '230px' : '50px'}
					>
						<RouterBreadcrumbs location={location} />
					</Box>

					<Box display='flex' justifyContent='flex-end' alignItems='center' marginRight={4}>
						<ThemeSwitcher />
					</Box>
				</Box>
			</CustomToolBar>
		</AppBar>
	)
}

export default TopBar
