import { useState } from 'react'
import { Button } from '@mui/material'
import { Link } from 'react-router-dom'
import { nodePath } from './api'
import type { SnapshotState } from './api'
import FreezeDialog from './FreezeDialog'

interface Props {
	nodeId: string
	frozen: boolean
	disabled?: boolean
	onFrozen: (next: SnapshotState) => void | Promise<void>
}

export default function NodeListAction({ nodeId, frozen, disabled, onFrozen }: Props) {
	const [open, setOpen] = useState(false)
	return (
		<>
			{frozen ? (
				<Button component={Link} to={nodePath(nodeId) + '/compare?from=served&to=built'} size='small'>
					Unfreeze…
				</Button>
			) : (
				<Button size='small' disabled={disabled} onClick={() => setOpen(true)}>
					Freeze…
				</Button>
			)}
			{open && (
				<FreezeDialog
					nodeId={nodeId}
					disabled={disabled || frozen}
					onClose={() => setOpen(false)}
					onFrozen={onFrozen}
				/>
			)}
		</>
	)
}
