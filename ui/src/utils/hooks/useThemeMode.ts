import { createTheme } from '@mui/material'
import { useEffect, useMemo, useState } from 'react'
import themeSettings from '../../theme/theme'

const useThemeMode = () => {
	const [mode, setMode] = useState('light')

	useEffect(() => {
		const getThemeMod = localStorage.getItem('themeMod')
		const themeMod = getThemeMod ? getThemeMod : 'light'
		setMode(themeMod)
	}, [])

	const colorMode = useMemo(
		() => ({
			toggleColorMode: () => setMode(prev => (prev === 'light' ? 'dark' : 'light'))
		}),
		[]
	)

	const theme: any = useMemo(() => createTheme(themeSettings(mode)), [mode])

	return [theme, colorMode]
}

export default useThemeMode
