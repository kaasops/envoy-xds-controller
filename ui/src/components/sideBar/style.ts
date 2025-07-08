import { Box, ListItemButton, styled } from '@mui/material'
import tokens from '../../theme/colors'

export const DrawerHeader = styled(Box)(({ theme }) => {
	const colors = tokens(theme.palette.mode)
	return {
		display: 'flex',
		alignItems: 'center',
		padding: '18px 20px',
		gap: 15,
		backgroundColor: colors.secondary[100],
		height: 85
	}
})

export const DrawerLogo = styled(Box)({
	display: 'flex',
	flexDirection: 'column',
	color: '#E7E8EB'
})

export const ListItemButtonNav = styled(ListItemButton)(({ theme }) => {
	const colors = tokens(theme.palette.mode)

	return {
		height: 50,
		backgroundColor: colors.primary[100],
		'& .MuiTypography-root': {
			color: colors.gray[400]
		},
		'& .MuiSvgIcon-root': {
			color: colors.gray[400],
			fill: colors.gray[400],
			width: 40,
			height: 35
		},
		'&.active': {
			backgroundColor: colors.primary[200],
			'& .MuiTypography-root': {
				color: colors.gray[200]
			},
			'& .MuiSvgIcon-root': {
				color: colors.blue.DEFAULT,
				fill: colors.blue.DEFAULT
			},
			'&:hover': {
				cursor: 'auto'
			}
		},

		'&:hover': {
			backgroundColor: colors.primary[200],
			cursor: 'pointer',
			'& .MuiSvgIcon-root': {
				color: colors.blue.DEFAULT
			},
			'& .MuiTypography-root': {
				color: colors.gray[200]
			}
		}
	}
})
