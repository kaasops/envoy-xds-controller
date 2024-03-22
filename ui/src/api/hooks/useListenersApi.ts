import { useQuery } from "@tanstack/react-query"
import GetListenersService from "../services/getListenersService"

export const useGetListenerApi = (nodeId: string, listenerName: string, loadDataFlag: boolean) => {
    return useQuery({
        queryKey: ['listener', nodeId, listenerName, loadDataFlag],
        queryFn: () => GetListenersService.getListeners(nodeId, listenerName),
        enabled: !!nodeId && !!listenerName && listenerName !== '' && loadDataFlag,
        select: (data) => data
    })
}

export const useGetAllListenersApi = (nodeId: string, listenerName: string, loadDataFlag: boolean) => {
    return useQuery({
        queryKey: ['listeners', nodeId, listenerName, loadDataFlag],
        queryFn: () => GetListenersService.getListeners(nodeId, listenerName),
        enabled: !!nodeId && loadDataFlag,
        select: (data) => data
    })
}
