import { useQuery } from "@tanstack/react-query"
import GetClustersService from "../services/getClustersService"

export const useGetClustersApi = (nodeId: string, clustersName: string, loadDataFlag: boolean) => {
    return useQuery({
        queryKey: ['clusters', nodeId, clustersName, loadDataFlag],
        queryFn: () => GetClustersService.getClusters(nodeId, clustersName),
        enabled: !!nodeId && loadDataFlag,
        select: (data) => data
    })
}