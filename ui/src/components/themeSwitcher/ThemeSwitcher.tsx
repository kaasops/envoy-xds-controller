import { DarkMode, LightMode } from '@mui/icons-material'
import { Grid, IconButton, useTheme } from '@mui/material'
import { useContext } from 'react'
import { ColorModeContext } from '../../theme/theme'

function ThemeSwitcher() {
	const theme = useTheme()
	const colorMode: any = useContext(ColorModeContext)

	const changeTheme = () => {
		const color = theme.palette.mode

		if (color === 'dark') {
			localStorage.setItem('themeMod', 'light')
		} else {
			localStorage.setItem('themeMod', 'dark')
		}
		colorMode.toggleColorMode()
	}

	return (
		<Grid onClick={changeTheme}>
			<IconButton sx={{ '&:focus': { outline: 'none' } }}>
				{theme.palette.mode === 'dark' ? <DarkMode /> : <LightMode />}
			</IconButton>
		</Grid>
	)
}

export default ThemeSwitcher
