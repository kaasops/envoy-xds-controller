import { UseFormReset } from 'react-hook-form'
import { IVirtualServiceForm } from '../../components/virtualServiceForm/types.ts'
import { useCallback, useEffect } from 'react'
import { ResourceRef } from '../../gen/common/v1/common_pb.ts'
import { GetVirtualServiceResponse } from '../../gen/virtual_service/v1/virtual_service_pb'

interface ISetDefaultValuesVSFormProps {
	reset: UseFormReset<IVirtualServiceForm>
	isCreate: boolean
	virtualServiceInfo: GetVirtualServiceResponse | undefined
}

export const useSetDefaultValuesVSForm = ({ reset, isCreate, virtualServiceInfo }: ISetDefaultValuesVSFormProps) => {
	const setDefaultValues = useCallback(() => {
		if (isCreate || !virtualServiceInfo) return

		const vhDomains = virtualServiceInfo?.virtualHost?.domains || []

		reset({
			name: virtualServiceInfo.name,
			nodeIds: virtualServiceInfo.nodeIds || [],
			accessGroup: virtualServiceInfo.accessGroup,
			templateUid: virtualServiceInfo.template?.uid,
			listenerUid: virtualServiceInfo.listener?.uid,
			accessLogConfigUid: (virtualServiceInfo.accessLog?.value as ResourceRef)?.uid || '',
			useRemoteAddress: virtualServiceInfo.useRemoteAddress,
			templateOptions: virtualServiceInfo.templateOptions,
			virtualHostDomains: vhDomains,
			additionalHttpFilterUids: virtualServiceInfo.additionalHttpFilters?.map(filter => filter.uid) || [],
			additionalRouteUids: virtualServiceInfo.additionalRoutes?.map(router => router.uid) || [],
			description: virtualServiceInfo.description
		})
	}, [reset, isCreate, virtualServiceInfo])

	useEffect(() => {
		setDefaultValues()
	}, [setDefaultValues])

	return { setDefaultValues }
}
