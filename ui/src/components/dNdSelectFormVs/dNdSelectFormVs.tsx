import React, { useEffect, useState } from 'react'
import { Control, Controller, FieldErrors, UseFormSetValue } from 'react-hook-form'
import { HTTPFilterListItem, ListHTTPFiltersResponse } from '../../gen/http_filter/v1/http_filter_pb.ts'
import { ListRoutesResponse, RouteListItem } from '../../gen/route/v1/route_pb.ts'
import { validationRulesVsForm } from '../../utils/helpers/validationRulesVsForm.ts'
import { dNdBox } from './style.ts'
import Autocomplete from '@mui/material/Autocomplete'
import Box from '@mui/material/Box'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { ToolTipVs } from '../toolTipVs/toolTipVs.tsx'
import { DNdElements } from './dNdElements.tsx'
import { AutocompleteOption, PopoverOption, RenderInputField } from '../autocompleteVs'
import { AddOrReplaceButtons } from '../virtualHostDomains'

export type nameFieldKeys = Extract<keyof IVirtualServiceForm, 'additionalHttpFilterUids' | 'additionalRouteUids'>

export type ItemDnd = HTTPFilterListItem | RouteListItem

interface IdNdSelectFormVsProps {
	nameField: nameFieldKeys
	data: ListHTTPFiltersResponse | ListRoutesResponse | undefined
	control: Control<IVirtualServiceForm>
	setValue: UseFormSetValue<IVirtualServiceForm>
	errors: FieldErrors<IVirtualServiceForm>
	isErrorFetch: boolean
	isFetching: boolean
}

export const DNdSelectFormVs: React.FC<IdNdSelectFormVsProps> = ({
	nameField,
	data,
	control,
	setValue,
	errors,
	isFetching,
	isErrorFetch
}) => {
	const titleMessage = nameField === 'additionalHttpFilterUids' ? 'HTTP filter' : 'Route'
	const readMode = useViewModeStore(state => state.viewMode) === 'read'

	const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)
	const [popoverOption, setPopoverOption] = useState<ItemDnd | null>(null)
	const templateOptionMode =
		nameField === 'additionalHttpFilterUids' ? 'additionalHttpFilterMode' : 'additionalRouteMode'

	const SUPPORTED_TYPES = new Set(['http_filter.v1.HTTPFilterListItem', 'route.v1.RouteListItem'])

	const handleOpenPopover = (event: React.MouseEvent<HTMLButtonElement>, option: AutocompleteOption) => {
		if (!SUPPORTED_TYPES.has(option.$typeName)) return

		event.stopPropagation()
		setAnchorEl(event.currentTarget)
		setPopoverOption(option as ItemDnd)
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
		<Box sx={{ ...dNdBox }}>
			<Box display='flex' justifyContent='space-between' alignItems='center'>
				<ToolTipVs titleMessage={titleMessage} delay={500} isDnD={true} />
				<AddOrReplaceButtons control={control} setValue={setValue} mode={templateOptionMode} />
			</Box>
			<Controller
				name={nameField}
				control={control}
				rules={{
					validate: validationRulesVsForm[nameField]
				}}
				render={({ field }) => (
					<>
						<Autocomplete
							multiple
							className={`dndAutoComplete-${nameField}`}
							disabled={readMode}
							loading={isFetching}
							options={data?.items || []}
							value={(data?.items || []).filter(item => field.value.includes(item.uid))}
							onChange={(_, newValue) => field.onChange(newValue.map(item => item.uid))}
							getOptionLabel={option => option.name}
							renderInput={params => (
								<RenderInputField
									className={'autocompleteVs'}
									variant='standard'
									params={params}
									nameField={nameField}
									errors={errors}
									isFetching={isFetching}
									isErrorFetch={isErrorFetch}
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
				)}
			/>
			<DNdElements
				titleMessage={titleMessage}
				nameField={nameField}
				control={control}
				data={data}
				setValue={setValue}
			/>
		</Box>
	)
}
