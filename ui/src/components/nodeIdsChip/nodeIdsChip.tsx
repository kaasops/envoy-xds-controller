import Chip from '@mui/material/Chip'
import Stack from '@mui/material/Stack'
import React, { useState } from 'react'
import { Box, Menu, MenuItem } from '@mui/material'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'

interface INodeIdsChipProps {
	nodeIsData: string[]
}

export const NodeIdsChip: React.FC<INodeIdsChipProps> = ({ nodeIsData }) => {
	const MAX_VISIBLE = 2
	const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)
	const open = Boolean(anchorEl)

	const handleToggle = (event: React.MouseEvent<HTMLElement>) => {
		setAnchorEl(anchorEl ? null : event.currentTarget)
	}

	const handleClose = () => {
		setAnchorEl(null)
	}

	return (
		<Box
			sx={{
				width: '100%',
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'space-between',
				gap: 1,
				overflow: 'hidden'
			}}
		>
			<Stack direction='row' spacing={1} sx={{ flexGrow: 1, overflow: 'hidden' }}>
				{nodeIsData.slice(0, MAX_VISIBLE).map((item, index) => (
					<Chip key={index} label={item} />
				))}
				{nodeIsData.length > MAX_VISIBLE && (
					<Chip
						label={`+${nodeIsData.length - MAX_VISIBLE} more`}
						variant='outlined'
						sx={{ cursor: 'pointer' }}
						onClick={handleToggle}
						icon={<ExpandMoreIcon />}
					/>
				)}
			</Stack>

			{nodeIsData.length > MAX_VISIBLE && (
				<Menu anchorEl={anchorEl} open={open} onClose={handleClose}>
					{nodeIsData.slice(MAX_VISIBLE).map((item, index) => (
						<MenuItem key={index} onClick={handleClose}>
							<Chip key={index} label={item} />
						</MenuItem>
					))}
				</Menu>
			)}
		</Box>
	)
}
