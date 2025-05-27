import React from 'react'
import { Control, Controller, FieldErrors } from 'react-hook-form'
import Box from '@mui/material/Box'
import FormControl from '@mui/material/FormControl'
import FormControlLabel from '@mui/material/FormControlLabel'
import Radio from '@mui/material/Radio'
import RadioGroup from '@mui/material/RadioGroup'
import Typography from '@mui/material/Typography'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'

interface IRemoteAddrFormVsProps {
	nameField: Extract<keyof IVirtualServiceForm, 'useRemoteAddress'>
	control: Control<IVirtualServiceForm>
	errors: FieldErrors<IVirtualServiceForm>
}

export const RemoteAddrFormVs: React.FC<IRemoteAddrFormVsProps> = ({ nameField, control, errors }) => {
	const readMode = useViewModeStore(state => state.viewMode) === 'read'

	return (
		<Box
			display='flex'
			justifyContent='center'
			sx={{
				width: '100%',
				border: '1px solid gray',
				borderRadius: 1,
				p: 2,
				pt: 0.5,
				display: 'flex',
				flexDirection: 'column',
				gap: 2
			}}
		>
			<Typography fontSize={15} color='gray' mt={1}>
				Use Remote Address
			</Typography>
			<Controller
				name={nameField}
				control={control}
				render={({ field }) => (
					<FormControl error={!!errors[nameField]} sx={{ alignItems: 'center' }} disabled={readMode}>
						<RadioGroup
							value={field.value ?? ''}
							onChange={e => {
								const value =
									e.target.value === 'true' ? true : e.target.value === 'false' ? false : null
								field.onChange(value)
							}}
						>
							<Box display='flex' justifyContent='center' alignItems='center'>
								<FormControlLabel value='true' control={<Radio />} label='True' />
								<FormControlLabel value='false' control={<Radio />} label='False' />
								<FormControlLabel value='' control={<Radio />} label='Default (Null)' />
							</Box>
						</RadioGroup>
					</FormControl>
				)}
			/>
		</Box>
	)
}
