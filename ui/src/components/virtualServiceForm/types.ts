import { GetVirtualServiceResponse } from '../../gen/virtual_service/v1/virtual_service_pb.ts'

export interface IVirtualServiceFormProps {
	virtualServiceInfo?: GetVirtualServiceResponse
	isEdit?: boolean
	iseEditDomain?: boolean
}

export interface ITemplateOption {
	field: string
	modifier: number
}

export interface IVirtualServiceForm {
	name: string
	description: string
	nodeIds: string[]
	accessGroup: string
	templateUid: string
	listenerUid: string
	virtualHostDomains: string[]
	accessLogConfigUid: string
	additionalHttpFilterUids: string[]
	additionalRouteUids: string[]
	useRemoteAddress: boolean | undefined
	templateOptions: ITemplateOption[]
	viewTemplateMode: boolean
	virtualHostDomainsMode: boolean
	additionalHttpFilterMode: boolean
	additionalRouteMode: boolean
}
