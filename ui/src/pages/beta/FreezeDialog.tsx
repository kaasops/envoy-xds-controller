import { useId, useState } from 'react'
import {
	Alert,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogContentText,
	DialogTitle,
	TextField
} from '@mui/material'
import { useIsMutating, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiError, freezeNode, httpStatus, snapshotQueryKey } from './api'
import type { SnapshotState } from './api'

interface Props {
	nodeId: string
	hasServed?: boolean
	disabled?: boolean
	onClose: () => void
	onFailure?: (message: string) => void
	onFrozen: (next: SnapshotState) => void | Promise<void>
}

function freezeError(error: unknown) {
	const message = apiError(error)
	if (httpStatus(error) === 403 && message === 'snapshot control is turned off on this controller') {
		return 'Writes are disabled on this controller (snapshotControl.enabled=false).'
	}
	if (httpStatus(error) === 403 && message === 'node is not available to you') {
		return 'You do not have access to this node.'
	}
	return message
}

export default function FreezeDialog({ nodeId, hasServed, disabled, onClose, onFailure, onFrozen }: Props) {
	const titleId = useId()
	const descriptionId = useId()
	const client = useQueryClient()
	const key = snapshotQueryKey(nodeId)
	const mutationKey = [...key, 'write']
	const writing = useIsMutating({ mutationKey }) > 0
	const [reason, setReason] = useState('')
	const mutation = useMutation({
		mutationKey,
		retry: false,
		networkMode: 'always',
		mutationFn: (value: string) => freezeNode(nodeId, value),
		onError: error => onFailure?.(freezeError(error)),
		onSuccess: async next => {
			await client.cancelQueries({ queryKey: key, exact: true })
			client.setQueryData(key, next)
			await onFrozen(next)
			onClose()
		}
	})
	const blocked = disabled || writing || mutation.isError || !reason.trim()
	return (
		<Dialog
			open
			fullWidth
			maxWidth='xs'
			aria-labelledby={titleId}
			aria-describedby={descriptionId}
			onClose={() => {
				if (!writing) onClose()
			}}
		>
			<form
				onSubmit={event => {
					event.preventDefault()
					if (!blocked && !client.isMutating({ mutationKey })) mutation.mutate(reason.trim())
				}}
			>
				<DialogTitle id={titleId}>Freeze node</DialogTitle>
				<DialogContent>
					<DialogContentText id={descriptionId} sx={{ overflowWrap: 'anywhere' }}>
						{hasServed === false
							? nodeId +
								' has no served snapshot. New Envoys will receive no configuration while it is frozen.'
							: hasServed
								? nodeId + ' will stay on its served snapshot.'
								: 'Freeze publication to ' +
									nodeId +
									'. Without a served snapshot, new Envoys will wait for configuration.'}{' '}
						The webhook will accept listener address conflicts on this node with a warning.
					</DialogContentText>
					<TextField
						autoFocus
						required
						fullWidth
						multiline
						minRows={2}
						margin='normal'
						label='Reason'
						helperText='Visible in VirtualService status.'
						value={reason}
						disabled={writing || mutation.isError}
						onChange={event => setReason(event.target.value)}
					/>
					{mutation.isError && (
						<Alert severity='error'>
							{freezeError(mutation.error)}
							{httpStatus(mutation.error) !== 403 &&
								' Close this dialog and refresh before trying again.'}
						</Alert>
					)}
				</DialogContent>
				<DialogActions>
					<Button disabled={writing} onClick={onClose}>
						{mutation.isError ? 'Close' : 'Cancel'}
					</Button>
					<Button variant='contained' type='submit' disabled={blocked}>
						Freeze node
					</Button>
				</DialogActions>
			</form>
		</Dialog>
	)
}
