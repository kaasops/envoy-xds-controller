import { Box, useMediaQuery } from '@mui/material'
import { Outlet } from 'react-router-dom'
import SideBar from '../components/sideBar/SideBar'
import TopBar from '../components/topBar/TopBar'

function Layout(): JSX.Element {
	const isSmallScreen = useMediaQuery('(min-width:1441px)')

	return (
		<Box display='flex' justifyContent='space-between' width='100%' height='100%'>
			<SideBar isSmallScreen={isSmallScreen} />
			<Box
				display='flex'
				justifyContent='center'
				flexDirection='column'
				width={`calc(100% - 240px)`}
				flexGrow={1}
				component='main'
				marginTop='85px'
			>
				<TopBar />
				<Outlet />
			</Box>
		</Box>
	)
}

export default Layout
