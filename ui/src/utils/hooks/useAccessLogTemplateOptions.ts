import { Control, UseFormSetValue, useWatch } from 'react-hook-form'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { useEffect } from 'react'
import { IVirtualServiceForm } from '../../components/virtualServiceForm'
import { updateTemplateOptions } from '../helpers'

interface UseAccessLogTemplateOptionsProps {
	control: Control<IVirtualServiceForm>
	setValue?: UseFormSetValue<IVirtualServiceForm>
}

export const useAccessLogTemplateOptions = ({
	control,
	setValue
}: UseAccessLogTemplateOptionsProps): void => {
	const optionKey = 'accessLogConfigs'
	const readMode = useViewModeStore(state => state.viewMode) === 'read'

	const isReplaceMode = useWatch({ control, name: 'additionalAccessLogConfigMode' })
	const accessLogField = useWatch({ control, name: 'accessLogConfigUids' })
	const currentTemplateOptions = useWatch({ control, name: 'templateOptions' })

	useEffect(() => {
		if (readMode || !setValue || !accessLogField?.length) return

		let updatedOptions = [...currentTemplateOptions]

		// If we're in Add mode and there are existing access log fields in the template,
		// add delete rules for them
		if (!isReplaceMode) {
			// TODO: hardcode
			for (const field of ['accessLog', 'accessLogConfig', 'accessLogs']) {
				updatedOptions = updateTemplateOptions({
					currentTemplateOptions: updatedOptions,
					optionKey: field,
					isReplaceMode: true,
					modifier: 3 // Delete
				})
			}
		} else {
			// Otherwise, just update the accessLogConfigs field as before
			updatedOptions = updateTemplateOptions({
				currentTemplateOptions,
				optionKey,
				isReplaceMode,
				modifier: 2
			})
		}

		if (JSON.stringify(updatedOptions) !== JSON.stringify(currentTemplateOptions)) {
			setValue('templateOptions', updatedOptions, {
				shouldValidate: true,
				shouldDirty: true,
				shouldTouch: true
			})
		}
	}, [isReplaceMode, readMode, accessLogField, currentTemplateOptions, setValue])
}
