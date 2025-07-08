import { Control, UseFormSetValue, useWatch } from 'react-hook-form'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { useEffect, useState } from 'react'
import { IVirtualServiceForm } from '../../components/virtualServiceForm'
import { FillTemplateResponse } from '../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'
import { updateTemplateOptionsNew } from '../helpers/updateTemplateOptions.ts'

interface UseAccessLogTemplateOptionsProps {
	control: Control<IVirtualServiceForm>
	setValue: UseFormSetValue<IVirtualServiceForm>
	fillTemplate: FillTemplateResponse | undefined
}

export const useAccessLogTemplateOptions = ({
	control,
	setValue,
	fillTemplate
}: UseAccessLogTemplateOptionsProps): void => {
	const readMode = useViewModeStore(state => state.viewMode) === 'read'

	const isReplaceMode = useWatch({ control, name: 'additionalAccessLogConfigMode' })
	const accessLogField = useWatch({ control, name: 'accessLogConfigUids' })
	const currentTemplateOptions = useWatch({ control, name: 'templateOptions' })

	const [initialRaw, setInitialRaw] = useState<string | undefined>(undefined)

	useEffect(() => {
		if (!readMode && fillTemplate?.raw && !initialRaw) {
			setInitialRaw(fillTemplate.raw)
		}
	}, [fillTemplate?.raw, readMode, initialRaw])

	useEffect(() => {
		if (!initialRaw || !accessLogField.length || readMode) return

		let parsedAccessLogVariation: string | false = false

		try {
			const parsed = JSON.parse(initialRaw)
			const keysToCheck = ['accessLog', 'accessLogConfig', 'accessLogs', 'accessLogConfigs']
			parsedAccessLogVariation = keysToCheck.find(key => key in parsed) || false
		} catch (e) {
			console.error('Error parsing fillTemplate.raw:', e)
			return
		}

		if (!parsedAccessLogVariation) return

		const updatedOptions = updateTemplateOptionsNew({
			currentTemplateOptions,
			optionKey: parsedAccessLogVariation,
			isReplaceMode
		})

		const isChanged = JSON.stringify(updatedOptions) !== JSON.stringify(currentTemplateOptions)

		if (isChanged) {
			setValue('templateOptions', updatedOptions, {
				shouldValidate: true,
				shouldDirty: true,
				shouldTouch: true
			})
		}
	}, [initialRaw, readMode, accessLogField, currentTemplateOptions, isReplaceMode, setValue])
}
