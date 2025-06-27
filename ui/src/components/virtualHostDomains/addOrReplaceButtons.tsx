import React from 'react'
import { Control, UseFormSetValue, useWatch } from 'react-hook-form'
import { IVirtualServiceForm } from '../virtualServiceForm'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { Button, ButtonGroup, Tooltip } from '@mui/material'
import {
	useAccessLogTemplateOptions,
	useHttpFilterTemplateOptions,
	useRouteTemplateOptions,
	useVHDomainsTemplateOptions
} from '../../utils/hooks'
import { FillTemplateResponse } from '../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'

interface IAddOrReplaceButtonsProps {
	control: Control<IVirtualServiceForm>
	setValue: UseFormSetValue<IVirtualServiceForm>
	mode:
		| 'virtualHostDomainsMode'
		| 'additionalHttpFilterMode'
		| 'additionalRouteMode'
		| 'additionalAccessLogConfigMode'
	fillTemplate?: FillTemplateResponse | undefined
}

const modeLabels: Record<IAddOrReplaceButtonsProps['mode'], string> = {
	virtualHostDomainsMode: 'Domains',
	additionalHttpFilterMode: 'HTTP Filters',
	additionalRouteMode: 'Routes',
	additionalAccessLogConfigMode: 'Access Log Configs'
}

export const AddOrReplaceButtons: React.FC<IAddOrReplaceButtonsProps> = ({ control, setValue, mode, fillTemplate }) => {
	const readMode = useViewModeStore(state => state.viewMode) === 'read'
	const templateUid = useWatch({ control, name: 'templateUid' })
	const isReplaceMode = useWatch({ control, name: mode as keyof IVirtualServiceForm })

	useVHDomainsTemplateOptions({ control, setValue })
	useHttpFilterTemplateOptions({ control, setValue })
	useRouteTemplateOptions({ control, setValue })
	useAccessLogTemplateOptions({ control, setValue, fillTemplate })

	const label = modeLabels[mode]

	if (readMode || !templateUid) return null

	return (
		<ButtonGroup variant='contained' size='small' sx={{ height: '1.5rem' }}>
			<Tooltip title={`Added ${label} to existing ones`}>
				<Button
					onClick={() => setValue(mode as keyof IVirtualServiceForm, false)}
					color={!isReplaceMode ? 'primary' : 'inherit'}
				>
					Add
				</Button>
			</Tooltip>
			<Tooltip title={`Replace existing ${label}`}>
				<Button
					onClick={() => setValue(mode as keyof IVirtualServiceForm, true)}
					color={isReplaceMode ? 'primary' : 'inherit'}
				>
					Rep
				</Button>
			</Tooltip>
		</ButtonGroup>
	)
}
