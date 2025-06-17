import React from 'react'
import { AutocompleteRenderInputParams, TextField } from '@mui/material'
import Typography from '@mui/material/Typography'
import CircularProgress from '@mui/material/CircularProgress'
import { nameFieldKeys } from './autocompleteVs'
import { FieldErrors } from 'react-hook-form'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import { ListenerListItem } from '../../gen/listener/v1/listener_pb.ts'
import { VirtualServiceTemplateListItem } from '../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'
import { AccessLogConfigListItem } from '../../gen/access_log_config/v1/access_log_config_pb.ts'

interface IRenderInputFieldProps {
	className: 'autocompleteVs' | 'dndAutocomplete'
	params: AutocompleteRenderInputParams
	nameField: nameFieldKeys
	errors: FieldErrors<IVirtualServiceForm>
	selectedItem?: ListenerListItem | VirtualServiceTemplateListItem | AccessLogConfigListItem | null
	isFetching: boolean
	isErrorFetch: boolean
	variant?: string
}

const fieldTitles: Record<string, string> = {
	templateUid: 'Template',
	listenerUid: 'Listeners',
	accessLogConfigUid: 'AccessLogConfig'
}

export const RenderInputField: React.FC<IRenderInputFieldProps> = ({
	className,
	params,
	nameField,
	errors,
	isFetching,
	isErrorFetch,
	selectedItem,
	variant
}) => {
	const titleMessage = fieldTitles[nameField] || nameField

	return (
		<TextField
			variant={variant ? 'standard' : 'outlined'}
			{...params}
			label={fieldTitles[nameField]}
			error={!!errors[nameField] || isErrorFetch}
			helperText={errors[nameField]?.message || (isErrorFetch ? `Error loading ${titleMessage} data` : '')}
			onKeyDown={e => {
				const container = document.querySelector(`.${className}-${nameField}`)
				const autocompletePopup = document.querySelector('.MuiAutocomplete-popper')
				const isAutocompleteOpen = container && autocompletePopup && autocompletePopup.clientHeight > 0

				if (e.key === 'Enter' && isAutocompleteOpen) {
					e.preventDefault()
				}
			}}
			slotProps={{
				input: {
					...params.InputProps,
					endAdornment: (
						<>
							{className === 'autocompleteVs' && (
								<Typography variant='body2' sx={{ wordWrap: 'break-word' }} color='textDisabled'>
									{selectedItem?.description || ''}
								</Typography>
							)}
							{isFetching ? <CircularProgress color='inherit' size={20} /> : null}
							{params.InputProps.endAdornment}
						</>
					)
				}
			}}
		/>
	)
}
