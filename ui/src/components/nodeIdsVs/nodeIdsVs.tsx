import React from 'react'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import { ListNodesResponse } from '../../gen/node/v1/node_pb.ts'
import { Control, Controller, FieldErrors } from 'react-hook-form'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { nodeIdsBlock } from './style.ts'
import { ToolTipVs } from '../toolTipVs/toolTipVs.tsx'
import Box from '@mui/material/Box'
import { validationRulesVsForm } from '../../utils/helpers/validationRulesVsForm.ts'
import { Autocomplete, AutocompleteRenderInputParams, CircularProgress, TextField } from '@mui/material'

interface INodeIdsVsProps {
	nameField: Extract<keyof IVirtualServiceForm, 'nodeIds'>
	dataNodes: ListNodesResponse | undefined
	control: Control<IVirtualServiceForm>
	errors: FieldErrors<IVirtualServiceForm>
	isFetching: boolean
	isErrorFetch: boolean
}

export const NodeIdsVs: React.FC<INodeIdsVsProps> = ({
	nameField,
	dataNodes,
	errors,
	control,
	isFetching,
	isErrorFetch
}) => {
	const titleMessage = 'NodeIDs'
	const readMode = useViewModeStore(state => state.viewMode) === 'read'

	const renderInput = (params: AutocompleteRenderInputParams) => (
		<TextField
			{...params}
			variant='standard'
			error={!!errors[nameField] || isErrorFetch}
			helperText={errors[nameField]?.message || (isErrorFetch ? `Error loading ${titleMessage} data` : '')}
			onKeyDown={e => {
				const container = document.querySelector('.nodeIdsBlock')
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
							{isFetching ? <CircularProgress color='inherit' size={20} /> : null}
							{params.InputProps.endAdornment}
						</>
					)
				}
			}}
		/>
	)

	return (
		<Box className='nodeIdsBlock' sx={{ ...nodeIdsBlock }}>
			<ToolTipVs titleMessage={titleMessage} />

			<Controller
				name={nameField}
				control={control}
				rules={{
					validate: validationRulesVsForm[nameField]
				}}
				render={({ field }) => (
					<Autocomplete
						multiple
						disabled={readMode}
						loading={isFetching}
						value={field.value || []}
						options={dataNodes?.items?.map(node => node.id) || []}
						onChange={(_, newValue) => field.onChange(newValue)}
						renderInput={params => renderInput(params)}
					/>
				)}
			/>
		</Box>
	)
}
