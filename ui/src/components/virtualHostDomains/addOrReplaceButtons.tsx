import React from 'react'
import { Control, UseFormSetValue, useWatch } from 'react-hook-form'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { Button, ButtonGroup, Tooltip } from '@mui/material'
import { useHttpFilterTemplateOptions, useRouteTemplateOptions, useVHDomainsTemplateOptions } from '../../utils/hooks'

interface IAddOrReplaceButtonsProps {
	control: Control<IVirtualServiceForm>
	setValue: UseFormSetValue<IVirtualServiceForm>
	mode: 'virtualHostDomainsMode' | 'additionalHttpFilterMode' | 'additionalRouteMode'
}

const modeLabels: Record<IAddOrReplaceButtonsProps['mode'], string> = {
	virtualHostDomainsMode: 'Domains',
	additionalHttpFilterMode: 'HTTP Filters',
	additionalRouteMode: 'Routes'
}

export const AddOrReplaceButtons: React.FC<IAddOrReplaceButtonsProps> = ({ control, setValue, mode }) => {
	const readMode = useViewModeStore(state => state.viewMode) === 'read'
	const templateUid = useWatch({ control, name: 'templateUid' })
	const isReplaceMode = useWatch({ control, name: mode })

	useVHDomainsTemplateOptions({ control, setValue })
	useHttpFilterTemplateOptions({ control, setValue })
	useRouteTemplateOptions({ control, setValue })

	const label = modeLabels[mode]

	if (readMode || !templateUid) return null

	return (
		<ButtonGroup variant='contained' size='small' sx={{ height: '1.5rem' }}>
			<Tooltip title={`Added ${label} to existing ones`}>
				<Button onClick={() => setValue(mode, false)} color={!isReplaceMode ? 'primary' : 'inherit'}>
					Add
				</Button>
			</Tooltip>
			<Tooltip title={`Replace existing ${label}`}>
				<Button onClick={() => setValue(mode, true)} color={isReplaceMode ? 'primary' : 'inherit'}>
					Rep
				</Button>
			</Tooltip>
		</ButtonGroup>
	)
}
