import React, { useState } from 'react'
import {
	Control,
	Controller,
	FieldErrors,
	UseFormClearErrors,
	UseFormSetError,
	UseFormSetValue,
	UseFormWatch
} from 'react-hook-form'
import { Box, TextField } from '@mui/material'
import Stack from '@mui/material/Stack'
import Chip from '@mui/material/Chip'
import { validationRulesVsForm } from '../../utils/helpers/validationRulesVsForm.ts'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'

type nameFieldKeys = Extract<keyof IVirtualServiceForm, 'virtualHostDomains'>

interface IInputWithChipsProps {
	nameField: nameFieldKeys
	watch: UseFormWatch<IVirtualServiceForm>
	setValue: UseFormSetValue<IVirtualServiceForm>
	control: Control<IVirtualServiceForm>
	errors: FieldErrors<IVirtualServiceForm>
	setError: UseFormSetError<IVirtualServiceForm>
	clearErrors: UseFormClearErrors<IVirtualServiceForm>
}

export const InputWithChips: React.FC<IInputWithChipsProps> = ({
	nameField,
	watch,
	setValue,
	control,
	errors,
	setError,
	clearErrors
}) => {
	const [inputValue, setInputValue] = useState('')
	const watchFiled = watch(nameField)
	const titleMessage = 'Domains'

	const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		setInputValue(e.target.value)
	}

	const handleKeyUp = (event: React.KeyboardEvent<HTMLInputElement>) => {
		if (event.key === 'Enter' || event.key === ',') {
			event.preventDefault()

			const trimmedValue = inputValue.trim()
			if (!trimmedValue) return

			const cleanedValue = trimmedValue.replace(/,$/, '')
			const currentTags = watch(nameField) || []

			if (!/^[a-zA-Z0-9_-]+$/.test(cleanedValue)) {
				setError(nameField, {
					type: 'manual',
					message: `${titleMessage} must contain only letters, numbers, hyphens, and underscores`
				})
				return
			}

			if (!currentTags.includes(cleanedValue)) {
				const updatedTags = [...currentTags, cleanedValue]
				setValue(nameField, updatedTags, { shouldValidate: true })
				clearErrors(nameField)
			}

			setInputValue('')
		}
	}

	const handleDeleteChip = (chipToDelete: string) => {
		const updatedTags = watchFiled.filter(elem => elem !== chipToDelete)
		setValue(nameField, updatedTags)
	}

	return (
		<Box display='flex' flexDirection='column'>
			<Stack direction='row' spacing={1} flexWrap='wrap' mb={watchFiled.length > 0 ? 1.3 : 0}>
				{watchFiled.map((elem, index) => (
					<Chip key={index} label={elem} onDelete={() => handleDeleteChip(elem)} />
				))}
			</Stack>

			<Controller
				name={nameField}
				control={control}
				rules={{
					validate: validationRulesVsForm[nameField]
				}}
				render={() => (
					<TextField
						value={inputValue}
						onChange={handleInputChange}
						onKeyUp={handleKeyUp}
						onKeyDown={event => {
							if (event.key === 'Enter') event.preventDefault() // Блокируем отправку формы
						}}
						fullWidth
						size='small'
						placeholder='Add tags. Press Enter or comma to add tags'
						error={!!errors[nameField]}
						label={
							errors[nameField]?.message ??
							`Enter ${titleMessage} (press Enter or use commas to separate)`
						}
					/>
				)}
			/>
		</Box>
	)
}
