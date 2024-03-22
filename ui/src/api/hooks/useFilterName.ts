import { useQuery } from "@tanstack/react-query"
import GetFilterNameService from "../services/getFilterNameService"

export const useGetFilterName = (nodeID: string, listenerName: string, filterChainName: string, filterName: string, loadDataButton: boolean) => {
    return useQuery({
        queryKey: ['filterName', nodeID, listenerName, filterChainName, filterName, loadDataButton],
        queryFn: () => GetFilterNameService.getFilterName(nodeID, listenerName, filterChainName, filterName),
        enabled: !!nodeID && !!listenerName && !!filterChainName && !!filterName && loadDataButton,
        select: (data) => data
    })
}