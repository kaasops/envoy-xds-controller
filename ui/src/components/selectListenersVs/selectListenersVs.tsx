import React from 'react'
import { Control, Controller, FieldErrors } from 'react-hook-form'
import { ListListenersResponse } from '../../gen/listener/v1/listener_pb.ts'
import { validationRulesVsForm } from '../../utils/helpers/validationRulesVsForm.ts'
import { FormControl, InputLabel, MenuItem, Select } from '@mui/material'
import CircularProgress from '@mui/material/CircularProgress'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'

interface ISelectListenersVsProps {
	control: Control<IVirtualServiceForm, any>
	listeners: ListListenersResponse | undefined
	errors: FieldErrors<IVirtualServiceForm>
	isFetching: boolean
	isErrorFetch: boolean
}

export const SelectListenersVs: React.FC<ISelectListenersVsProps> = ({
	control,
	errors,
	listeners,
	isFetching,
	isErrorFetch
}) => {
	return (
		<Controller
			name='listenerUid'
			control={control}
			rules={{
				validate: validationRulesVsForm.listenerUid
			}}
			render={({ field }) => (
				<FormControl fullWidth error={!!errors.listenerUid || isErrorFetch}>
					<InputLabel>
						{errors.listenerUid?.message ??
							(isErrorFetch ? 'Error loading ListenersVs data' : 'Select Listener VS')}
					</InputLabel>
					<Select
						fullWidth
						error={!!errors.listenerUid || isErrorFetch}
						label={
							errors.listenerUid?.message ??
							(isErrorFetch ? 'Error loading ListenersVs data' : 'Select Listener VS')
						}
						value={field.value || ''}
						onChange={e => field.onChange(e.target.value)}
						IconComponent={
							isFetching ? () => <CircularProgress size={20} sx={{ marginRight: 2 }} /> : undefined
						}
						sx={{ '& .MuiSelect-icon': { width: '24px', height: '24px' } }}
					>
						{isErrorFetch && (
							<MenuItem disabled>
								<span style={{ color: 'error' }}>Error loading ListenersVs data</span>
							</MenuItem>
						)}

						{listeners?.items?.map(listener => (
							<MenuItem key={listener.uid} value={listener.uid}>
								{listener.name}
							</MenuItem>
						))}
					</Select>
				</FormControl>
			)}
		/>
	)
}
