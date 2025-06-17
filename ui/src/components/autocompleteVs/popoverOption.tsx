import React from 'react'
import { ItemVs } from './autocompleteVs.tsx'
import { ClickAwayListener, Popper } from '@mui/material'
import Box from '@mui/material/Box'
import { ItemDnd } from '../dNdSelectFormVs/dNdSelectFormVs.tsx'
import { AutocompleteCodeEditorVs } from './autocompleteCodeEditorVs.tsx'

interface IPopoverOptionProps {
	anchorEl: HTMLElement | null
	option: ItemVs | ItemDnd | null
	onClose: () => void
}

export const PopoverOption: React.FC<IPopoverOptionProps> = ({ anchorEl, option, onClose }) => {
	const isOpen = option && anchorEl && document.body.contains(anchorEl)

	if (!isOpen || !anchorEl) return null

	return (
		<ClickAwayListener onClickAway={onClose}>
			<Popper
				open={Boolean(anchorEl && option)}
				anchorEl={anchorEl}
				placement='right'
				disablePortal={false}
				style={{ zIndex: 1300 }}
				onClick={e => e.stopPropagation()}
			>
				{option && (
					<Box
						sx={{
							bgcolor: 'background.paper',
							boxShadow: 3,
							borderRadius: 1,
							p: 1,
							maxWidth: '90vw',
							overflow: 'auto',
							display: 'inline-block'
						}}
					>
						<AutocompleteCodeEditorVs raw={option.raw} />
					</Box>
				)}
			</Popper>
		</ClickAwayListener>
	)
}
