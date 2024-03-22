import { useQuery } from "@tanstack/react-query";
import GetFilterChainService from "../services/getFilterChainService";

export const useGetFilterChain = (nodeID: string, listenerName: string, filterChainName: string, loadDataButton: boolean) => {
    return useQuery({
        queryKey: ['filterChain', nodeID, listenerName, filterChainName, loadDataButton],
        queryFn: () => GetFilterChainService.getFilterChain(nodeID, listenerName, filterChainName),
        enabled: !!nodeID && !!listenerName && !!filterChainName && loadDataButton,
        select: (data) => data
    })
}