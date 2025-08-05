import { UIDS, VirtualHost } from '../../gen/common/v1/common_pb.ts'
import { ITemplateOption, IVirtualServiceForm } from '../../components/virtualServiceForm'
import {
	CreateVirtualServiceRequest,
	UpdateVirtualServiceRequest
} from '../../gen/virtual_service/v1/virtual_service_pb'
import { TemplateOption } from '../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'

export const buildVirtualHost = (vHDomains: string[] = []): VirtualHost => ({
	$typeName: 'common.v1.VirtualHost',
	domains: vHDomains
})

export const buildTemplateOptions = (templateOptions: ITemplateOption[] = []): TemplateOption[] => {
	const hasValidOption = templateOptions.some(option => option.field !== '' || option.modifier !== 0)

	if (!hasValidOption) return []

	return templateOptions.map(option => ({
		...option,
		$typeName: 'virtual_service_template.v1.TemplateOption' as const
	}))
}

export const buildAccessLogConfig = (
	uids?: string[]
): { case: 'accessLogConfigUids'; value: UIDS } | { case: undefined } => {
	return uids ? { 
		case: 'accessLogConfigUids' as const, 
		value: {
			$typeName: 'common.v1.UIDS',
			uids: uids
		} 
	} : { case: undefined }
}

const buildExtraFields = (extraFields?: Record<string, string>) => {
	return extraFields || {}
}

export const buildBaseVSData = (data: IVirtualServiceForm) => {
	const {
		//unnecessary data
		additionalRouteMode,
		additionalHttpFilterMode,
		virtualHostDomainsMode,
		viewTemplateMode,
		//necessary data
		virtualHostDomains,
		templateOptions,
		accessLogConfigUids,
		extraFields,
		...rest
	} = data

	return {
		...rest,
		virtualHost: buildVirtualHost(virtualHostDomains),
		templateOptions: buildTemplateOptions(templateOptions),
		accessLogConfig: buildAccessLogConfig(accessLogConfigUids),
		extraFields: buildExtraFields(extraFields)
	}
}

export const buildCreateVSData = (data: IVirtualServiceForm): CreateVirtualServiceRequest => ({
	...buildBaseVSData(data),
	$typeName: 'virtual_service.v1.CreateVirtualServiceRequest' as const
})

export const buildUpdateVSData = (data: IVirtualServiceForm, uid: string): UpdateVirtualServiceRequest => {
	const { name, ...baseDataWithoutName } = buildBaseVSData(data)

	return {
		...baseDataWithoutName,
		uid,
		$typeName: 'virtual_service.v1.UpdateVirtualServiceRequest' as const
	}
}
