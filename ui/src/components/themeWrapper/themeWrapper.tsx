import { ReactNode } from 'react'
import { ColorModeContext } from '../../theme/theme.ts'
import { ThemeProvider } from '@emotion/react'
import { CssBaseline } from '@mui/material'

export const ThemedWrapper = ({ children, theme, colorMode }: { children: ReactNode; theme: any; colorMode: any }) => (
	<ColorModeContext.Provider value={colorMode}>
		<ThemeProvider theme={theme}>
			<CssBaseline enableColorScheme />
			{children}
		</ThemeProvider>
	</ColorModeContext.Provider>
)
