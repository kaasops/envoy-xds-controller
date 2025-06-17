import React from 'react'
import { FieldErrors, UseFormRegister } from 'react-hook-form'
import TextField from '@mui/material/TextField'
import { validationRulesVsForm } from '../../utils/helpers/validationRulesVsForm.ts'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'

type nameFieldKeys = Extract<keyof IVirtualServiceForm, 'name'>

interface ITextFieldFormVsProps {
	nameField: nameFieldKeys
	register: UseFormRegister<IVirtualServiceForm>
	errors: FieldErrors<IVirtualServiceForm>
	isDisabled?: boolean | undefined
}

export const TextFieldFormVs: React.FC<ITextFieldFormVsProps> = ({ register, nameField, errors, isDisabled }) => {
	const titleMessage = 'Name'

	return (
		<TextField
			{...register(nameField, {
				validate: validationRulesVsForm[nameField]
			})}
			fullWidth
			disabled={isDisabled}
			error={!!errors[nameField]}
			label={titleMessage}
			helperText={errors[nameField]?.message}
		/>
	)
}
