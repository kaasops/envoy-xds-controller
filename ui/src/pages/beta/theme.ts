import { createTheme, type PaletteMode } from '@mui/material/styles'

const fontFamily = "Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
export const consoleMonospace = "'IBM Plex Mono', ui-monospace, 'SFMono-Regular', Menlo, Consolas, monospace"

export function createConsoleTheme(mode: PaletteMode) {
	const dark = mode === 'dark'
	const border = dark ? '#566880' : '#b9c5d3'
	const tableHeader = dark ? '#202c3d' : '#f8f9fb'
	const secondaryText = dark ? '#a6b5ca' : '#52647d'
	const divider = dark ? '#334155' : '#dce2e9'
	const selected = dark ? '#293d61' : '#e9efff'
	const accent = dark ? '#a7c1ff' : '#204ebc'
	const success = { main: dark ? '#8ddac2' : '#116149', background: dark ? '#203e36' : '#e7f4ef' }
	const warning = { main: dark ? '#f2c77a' : '#80500c', background: dark ? '#443925' : '#faf1dd' }
	const error = { main: dark ? '#ffb4ad' : '#a03631', background: dark ? '#462c31' : '#fbebea' }
	const neutral = { main: dark ? '#c0cbdb' : '#40526a', background: dark ? '#28364a' : '#edf1f7' }

	const chipColors: Partial<Record<string, typeof neutral>> = { success, warning, error, default: neutral }

	return createTheme({
		palette: {
			mode,
			primary: { main: dark ? '#9eb9ff' : '#2854c4', contrastText: dark ? '#102140' : '#ffffff' },
			text: { primary: dark ? '#e5edf8' : '#152237', secondary: secondaryText },
			background: { default: dark ? '#121b28' : '#f3f5f8', paper: dark ? '#192332' : '#ffffff' },
			divider,
			action: { hover: neutral.background, selected, active: secondaryText },
			info: { main: dark ? '#bed1ff' : '#214fb9' },
			success: { main: success.main },
			warning: { main: warning.main },
			error: { main: error.main }
		},
		shape: { borderRadius: 7 },
		typography: {
			fontFamily,
			body1: { fontSize: '0.875rem', lineHeight: 1.48 },
			body2: { fontSize: '0.8125rem', lineHeight: 1.48 },
			h4: { fontSize: '2.125rem', lineHeight: 1.18, letterSpacing: '-.04em', fontWeight: 700 },
			h6: { fontSize: '1.0625rem', lineHeight: 1.35, letterSpacing: '-.018em', fontWeight: 650 },
			subtitle1: { fontSize: '1.0625rem', fontWeight: 600 },
			subtitle2: { fontSize: '0.9375rem', fontWeight: 600 },
			button: { fontSize: '0.8125rem', fontWeight: 600, textTransform: 'none', letterSpacing: 0 }
		},
		components: {
			MuiPaper: {
				styleOverrides: {
					outlined: { borderRadius: 10 },
					root: { fontFamily, backgroundImage: 'none', '& code, & pre': { fontFamily: consoleMonospace } }
				}
			},
			MuiTableCell: {
				styleOverrides: {
					root: { fontSize: '0.8125rem', borderBottomColor: divider },
					head: {
						borderBottomColor: divider,
						fontSize: '0.75rem',
						fontWeight: 650,
						color: secondaryText,
						backgroundColor: tableHeader
					}
				}
			},
			MuiLink: {
				defaultProps: { underline: 'hover' },
				styleOverrides: {
					root: ({ ownerState }) => ({
						color: !ownerState.color || ownerState.color === 'primary' ? accent : undefined,
						'&:focus-visible': { outline: `2px solid ${accent}`, outlineOffset: 2 }
					})
				}
			},
			MuiOutlinedInput: { styleOverrides: { notchedOutline: { borderColor: border } } },
			MuiButton: { styleOverrides: { outlinedPrimary: { borderColor: border, color: accent } } },
			MuiListItemButton: {
				styleOverrides: {
					root: {
						color: secondaryText,
						'&.Mui-selected': {
							color: accent,
							backgroundColor: selected,
							boxShadow: `inset 3px 0 0 ${accent}`
						},
						'&.Mui-selected:hover': { backgroundColor: selected }
					}
				}
			},
			MuiChip: {
				styleOverrides: {
					outlined: ({ ownerState }) => {
						const signal = chipColors[ownerState.color ?? 'default']
						if (!signal) return {}
						return {
							backgroundColor: signal.background,
							color: signal.main,
							borderColor: 'transparent',
							borderRadius: 5,
							fontSize: '0.75rem',
							fontWeight: 600,
							'&[data-status] .MuiChip-label': {
								display: 'inline-flex',
								alignItems: 'baseline',
								gap: 5,
								'&::before': {
									content: '""',
									flex: '0 0 6px',
									width: 6,
									height: 6,
									alignSelf: 'center',
									borderRadius: '50%',
									backgroundColor: 'currentColor'
								}
							}
						}
					}
				}
			}
		}
	})
}
