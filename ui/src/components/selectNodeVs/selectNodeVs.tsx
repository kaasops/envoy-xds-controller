import React from 'react'
import Box from '@mui/material/Box'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'

import { Control, Controller, FieldErrors } from 'react-hook-form'
import { styleBox, styleTooltip } from './style.ts'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import { ListNodesResponse } from '../../gen/node/v1/node_pb'
import FormControl from '@mui/material/FormControl'
import Select from '@mui/material/Select'
import { validationRulesVsForm } from '../../utils/helpers/validationRulesVsForm.ts'
import MenuItem from '@mui/material/MenuItem'
import CircularProgress from '@mui/material/CircularProgress'
import Chip from '@mui/material/Chip'
import FormHelperText from '@mui/material/FormHelperText'
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'

interface ISelectNodeVsProps {
	nameField: Extract<keyof IVirtualServiceForm, 'nodeIds'>
	dataNodes: ListNodesResponse | undefined
	control: Control<IVirtualServiceForm>
	errors: FieldErrors<IVirtualServiceForm>
	isFetching: boolean
	isErrorFetch: boolean
}

export const SelectNodeVs: React.FC<ISelectNodeVsProps> = ({
	nameField,
	dataNodes,
	errors,
	control,
	isFetching,
	isErrorFetch
}) => {
	const titleMessage = 'NodeIDs'
	const readMode = useViewModeStore(state => state.viewMode) === 'read'

	return (
		<Box sx={{ ...styleBox }}>
			<Typography fontSize={15} color='gray' mt={1} display='flex' alignItems='center' gap={0.5}>
				{titleMessage}
				<Tooltip
					title={`Select ${titleMessage.slice(0, -1)}.`}
					placement='bottom-start'
					enterDelay={800}
					disableInteractive
					slotProps={{ ...styleTooltip }}
				>
					<InfoOutlinedIcon fontSize='inherit' sx={{ cursor: 'pointer', fontSize: '14px' }} />
				</Tooltip>
			</Typography>
			<Controller
				name={nameField}
				control={control}
				rules={{
					validate: validationRulesVsForm[nameField]
				}}
				render={({ field }) => (
					<FormControl fullWidth error={!!errors[nameField] || isErrorFetch}>
						<Select
							multiple
							disabled={readMode}
							label={titleMessage}
							value={field.value || []}
							onChange={event => {
								const selectedValues = event.target.value
								field.onChange(selectedValues)
							}}
							variant='standard'
							IconComponent={
								isFetching ? () => <CircularProgress size={20} sx={{ marginRight: 2 }} /> : undefined
							}
							sx={{ '& .MuiSelect-icon': { width: '24px', height: '24px' } }}
							renderValue={selected => (
								<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
									{selected.map(value => (
										<Chip
											key={value}
											label={value}
											// чтобы клик не открывал меню
											onMouseDown={e => e.stopPropagation()}
											onDelete={() => {
												const newValues = field.value.filter(v => v !== value)
												field.onChange(newValues) // Обновляем состояние
											}}
											variant='outlined'
										/>
									))}
								</Box>
							)}
						>
							{isErrorFetch && (
								<MenuItem disabled>
									<span style={{ color: 'error' }}>{`Error loading ${titleMessage} data`}</span>
								</MenuItem>
							)}

							{dataNodes?.items?.map(node => (
								<MenuItem key={node.id} value={node.id}>
									{node.id}
								</MenuItem>
							))}
						</Select>
						<FormHelperText sx={{ ml: 0 }}>
							{errors[nameField]?.message || (isErrorFetch && `Error loading ${titleMessage} data`)}
						</FormHelperText>
					</FormControl>
				)}
			/>
		</Box>
	)
}
