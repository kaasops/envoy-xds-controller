import React from 'react'
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined'
import Tooltip from '@mui/material/Tooltip'
import { styleTooltip } from './style.ts'
import Typography from '@mui/material/Typography'

export const TooltipTemplateOptions: React.FC = () => {
	return (
		<Typography fontSize={15} color='gray' mt={1} display='flex' alignItems='center' gap={0.5}>
			Template options
			<Tooltip
				title={
					<>
						<p>Specify the property path and select the modifier parameter.</p>
						<p>
							<strong>Modifiers:</strong>
						</p>
						<ul>
							<li>
								<strong>merge</strong> (default) - Merges objects, appends to lists
							</li>
							<li>
								<strong>replace</strong> - Completely replaces objects or lists
							</li>
							<li>
								<strong>delete</strong> - Removes the field from configuration
							</li>
						</ul>
						<p>
							<strong>Example:</strong> path - virtualHost.domains, modifier - replace
						</p>
					</>
				}
				placement='bottom-start'
				enterDelay={500}
				slotProps={{ ...styleTooltip }}
			>
				<InfoOutlinedIcon fontSize='inherit' sx={{ cursor: 'pointer', fontSize: '14px' }} />
			</Tooltip>
		</Typography>
	)
}
