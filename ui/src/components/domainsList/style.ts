import { ListItemButton, styled } from '@mui/material'
import tokens from '../../theme/colors'

export const styleDomainListBox = {
	width: '100%',
	height: '100%',
	bgcolor: 'background.paper',
	paddingX: 3,
	paddingY: 3,
	borderRadius: 2,
	boxShadow: `0px 0px 8px 0px rgba(0,0,0,0.2),
             0px 0px 0px 0px rgba(0,0,0,0.14),
              0px 1px 3px 0px rgba(0,0,0,0.12)`
}

export const ListItemButtonDomain = styled(ListItemButton)(({ theme }) => {
	const colors = tokens(theme.palette.mode)

	return {
		backgroundColor: colors.primary[400],
		paddingTop: '4px',
		paddingBottom: '4px',
		'&.active': {
			paddingTop: '1px',
			paddingBottom: '1px',
			backgroundColor: colors.primary[900]
		}
	}
})
