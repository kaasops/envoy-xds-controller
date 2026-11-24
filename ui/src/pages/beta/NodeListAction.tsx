import { useState } from 'react'
import { Button } from '@mui/material'
import { Link } from 'react-router-dom'
import { nodePath } from './api'
import type { SnapshotState } from './api'
import PauseDialog from './PauseDialog'

interface Props {
	nodeId: string
	paused: boolean
	disabled?: boolean
	onPaused: (next: SnapshotState) => void | Promise<void>
}

export default function NodeListAction({ nodeId, paused, disabled, onPaused }: Props) {
	const [open, setOpen] = useState(false)
	return (
		<>
			{paused ? (
				<Button component={Link} to={nodePath(nodeId) + '/compare?from=served&to=pending'} size='small'>
					Resume…
				</Button>
			) : (
				<Button size='small' disabled={disabled} onClick={() => setOpen(true)}>
					Pause…
				</Button>
			)}
			{open && (
				<PauseDialog
					nodeId={nodeId}
					disabled={disabled || paused}
					onClose={() => setOpen(false)}
					onPaused={onPaused}
				/>
			)}
		</>
	)
}
