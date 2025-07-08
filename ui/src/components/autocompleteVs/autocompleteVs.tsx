import React, { useEffect, useState } from 'react'
import { IVirtualServiceForm } from '../virtualServiceForm'
import { ListenerListItem, ListListenersResponse } from '../../gen/listener/v1/listener_pb.ts'
import {
	ListVirtualServiceTemplatesResponse,
	VirtualServiceTemplateListItem
} from '../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'
import { Control, Controller, FieldErrors } from 'react-hook-form'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { validationRulesVsForm } from '../../utils/helpers'
import Autocomplete from '@mui/material/Autocomplete'
import { RenderInputField } from './renderInputField.tsx'
import { AutocompleteOption } from './autocompleteOption.tsx'
import { PopoverOption } from './popoverOption.tsx'

export type nameFieldKeys = Extract<keyof IVirtualServiceForm, 'templateUid' | 'listenerUid'>

export type ItemVs = ListenerListItem | VirtualServiceTemplateListItem

interface IAutocompleteVsProps {
	nameField: nameFieldKeys
	control: Control<IVirtualServiceForm>
	data: ListListenersResponse | ListVirtualServiceTemplatesResponse | undefined
	errors: FieldErrors<IVirtualServiceForm>
	isFetching: boolean
	isErrorFetch: boolean
}

export const AutocompleteVs: React.FC<IAutocompleteVsProps> = ({
	nameField,
	data,
	control,
	errors,
	isErrorFetch,
	isFetching
}) => {
	const readMode = useViewModeStore(state => state.viewMode) === 'read'

	const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)
	const [popoverOption, setPopoverOption] = useState<ItemVs | null>(null)

	const SUPPORTED_TYPES = new Set([
		'listener.v1.ListenerListItem',
		'virtual_service_template.v1.VirtualServiceTemplateListItem'
	])

	const handleOpenPopover = (event: React.MouseEvent<HTMLButtonElement>, option: AutocompleteOption) => {
		if (!SUPPORTED_TYPES.has(option.$typeName)) return

		event.stopPropagation()
		setAnchorEl(event.currentTarget)
		setPopoverOption(option as ItemVs)
	}

	const handleClosePopover = () => {
		setAnchorEl(null)
		setPopoverOption(null)
	}

	useEffect(() => {
		if (anchorEl && !document.body.contains(anchorEl)) {
			handleClosePopover()
		}
	}, [anchorEl])

	return (
		<>
			<Controller
				name={nameField}
				control={control}
				rules={{ validate: validationRulesVsForm[nameField] }}
				render={({ field }) => {
					const filteredItems = (data?.items || []).filter(item => {
						if (nameField === 'listenerUid') return item.$typeName === 'listener.v1.ListenerListItem'
						if (nameField === 'templateUid')
							return item.$typeName === 'virtual_service_template.v1.VirtualServiceTemplateListItem'
						return false
					})

					const selectedItem = filteredItems.find(item => item.uid === field.value) || null

					return (
						<>
							<Autocomplete
								className={`autocompleteVs-${nameField}`}
								disabled={readMode}
								loading={isFetching}
								options={filteredItems}
								value={selectedItem}
								getOptionLabel={option => option.name}
								isOptionEqualToValue={(option, value) => option.uid === value.uid}
								onChange={(_, newValue) => field.onChange(newValue ? newValue.uid : '')}
								renderInput={params => (
									<RenderInputField
										className={'autocompleteVs'}
										params={params}
										nameField={nameField}
										errors={errors}
										isFetching={isFetching}
										isErrorFetch={isErrorFetch}
										selectedItem={selectedItem}
									/>
								)}
								renderOption={(props, option) => (
									<AutocompleteOption
										key={option.uid}
										option={option}
										props={props}
										onPreviewClick={handleOpenPopover}
									/>
								)}
							/>

							<PopoverOption anchorEl={anchorEl} option={popoverOption} onClose={handleClosePopover} />
						</>
					)
				}}
			/>
		</>
	)
}
