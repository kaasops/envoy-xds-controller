import { Control, UseFormSetValue, useWatch } from 'react-hook-form'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { useEffect, useState } from 'react'
import { IVirtualServiceForm } from '../../components/virtualServiceForm'
import { FillTemplateResponse } from '../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'
import { updateTemplateOptions } from '../helpers'

interface UseAccessLogTemplateOptionsProps {
	control: Control<IVirtualServiceForm>
	setValue?: UseFormSetValue<IVirtualServiceForm>
	fillTemplate?: FillTemplateResponse
}

export const useAccessLogTemplateOptions = ({
	control,
	setValue,
	fillTemplate
}: UseAccessLogTemplateOptionsProps): void => {
	const readMode = useViewModeStore(state => state.viewMode) === 'read'
	const accessLogField = useWatch({ control, name: 'accessLogConfigUid' })
	const currentOptions = useWatch({ control, name: 'templateOptions' })

	const [initialRaw, setInitialRaw] = useState<string | undefined>(undefined)
	const [wasAccessLogSet, setWasAccessLogSet] = useState(false)

	useEffect(() => {
		if (!readMode && fillTemplate?.raw && !initialRaw) {
			setInitialRaw(fillTemplate.raw)
		}
	}, [fillTemplate?.raw, readMode, initialRaw])

	useEffect(() => {
		if (!initialRaw || readMode || !setValue) return

		let hasAccessLogInRaw = false
		try {
			const parsed = JSON.parse(initialRaw)
			hasAccessLogInRaw = 'accessLog' in parsed
		} catch (e) {
			console.error('Error parsing fillTemplate.raw:', e)
			return
		}

		const shouldHaveAccessLog = Boolean(accessLogField) && hasAccessLogInRaw

		if (shouldHaveAccessLog === wasAccessLogSet) return
		setWasAccessLogSet(shouldHaveAccessLog)

		const updatedOptions = updateTemplateOptions({
			currentTemplateOptions: currentOptions,
			optionKey: 'accessLog',
			isReplaceMode: shouldHaveAccessLog,
			modifier: 3
		})

		if (JSON.stringify(updatedOptions) !== JSON.stringify(currentOptions)) {
			setValue('templateOptions', updatedOptions, {
				shouldValidate: true,
				shouldDirty: true,
				shouldTouch: true
			})
		}
	}, [initialRaw, accessLogField, currentOptions, readMode, setValue, wasAccessLogSet])
}
