import { IVirtualServiceForm } from '../../components/virtualServiceForm/types.ts'
import {
	FillTemplateRequest,
	TemplateOption
} from '../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'
import { useCallback, useEffect, useMemo } from 'react'
import { debounce } from '@mui/material'
import { useFillTemplate } from '../../api/grpc/hooks/useVirtualService.ts'

interface IUseFillTemplate {
	formValues: IVirtualServiceForm
}

export const useFillTemplateHook = ({ formValues }: IUseFillTemplate) => {
	const { getTemplate, rawData: rawDataTemplate, isLoadingFillTemplate, errorFillTemplate } = useFillTemplate()

	const prepareTemplateRequestData = useCallback((): FillTemplateRequest => {
		const {
			nodeIds,
			virtualHostDomains,
			templateOptions,
			accessLogConfigUid,
			viewTemplateMode,
			virtualHostDomainsMode,
			additionalRouteMode,
			additionalHttpFilterMode,
			useRemoteAddress,
			...rest
		} = formValues || {}

		const cleanedTemplateOptions: TemplateOption[] =
			Array.isArray(templateOptions) &&
			templateOptions.some(opt => !opt.field || opt.field.trim() === '' || !opt.modifier || opt.modifier === 0)
				? []
				: templateOptions.map(opt => ({
						...opt,
						$typeName: 'virtual_service_template.v1.TemplateOption'
					}))

		return {
			$typeName: 'virtual_service_template.v1.FillTemplateRequest',
			...rest,
			virtualHost: {
				$typeName: 'common.v1.VirtualHost',
				domains: virtualHostDomains || []
			},
			accessLogConfig: {
				value: accessLogConfigUid || '',
				case: 'accessLogConfigUid'
			},
			templateOptions: cleanedTemplateOptions,
			expandReferences: viewTemplateMode,
			useRemoteAddress
		}
	}, [formValues])

	const debouncedGetFillTemplate = useMemo(
		() => debounce((data: FillTemplateRequest) => getTemplate(data), 500),
		[getTemplate]
	)

	useEffect(() => {
		const requestData = prepareTemplateRequestData()

		if (!requestData.templateUid) return

		debouncedGetFillTemplate(requestData)

		return () => {
			debouncedGetFillTemplate.clear()
		}
	}, [formValues, prepareTemplateRequestData, debouncedGetFillTemplate])

	return { prepareTemplateRequestData, rawDataTemplate, isLoadingFillTemplate, errorFillTemplate }
}
