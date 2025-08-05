import { UseFormReset } from 'react-hook-form'
import { IVirtualServiceForm } from '../../components/virtualServiceForm'
import { useCallback, useEffect } from 'react'
import { GetVirtualServiceResponse } from '../../gen/virtual_service/v1/virtual_service_pb'
import { TemplateOptionModifier } from '../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'

interface ISetDefaultValuesVSFormProps {
	reset: UseFormReset<IVirtualServiceForm>
	isCreate: boolean
	virtualServiceInfo: GetVirtualServiceResponse | undefined
}

export const useSetDefaultValuesVSForm = ({ reset, isCreate, virtualServiceInfo }: ISetDefaultValuesVSFormProps) => {
	const setDefaultValues = useCallback(() => {
		if (isCreate || !virtualServiceInfo) return

		const vhDomains = virtualServiceInfo?.virtualHost?.domains || []
		const hasReplaceModifierForVHDomains =
			virtualServiceInfo.templateOptions?.some(
				opt => opt.field === 'virtualHost.domains' && opt.modifier === TemplateOptionModifier.REPLACE
			) || false
		const hasReplaceHttpFilters =
			virtualServiceInfo.templateOptions?.some(
				opt => opt.field === 'additionalHttpFilters' && opt.modifier === TemplateOptionModifier.REPLACE
			) || false

		const hasReplaceRoutes =
			virtualServiceInfo.templateOptions?.some(
				opt => opt.field === 'additionalRoutes' && opt.modifier === TemplateOptionModifier.REPLACE
			) || false

		reset({
			name: virtualServiceInfo.name,
			nodeIds: virtualServiceInfo.nodeIds || [],
			accessGroup: virtualServiceInfo.accessGroup,
			templateUid: virtualServiceInfo.template?.uid,
			listenerUid: virtualServiceInfo.listener?.uid,
			accessLogConfigUids:
				virtualServiceInfo.accessLog?.case === 'accessLogConfigs'
					? virtualServiceInfo.accessLog.value.refs.map(ref => ref.uid)
					: [],
			useRemoteAddress: virtualServiceInfo.useRemoteAddress,
			templateOptions: virtualServiceInfo.templateOptions,
			virtualHostDomains: vhDomains,
			additionalHttpFilterUids: virtualServiceInfo.additionalHttpFilters?.map(filter => filter.uid) || [],
			additionalRouteUids: virtualServiceInfo.additionalRoutes?.map(router => router.uid) || [],
			description: virtualServiceInfo.description,
			virtualHostDomainsMode: hasReplaceModifierForVHDomains,
			additionalHttpFilterMode: hasReplaceHttpFilters,
			additionalRouteMode: hasReplaceRoutes,
			extraFields: virtualServiceInfo.extraFields || {}
		})
	}, [reset, isCreate, virtualServiceInfo])

	useEffect(() => {
		setDefaultValues()
	}, [setDefaultValues])

	return { setDefaultValues }
}
