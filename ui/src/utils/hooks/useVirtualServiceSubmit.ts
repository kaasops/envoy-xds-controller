import { IVirtualServiceForm } from '../../components/virtualServiceForm'
import { buildCreateVSData, buildUpdateVSData } from '../helpers'
import { QueryObserverResult, RefetchOptions, UseMutateAsyncFunction } from '@tanstack/react-query'
import { Message } from '@bufbuild/protobuf'
import {
	CreateVirtualServiceRequest,
	GetVirtualServiceResponse,
	ListVirtualServicesResponse,
	UpdateVirtualServiceRequest
} from '../../gen/virtual_service/v1/virtual_service_pb'
import { useNavigate } from 'react-router-dom'

interface IUseVirtualServiceSubmit {
	isCreate: boolean
	createVirtualService: UseMutateAsyncFunction<
		Message<'virtual_service.v1.CreateVirtualServiceResponse'>,
		Error,
		CreateVirtualServiceRequest
	>
	virtualServiceInfo?: GetVirtualServiceResponse | undefined
	updateVS: UseMutateAsyncFunction<
		Message<'virtual_service.v1.UpdateVirtualServiceResponse'>,
		Error,
		UpdateVirtualServiceRequest
	>
	resetQueryUpdateVs: () => void
	groupId: string | undefined
	refetch: (options?: RefetchOptions | undefined) => Promise<QueryObserverResult<ListVirtualServicesResponse, Error>>
}

export const useVirtualServiceSubmit = ({
	isCreate,
	createVirtualService,
	virtualServiceInfo,
	updateVS,
	resetQueryUpdateVs,
	groupId,
	refetch
}: IUseVirtualServiceSubmit) => {
	const navigate = useNavigate()

	const submitVSService = async (data: IVirtualServiceForm) => {
		if (isCreate) {
			const createData = buildCreateVSData(data)
			await createVirtualService(createData)
		} else if (!isCreate && virtualServiceInfo) {
			const updateData = buildUpdateVSData(data, virtualServiceInfo.uid)
			await updateVS(updateData)
			resetQueryUpdateVs()
		}

		navigate(`/accessGroups/${groupId}/virtualServices`, {
			state: {
				successMessage: `Virtual Service ${data.name.toUpperCase()} ${
					isCreate ? 'created' : 'update'
				} successfully`
			}
		})

		await refetch()
	}

	return { submitVSService }
}
