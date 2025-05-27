import { Alert, Snackbar } from '@mui/material'
import React, { useEffect, useState } from 'react'
import { FieldErrors } from 'react-hook-form'
import { useTabStore } from '../../store/tabIndexStore.ts'

interface IErrorSnackBarVsProps {
	errors: FieldErrors<any>
	errorUpdateVs: Error | null
	errorCreateVs: Error | null
	errorFillTemplate: Error | null
	isSubmitted: boolean
	isFormReady: boolean
}

export const ErrorSnackBarVs: React.FC<IErrorSnackBarVsProps> = ({
	errors,
	errorUpdateVs,
	errorCreateVs,
	errorFillTemplate,
	isSubmitted,
	isFormReady
}) => {
	const [open, setOpen] = useState(false)
	const [message, setMessage] = useState('')
	const [severity, setSeverity] = useState<'error' | 'warning'>('warning')
	const [autoHideDuration, setAutoHideDuration] = useState<number | null>(3000)
	const setTabIndex = useTabStore(state => state.setTabIndex)

	useEffect(() => {
		if (isSubmitted) {
			if (Object.keys(errors).length > 0 || !isFormReady) {
				const errorMessages = Object.values(errors)
					.map((error: any) => error.message)
					.join('\n')

				setMessage(!isFormReady ? 'Fields Name, NodeIds and Template is required' : errorMessages)
				setSeverity('warning')
				setAutoHideDuration(3000)
				setOpen(true)
				setTabIndex(0)
			} else if (errorFillTemplate) {
				setMessage(errorFillTemplate.message.replace(/^\[unknown]\s*/, ''))
				setSeverity('warning')
				setAutoHideDuration(3000)
				setOpen(true)
			} else if (errorUpdateVs || errorCreateVs) {
				setMessage(errorUpdateVs?.message || errorCreateVs?.message || 'An error occurred')
				setSeverity('error')
				setAutoHideDuration(null)
				setOpen(true)
			}
		}
	}, [errors, errorUpdateVs, errorCreateVs, isSubmitted, isFormReady, errorFillTemplate, setTabIndex])

	const handleClose = () => setOpen(false)

	return (
		<Snackbar
			open={open}
			autoHideDuration={autoHideDuration}
			onClose={handleClose}
			anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
		>
			<Alert onClose={handleClose} severity={severity} variant='filled' sx={{ width: '50%' }}>
				{message}
			</Alert>
		</Snackbar>
	)
}
