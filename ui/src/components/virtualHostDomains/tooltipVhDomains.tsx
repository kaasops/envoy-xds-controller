import React from 'react'
import Tooltip from '@mui/material/Tooltip'
import { styleTooltip } from './style.ts'
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined'
import Typography from '@mui/material/Typography'

interface ITooltipVhDomainsProps {
	title?: string
}

export const TooltipVhDomains: React.FC<ITooltipVhDomainsProps> = () => {
	return (
		<Typography fontSize={15} color='gray' mt={1} display='flex' alignItems='center' gap={0.5}>
			Configure the Domains
			<Tooltip
				title='Enter Domain. Press Enter to add it to the list or use key Add Domain.'
				placement='bottom-start'
				enterDelay={300}
				slotProps={{ ...styleTooltip }}
			>
				<InfoOutlinedIcon fontSize='inherit' sx={{ cursor: 'pointer', fontSize: '14px' }} />
			</Tooltip>
		</Typography>
	)
}
