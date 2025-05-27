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
	variant?: 'standard' | 'outlined'
	isDisabled?: boolean | undefined
}

export const TextFieldFormVs: React.FC<ITextFieldFormVsProps> = ({
	register,
	nameField,
	errors,
	variant,
	isDisabled
}) => {
	const titleMessage = 'Name'

	return (
		<TextField
			{...register(nameField, {
				required: `The ${titleMessage} field is required`,
				validate: validationRulesVsForm[nameField]
			})}
			fullWidth
			disabled={isDisabled}
			error={!!errors[nameField]}
			label={titleMessage}
			helperText={errors[nameField]?.message}
			variant={variant}
		/>
	)
}
