import React, { useState } from 'react'
import { QueryObserverResult, RefetchOptions } from '@tanstack/react-query'
import Dialog from '@mui/material/Dialog'
import { Alert, Button, DialogActions, DialogTitle } from '@mui/material'
import DialogContent from '@mui/material/DialogContent'
import { ListVirtualServicesResponse } from '../../gen/virtual_service/v1/virtual_service_pb.ts'

import { useDeleteVs } from '../../api/grpc/hooks/useVirtualService.ts'
import Snackbar from '@mui/material/Snackbar'

interface IDialogDeleteVSProps {
	serviceName: string
	openDialog: boolean
	setOpenDialog: React.Dispatch<React.SetStateAction<boolean>>
	refetchServices: (
		options?: RefetchOptions | undefined
	) => Promise<QueryObserverResult<ListVirtualServicesResponse, Error>>
	selectedUid: string
	setSelectedUid: React.Dispatch<React.SetStateAction<string>>
}

const DialogDeleteVS: React.FC<IDialogDeleteVSProps> = ({
	serviceName,
	openDialog,
	setOpenDialog,
	refetchServices,
	selectedUid,
	setSelectedUid
}) => {
	const { deleteVirtualService } = useDeleteVs()
	const [openSnackBar, setOpenSnackBar] = useState(false)
	const [snackMessage, setSnackMessage] = useState<string | null>(null)
	const [isError, setIsError] = useState(false)

	const handleConfirmDelete = async () => {
		if (!selectedUid.trim()) return

		try {
			await deleteVirtualService(selectedUid)
			await refetchServices()
			setSnackMessage(`Virtual service: ${serviceName.toUpperCase()} deleted successfully.`)
			setOpenSnackBar(true)
			setIsError(false)
		} catch (error) {
			console.log(error)
			setSnackMessage(`${error}`)
			setIsError(true)
			setOpenSnackBar(true)
		} finally {
			setOpenDialog(false)
			setSelectedUid('')
		}
	}

	const handleCloseDialog = () => {
		setOpenDialog(false)
		setSelectedUid('')
	}

	return (
		<>
			<Dialog open={openDialog} onClose={handleCloseDialog}>
				<DialogTitle>Remove Virtual Service</DialogTitle>
				<DialogContent>Are you sure you want to delete this VS: {serviceName.toUpperCase()}?</DialogContent>
				<DialogActions>
					<Button onClick={handleCloseDialog} color='primary'>
						Cancel
					</Button>
					<Button onClick={handleConfirmDelete} color='error' autoFocus>
						Delete
					</Button>
				</DialogActions>
			</Dialog>

			<Snackbar
				open={openSnackBar}
				autoHideDuration={4000}
				onClose={() => setOpenSnackBar(false)}
				anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
			>
				<Alert
					onClose={() => setOpenSnackBar(false)}
					severity={isError ? 'error' : 'success'}
					variant='filled'
					sx={{ width: '50%' }}
				>
					{snackMessage}
				</Alert>
			</Snackbar>
		</>
	)
}

export default DialogDeleteVS
