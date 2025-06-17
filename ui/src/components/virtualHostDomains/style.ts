import styled from '@mui/material/styles/styled'
import CardContent from '@mui/material/CardContent'

export const styleBox = {
	width: '100%',
	height: '100%',
	border: '1px solid gray',
	borderRadius: 1,
	p: 1.75,
	pt: 0.5,
	display: 'flex',
	flexDirection: 'column',
	gap: 1
}

export const styleTooltip = {
	popper: {
		modifiers: [
			{
				name: 'offset',
				options: {
					offset: [0, -12]
				}
			}
		]
	}
}

export const CustomCardContent = styled(CardContent)(() => ({
	display: 'flex',
	justifyContent: 'space-between',
	width: '100%',
	alignItems: 'center',
	height: '100%',
	padding: 1,

	'&:last-child': {
		paddingBottom: 1
	},

	'& .MuiTypography-root': {
		fontSize: '0.9rem'
	}
}))
