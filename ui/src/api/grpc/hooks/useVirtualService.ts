import { useMutation, useQuery } from '@tanstack/react-query'
import {
	accessGroupsServiceClient,
	accessLogServiceClient,
	httpFilterServiceClient,
	listenerServiceClient,
	nodeServiceClient,
	permissionsServiceClient,
	routeServiceClient,
	templateServiceClient,
	utilServiceClient,
	virtualServiceClient
} from '../client.ts'
import {
	CreateVirtualServiceRequest,
	UpdateVirtualServiceRequest
} from '../../../gen/virtual_service/v1/virtual_service_pb.ts'
import { FillTemplateRequest } from '../../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'

export const useListVs = (flag: boolean, accessGroup?: string) => {
	const safeAccessGroup = accessGroup ?? ''

	return useQuery({
		queryKey: ['listVs', safeAccessGroup],
		queryFn: () =>
			virtualServiceClient.listVirtualServices(
				{
					accessGroup: safeAccessGroup
				}
				// metadata
			),
		enabled: flag
	})
}

export const useGetVs = (uid: string) => {
	return useQuery({
		queryKey: ['getVs', uid],
		queryFn: () => virtualServiceClient.getVirtualService({ uid }),
		gcTime: 0
	})
}

export const useCreateVs = () => {
	const createVirtualServiceMutation = useMutation({
		mutationKey: ['createVs'],
		mutationFn: (vsCreateData: CreateVirtualServiceRequest) =>
			virtualServiceClient.createVirtualService(vsCreateData)
	})

	return {
		createVirtualService: createVirtualServiceMutation.mutateAsync,
		errorCreateVs: createVirtualServiceMutation.error,
		isFetchingCreateVs: createVirtualServiceMutation.isPending
	}
}

export const useUpdateVs = () => {
	const updateVsMutations = useMutation({
		mutationKey: ['update'],
		mutationFn: (vsUpdateData: UpdateVirtualServiceRequest) =>
			virtualServiceClient.updateVirtualService(vsUpdateData)
	})

	return {
		updateVS: updateVsMutations.mutateAsync,
		successUpdateVs: updateVsMutations.isSuccess,
		errorUpdateVs: updateVsMutations.error,
		isFetchingUpdateVs: updateVsMutations.isPending,
		resetQueryUpdateVs: updateVsMutations.reset
	}
}

export const useDeleteVs = () => {
	const deleteVirtualServiceMutation = useMutation({
		mutationKey: ['deleteVs'],
		mutationFn: (uid: string) => virtualServiceClient.deleteVirtualService({ uid })
	})

	return {
		deleteVirtualService: deleteVirtualServiceMutation.mutateAsync,
		errorDeleteVs: deleteVirtualServiceMutation.error
	}
}

export const useAccessGroupsVs = () => {
	return useQuery({
		queryKey: ['accessGroupsVs'],
		queryFn: () => accessGroupsServiceClient.listAccessGroups({})
	})
}

export const useAccessLogsVs = (accessGroup?: string) => {
	return useQuery({
		queryKey: ['accessLogsVs', accessGroup],
		queryFn: () => accessLogServiceClient.listAccessLogConfigs({ accessGroup: accessGroup || '' })
	})
}

export const useHttpFilterVs = (accessGroup?: string) => {
	return useQuery({
		queryKey: ['httpFilterVs', accessGroup],
		queryFn: () => httpFilterServiceClient.listHTTPFilters({ accessGroup: accessGroup || '' })
	})
}

export const useListenerVs = (accessGroup?: string) => {
	return useQuery({
		queryKey: ['listenerVs', accessGroup],
		queryFn: () => listenerServiceClient.listListeners({ accessGroup: accessGroup || '' })
	})
}

export const useRouteVs = (accessGroup?: string) => {
	return useQuery({
		queryKey: ['routeVs', accessGroup],
		queryFn: () => routeServiceClient.listRoutes({ accessGroup: accessGroup || '' })
	})
}

export const useTemplatesVs = (accessGroup?: string) => {
	return useQuery({
		queryKey: ['templatesVs', accessGroup],
		queryFn: () => templateServiceClient.listVirtualServiceTemplates({ accessGroup: accessGroup || '' })
	})
}

export const useNodeListVs = (accessGroup?: string) => {
	return useQuery({
		queryKey: ['nodeListVs', accessGroup],
		queryFn: () => nodeServiceClient.listNodes({ accessGroup: accessGroup || '' })
	})
}

export const useVerifyDomains = (domains: string[]) => {
	return useQuery({
		queryKey: ['verifyDomains', domains],
		queryFn: () => utilServiceClient.verifyDomains({ domains: domains }),
		select: data => data,
		enabled: !!domains
	})
}

export const useFillTemplate = () => {
	const fillTemplateMutation = useMutation({
		mutationKey: ['fillTemplate'],
		mutationFn: (data: FillTemplateRequest) => templateServiceClient.fillTemplate(data)
	})

	return {
		getTemplate: fillTemplateMutation.mutate,
		fillTemplate: fillTemplateMutation.mutateAsync,
		isLoadingFillTemplate: fillTemplateMutation.isPending,
		rawData: fillTemplateMutation.data,
		errorFillTemplate: fillTemplateMutation.error
	}
}

export const useGetPermissions = () => {
	const getPermissionsMutation = useMutation({
		mutationKey: ['permissions'],
		mutationFn: () => permissionsServiceClient.listPermissions({})
	})

	return {
		getPermissions: getPermissionsMutation.mutateAsync
	}
}
