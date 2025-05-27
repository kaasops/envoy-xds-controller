import React from 'react'
import {
	Control,
	Controller,
	FieldErrors,
	useFieldArray,
	UseFormClearErrors,
	UseFormGetValues,
	UseFormRegister
} from 'react-hook-form'
import { TemplateOptionModifier } from '../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import FormControl from '@mui/material/FormControl'
import FormHelperText from '@mui/material/FormHelperText'
import IconButton from '@mui/material/IconButton'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Select from '@mui/material/Select'
import TextField from '@mui/material/TextField'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'

import DeleteIcon from '@mui/icons-material/Delete'
import { validationRulesVsForm } from '../../utils/helpers/validationRulesVsForm.ts'
import { styleBox, styleTooltip } from './style.ts'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'

interface ITemplateOptionsFormVsProps {
	register: UseFormRegister<IVirtualServiceForm>
	control: Control<IVirtualServiceForm>
	errors: FieldErrors<IVirtualServiceForm>
	getValues: UseFormGetValues<IVirtualServiceForm>
	clearErrors: UseFormClearErrors<IVirtualServiceForm>
}

export const TemplateOptionsFormVs: React.FC<ITemplateOptionsFormVsProps> = ({
	register,
	control,
	errors,
	getValues,
	clearErrors
}) => {
	const { fields, append, remove } = useFieldArray({
		control,
		name: 'templateOptions'
	})

	const readMode = useViewModeStore(state => state.viewMode) === 'read'

	const enumOptionsModifier = Object.entries(TemplateOptionModifier)
		.filter(([_, value]) => typeof value === 'number')
		.map(([key, value]) => ({
			label: key.toUpperCase(),
			value: value
		}))

	const validateTemplateOption = (fieldName: 'field' | 'modifier', index: number, value: string | number) => {
		const fieldValue = getValues(`templateOptions.${index}.field`)
		const modifierValue = getValues(`templateOptions.${index}.modifier`)

		const templateOption = {
			field: fieldName === 'field' ? String(value) : String(fieldValue), // Приводим к строке
			modifier: fieldName === 'modifier' ? Number(value) : Number(modifierValue) // Приводим к числу
		}

		const result = validationRulesVsForm.templateOptions([templateOption])

		if (fieldName === 'field' && modifierValue && result === true) {
			clearErrors(`templateOptions.${index}.modifier`)
		}

		if (fieldName === 'modifier' && fieldValue && result === true) {
			clearErrors(`templateOptions.${index}.field`)
		}

		return result
	}

	return (
		<Box sx={{ ...styleBox }}>
			<Typography fontSize={15} color='gray' mt={1} display='flex' alignItems='center' gap={0.5}>
				Template options
				<Tooltip
					title={
						<>
							<p>Specify the property path and select the modifier parameter.</p>
							<p>
								<strong>Modifiers:</strong>
							</p>
							<ul>
								<li>
									<strong>merge</strong> (default) - Merges objects, appends to lists
								</li>
								<li>
									<strong>replace</strong> - Completely replaces objects or lists
								</li>
								<li>
									<strong>delete</strong> - Removes the field from configuration
								</li>
							</ul>
							<p>
								<strong>Example:</strong> path - virtualHost.domains, modifier - replace
							</p>
						</>
					}
					placement='bottom-start'
					enterDelay={500}
					slotProps={{ ...styleTooltip }}
				>
					<InfoOutlinedIcon fontSize='inherit' sx={{ cursor: 'pointer', fontSize: '14px' }} />
				</Tooltip>
			</Typography>

			<Box display='flex' flexDirection='column' gap={2}>
				{fields.map((field, index) => (
					<Box key={field.id} display='flex' gap={2}>
						<TextField
							{...register(`templateOptions.${index}.field`, {
								validate: value => validateTemplateOption('field', index, value)
							})}
							key={field.id}
							fullWidth
							disabled={readMode}
							error={!!errors.templateOptions?.[index]?.field}
							label='Path'
							helperText={errors.templateOptions?.[index]?.field?.message}
						/>
						<Controller
							name={`templateOptions.${index}.modifier` as const}
							control={control}
							rules={{
								validate: value => validateTemplateOption('modifier', index, value)
							}}
							render={({ field }) => (
								<FormControl fullWidth error={!!errors.templateOptions?.[index]?.modifier}>
									<InputLabel>Modifier</InputLabel>
									<Select
										{...field}
										value={field.value === 0 ? '' : field.value}
										error={!!errors.templateOptions?.[index]?.modifier}
										fullWidth
										disabled={readMode}
										label='Modifier'
									>
										{enumOptionsModifier
											.filter(option => option.value !== 0)
											.map(option => (
												<MenuItem key={option.value} value={option.value}>
													{option.label}
												</MenuItem>
											))}
									</Select>
									<FormHelperText>
										{errors.templateOptions?.[index]?.modifier?.message}
									</FormHelperText>
								</FormControl>
							)}
						/>
						<IconButton size='large' onClick={() => remove(index)} color='error' disabled={readMode}>
							<DeleteIcon color={readMode ? 'disabled' : 'primary'} />
						</IconButton>
					</Box>
				))}
			</Box>
			<Button onClick={() => append({ field: '', modifier: 0 })} variant='contained' disabled={readMode}>
				Add Template option
			</Button>
		</Box>
	)
}
