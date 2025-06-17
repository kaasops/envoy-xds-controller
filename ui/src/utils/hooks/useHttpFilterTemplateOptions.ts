import { Control, UseFormSetValue, useWatch } from 'react-hook-form'
import { IVirtualServiceForm } from '../../components/virtualServiceForm/types.ts'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { useEffect } from 'react'
import { updateTemplateOptions } from '../helpers'

interface IUseHttpFilterTemplateOptions {
	control: Control<IVirtualServiceForm>
	setValue: UseFormSetValue<IVirtualServiceForm>
}

export const useHttpFilterTemplateOptions = ({ control, setValue }: IUseHttpFilterTemplateOptions) => {
	const optionKey = 'additionalHttpFilters'

	const readMode = useViewModeStore(state => state.viewMode) === 'read'

	const isReplaceMode = useWatch({ control, name: 'additionalHttpFilterMode' })
	const httpFiltersField = useWatch({ control, name: 'additionalHttpFilterUids' })
	const currentTemplateOptions = useWatch({ control, name: 'templateOptions' })

	useEffect(() => {
		if (readMode || !httpFiltersField) return

		const updatedOptions = updateTemplateOptions({ currentTemplateOptions, optionKey, isReplaceMode })

		if (updatedOptions !== currentTemplateOptions) {
			setValue('templateOptions', updatedOptions, {
				shouldValidate: true,
				shouldDirty: true,
				shouldTouch: true
			})
		}
	}, [isReplaceMode, readMode, setValue, httpFiltersField, currentTemplateOptions])
}
