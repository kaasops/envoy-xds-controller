import { styled, Toolbar } from '@mui/material'
import tokens from '../../theme/colors'

export const CustomToolBar = styled(Toolbar)(({ theme }) => {
	const colors = tokens(theme.palette.mode)
	return {
		justifyContent: 'space-between',
		padding: 20,
		width: '100%',
		height: 85,
		color: theme.palette.secondary.contrastText,
		backgroundColor: colors.secondary[300]
	}
})
