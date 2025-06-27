import { IVirtualServiceForm } from '../../components/virtualServiceForm'
import { Control, UseFormSetValue, useWatch } from 'react-hook-form'
import { useEffect } from 'react'

type nameFieldKeys = Extract<
	keyof IVirtualServiceForm,
	'additionalHttpFilterUids' | 'accessLogConfigUids' | 'additionalRouteUids' | 'virtualHostDomains'
>

const FIELD_MAP: Record<nameFieldKeys, { relatedFields: string[]; modeName: string }> = {
	virtualHostDomains: {
		relatedFields: ['virtualHost.domains'],
		modeName: 'virtualHostDomainsMode'
	},
	additionalHttpFilterUids: {
		relatedFields: ['additionalHttpFilters'],
		modeName: 'additionalHttpFilterMode'
	},
	additionalRouteUids: {
		relatedFields: ['additionalRoutes'],
		modeName: 'additionalRouteMode'
	},
	accessLogConfigUids: {
		relatedFields: ['accessLog', 'accessLogConfig', 'accessLogs', 'accessLogConfigs'],
		modeName: 'additionalAccessLogConfigMode'
	}
} as const

interface IProps {
	fieldName: nameFieldKeys
	control: Control<IVirtualServiceForm>
	setValue: UseFormSetValue<IVirtualServiceForm>
}

export const useRemoveFieldFromTemplate = ({ fieldName, control, setValue }: IProps) => {
	const fieldValue = useWatch({ control, name: fieldName })
	const templateOptions = useWatch({ control, name: 'templateOptions' })

	const config = FIELD_MAP[fieldName]

	useEffect(() => {
		const isEmpty = Array.isArray(fieldValue) && !fieldValue.length

		if (!isEmpty) return

		const filtered = templateOptions.filter(opt => !config.relatedFields.includes(opt.field))

		const isChanged = filtered.length !== templateOptions.length

		if (isChanged) {
			setValue('templateOptions', filtered, {
				shouldDirty: true,
				shouldTouch: true,
				shouldValidate: true
			})

			setValue(config.modeName as keyof IVirtualServiceForm, false, {
				shouldDirty: true,
				shouldTouch: true,
				shouldValidate: true
			})
		}
	}, [fieldValue, fieldName, templateOptions, config, setValue])
}
