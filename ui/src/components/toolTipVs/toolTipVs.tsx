import React from 'react'
import Tooltip from '@mui/material/Tooltip'
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined'
import Typography from '@mui/material/Typography'
import { toolTipVs, toolTipVsTypography } from './style.ts'

interface IToolTipVsProps {
	titleMessage: string
	isDnD?: boolean
	delay?: number
}

export const ToolTipVs: React.FC<IToolTipVsProps> = ({ titleMessage, delay = 800, isDnD }) => {
	return (
		<Typography className='toolTipVs' sx={{ ...toolTipVsTypography }}>
			{!isDnD ? titleMessage : `Configure ${titleMessage}`}
			<Tooltip
				title={
					!isDnD
						? `Select ${titleMessage.slice(0, -1)}.`
						: `Select ${titleMessage}s and arrange them in the desired order.`
				}
				placement='bottom-start'
				enterDelay={delay}
				disableInteractive
				slotProps={{ ...toolTipVs }}
			>
				<InfoOutlinedIcon fontSize='inherit' sx={{ cursor: 'pointer', fontSize: '14px' }} />
			</Tooltip>
		</Typography>
	)
}
