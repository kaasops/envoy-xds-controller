import { MenuOutlined } from '@mui/icons-material';
import { AppBar, Box, Grid } from '@mui/material';
import { useLocation } from 'react-router-dom';
import useSideBarState from '../../store/sideBarStore';
import RouterBreadcrumbs from '../routerBreadcrumbs/RouterBreadcrumbs';
import ThemeSwitcher from '../themeSwitcher/ThemeSwitcher';
import { CustomToolBar } from './style';

function TopBar() {
    const toggleSideBar = useSideBarState(state => state.toggleSideBar);
    const isOpenSideBar = useSideBarState(state => state.isOpenSideBar);
    const location = useLocation();

    return (
        <AppBar position='fixed' sx={{
            width: isOpenSideBar ? 'calc(100% - 270px)' : '100%',
            top: 0,
            left: 'auto',
            right: 0,
            zIndex: 5
        }}>
            <CustomToolBar>
                <Grid container justifyContent='space-between' alignItems='center'>
                    <Grid item sm={2} lg={2}>
                        <Box display='flex'
                            justifyContent='flex-start'
                            alignItems='center'
                            gap='15px'
                        >
                            <MenuOutlined onClick={() => toggleSideBar(!isOpenSideBar)}
                                sx={{ cursor: 'pointer', ...(isOpenSideBar && { display: 'none' }) }}
                            />

                        </Box>
                    </Grid>
                    <Grid item sm={8} lg={8} display='flex' justifyContent='center'>
                        <RouterBreadcrumbs location={location} />
                    </Grid>
                    <Grid item sm={2} lg={2} display='flex' justifyContent='flex-end' alignItems='center'>
                        <ThemeSwitcher />
                    </Grid>
                </Grid>
            </CustomToolBar>
        </AppBar>
    )
}

export default TopBar