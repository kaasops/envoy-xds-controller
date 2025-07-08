import { grey } from '@mui/material/colors'
import darkScrollbar from '@mui/material/darkScrollbar'
import { Context, createContext } from 'react'
import tokens from './colors'

export const themeSettings: any = (mode: string) => {
	const colors = tokens(mode)
	const os = navigator.platform
	const isWindows = os === 'Win32'

	return {
		components: {
			MuiCssBaseline: {
				styleOverrides: {
					body: isWindows
						? mode === 'dark'
							? darkScrollbar()
							: darkScrollbar({
									track: grey[200],
									thumb: grey[400],
									active: grey[400]
								})
						: undefined,
					...(isWindows
						? {
								scrollbarWidth: 'thin',
								'*::-webkit-scrollbar': {
									width: '0.9em',
									height: '0.9em'
								}
							}
						: undefined)
				}
			}
		},
		palette: {
			mode: mode,
			...(mode === 'dark'
				? {
						action: {
							active: colors.blue.DEFAULT
						},
						background: {
							default: colors.primary.DEFAULT,
							dark: colors.secondary.DEFAULT,
							paper: colors.primary[700]
						},
						border: {
							DEFAULT: colors.gray[50]
						},
						primary: {
							main: colors.blue.DEFAULT,
							secondary: colors.blue[50],
							dark: colors.blue[50]
						},
						secondary: {
							main: colors.secondary.DEFAULT,
							contrastText: colors.gray[100]
						},
						neutral: {
							main: colors.gray[100],
							dark: colors.primary[500],
							light: colors.white.DEFAULT
						},
						error: {
							main: colors.red
						},
						success: {
							main: colors.green
						},
						inherit: {
							main: colors.blue.DEFAULT
						},
						info: {
							main: colors.white[100]
						}
					}
				: {
						action: {
							active: colors.blue.DEFAULT
						},
						background: {
							default: colors.primary.DEFAULT,
							secondary: colors.secondary.DEFAULT,
							paper: colors.primary[700]
						},
						border: {
							DEFAULT: colors.gray[50]
						},
						primary: {
							main: colors.blue.DEFAULT,
							dark: colors.blue[50]
						},
						secondary: {
							main: colors.secondary.DEFAULT,
							contrastText: colors.gray[100]
						},
						neutral: {
							main: colors.gray[100],
							dark: colors.primary[100],
							light: colors.white[200]
						},
						error: {
							main: colors.red
						},
						success: {
							main: colors.green
						},
						inherit: {
							main: colors.blue.DEFAULT
						},
						info: {
							main: colors.white[100]
						}
					})
		},
		typography: {
			fontFamily: ['Roboto', 'sans-serif'].join(','),
			fontSize: 14,
			fontWeight: 400,
			h1: {
				fontFamily: ['Roboto', 'sans-serif'].join(','),
				fontSize: 40,
				fontWeight: 700
			},
			h2: {
				fontFamily: ['Roboto', 'sans-serif'].join(','),
				fontSize: 35,
				fontWeight: 700
			},
			h3: {
				fontFamily: ['Roboto', 'sans-serif'].join(','),
				fontSize: 30,
				fontWeight: 700
			},
			h4: {
				fontFamily: ['Roboto', 'sans-serif'].join(','),
				fontSize: 25,
				fontWeight: 600
			},
			p: {
				fontFamily: ['Roboto', 'sans-serif'].join(','),
				fontSize: 20,
				fontWeight: 500
			}
		},
		breakpoints: {
			values: {
				xs: 0,
				md: 1400,
				lg: 1900,
				xl: 2301
			}
		}
	}
}

interface IColorModeContext {
	toggleColorMode: () => void
}

export const ColorModeContext: Context<IColorModeContext> = createContext({
	toggleColorMode: () => {}
})
