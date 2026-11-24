import { createTheme } from '@mui/material'
import { useMemo, useState } from 'react'
import { themeSettings } from '../../theme'

export const useThemeMode = () => {
	const [mode, setMode] = useState(() => {
		try {
			return localStorage.getItem('themeMod') === 'dark' ? 'dark' : 'light'
		} catch {
			return 'light'
		}
	})

	const colorMode = useMemo(
		() => ({
			toggleColorMode: () => setMode(prev => (prev === 'light' ? 'dark' : 'light'))
		}),
		[]
	)

	const theme: any = useMemo(() => createTheme(themeSettings(mode)), [mode])

	return [theme, colorMode]
}
