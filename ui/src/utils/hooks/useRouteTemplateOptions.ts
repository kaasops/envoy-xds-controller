import { Control, UseFormSetValue, useWatch } from 'react-hook-form'
import { IVirtualServiceForm } from '../../components/virtualServiceForm'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { useEffect } from 'react'
import { updateTemplateOptions } from '../helpers'

interface IUseRouteTemplateOptions {
	control: Control<IVirtualServiceForm>
	setValue: UseFormSetValue<IVirtualServiceForm>
}

export const useRouteTemplateOptions = ({ control, setValue }: IUseRouteTemplateOptions) => {
	const optionKey = 'additionalRoutes'

	const readMode = useViewModeStore(state => state.viewMode) === 'read'

	const isReplaceMode = useWatch({ control, name: 'additionalRouteMode' })
	const routeField = useWatch({ control, name: 'additionalRouteUids' })
	const currentTemplateOptions = useWatch({ control, name: 'templateOptions' })

	useEffect(() => {
		if (readMode || !routeField) return

		const updatedOptions = updateTemplateOptions({ currentTemplateOptions, optionKey, isReplaceMode })

		if (updatedOptions !== currentTemplateOptions) {
			setValue('templateOptions', updatedOptions, {
				shouldValidate: true,
				shouldDirty: true,
				shouldTouch: true
			})
		}
	}, [isReplaceMode, readMode, routeField, currentTemplateOptions, setValue])
}
