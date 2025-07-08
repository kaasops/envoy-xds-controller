import React from 'react'
import { FieldErrors, UseFormRegister } from 'react-hook-form'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import { validationRulesVsForm } from '../../utils/helpers/validationRulesVsForm.ts'
import TextField from '@mui/material/TextField'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'

type nameFieldKeys = Extract<keyof IVirtualServiceForm, 'description'>

interface ITextAreaFomVsProps {
	nameField: nameFieldKeys
	register: UseFormRegister<IVirtualServiceForm>
	errors: FieldErrors<IVirtualServiceForm>
}

export const TextAreaFomVs: React.FC<ITextAreaFomVsProps> = ({ register, nameField, errors }) => {
	const titleMessage = 'Description'
	const readMode = useViewModeStore(state => state.viewMode) === 'read'
	return (
		<TextField
			{...register(nameField, {
				validate: validationRulesVsForm[nameField]
			})}
			fullWidth
			disabled={readMode}
			error={!!errors[nameField]}
			label={titleMessage}
			helperText={errors[nameField]?.message}
			multiline
			maxRows={5}
		/>
	)
}
