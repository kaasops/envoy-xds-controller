import { Box } from '@mui/material'

interface ICustomTabPanelProps {
	children?: React.ReactNode
	index: number
	value: number
}

function CustomTabPanel(props: ICustomTabPanelProps) {
	const { children, value, index, ...other } = props

	return (
		<div
			role='tabpanel'
			hidden={value !== index}
			id={`simple-tabpanel-${index}`}
			aria-labelledby={`simple-tab-${index}`}
			{...other}
		>
			{value === index && (
				<Box sx={{ p: 1 }} height='100%'>
					<Box className='Costyl style Pane'>{children}</Box>
				</Box>
			)}
		</div>
	)
}

export default CustomTabPanel
