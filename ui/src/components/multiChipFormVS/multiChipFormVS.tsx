import React from 'react'
import Box from '@mui/material/Box'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'

import { Control, FieldErrors, UseFormClearErrors, UseFormSetError, UseFormSetValue } from 'react-hook-form'
import { AutocompleteChipVs } from '../autocompleteChipVs/autocompleteChipVs.tsx'
import { styleBox, styleTooltip } from './style.ts'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined'

type nameFieldKeys = Extract<keyof IVirtualServiceForm, 'nodeIds' | 'virtualHostDomains'>

interface IMultiChipFormVSProps {
	nameFields: nameFieldKeys
	setValue: UseFormSetValue<IVirtualServiceForm>
	control: Control<IVirtualServiceForm>
	errors: FieldErrors<IVirtualServiceForm>
	setError: UseFormSetError<IVirtualServiceForm>
	clearErrors: UseFormClearErrors<IVirtualServiceForm>
}

export const MultiChipFormVS: React.FC<IMultiChipFormVSProps> = ({
	nameFields,
	errors,
	setError,
	clearErrors,
	control,
	setValue
}) => {
	const titleMessage = nameFields === 'nodeIds' ? 'NodeIDs' : 'Domains'

	return (
		<Box sx={{ ...styleBox }}>
			<Typography fontSize={15} color='gray' mt={1} display='flex' alignItems='center' gap={0.5}>
				Configure the {titleMessage}
				<Tooltip
					title={`Enter ${titleMessage.slice(0, -1)}. Press Enter to add it to the list.`}
					placement='bottom-start'
					enterDelay={800}
					disableInteractive
					slotProps={{ ...styleTooltip }}
				>
					<InfoOutlinedIcon fontSize='inherit' sx={{ cursor: 'pointer', fontSize: '14px' }} />
				</Tooltip>
			</Typography>

			<AutocompleteChipVs
				nameField={nameFields}
				control={control}
				setValue={setValue}
				errors={errors}
				setError={setError}
				clearErrors={clearErrors}
				variant={'standard'}
			/>
		</Box>
	)
}
