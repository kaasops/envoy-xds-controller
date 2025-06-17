import { Control, UseFormSetValue, useWatch } from 'react-hook-form'
import { IVirtualServiceForm } from '../../components/virtualServiceForm/types.ts'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { useEffect } from 'react'
import { updateTemplateOptions } from '../helpers'

interface IUseVHDomainsTemplateOptions {
	control: Control<IVirtualServiceForm>
	setValue: UseFormSetValue<IVirtualServiceForm>
}

export const useVHDomainsTemplateOptions = ({ control, setValue }: IUseVHDomainsTemplateOptions): void => {
	const optionKey = 'virtualHost.domains'
	const readMode = useViewModeStore(state => state.viewMode) === 'read'

	const isReplaceMode = useWatch({ control, name: 'virtualHostDomainsMode' })
	const vhDomainsField = useWatch({ control, name: 'virtualHostDomains' })
	const currentTemplateOptions = useWatch({ control, name: 'templateOptions' })

	useEffect(() => {
		if (readMode || !vhDomainsField) return

		const updatedOptions = updateTemplateOptions({ currentTemplateOptions, optionKey, isReplaceMode })

		if (updatedOptions !== currentTemplateOptions) {
			setValue('templateOptions', updatedOptions, {
				shouldValidate: true,
				shouldDirty: true,
				shouldTouch: true
			})
		}
	}, [isReplaceMode, readMode, vhDomainsField, currentTemplateOptions, setValue])
}
