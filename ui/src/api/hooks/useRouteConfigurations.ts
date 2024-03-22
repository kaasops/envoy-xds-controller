import { useQuery } from "@tanstack/react-query"
import GetRouteConfigurationsService from "../services/getRouteConfigurationsService"

export const useGetRouteConfigurations = (nodeID: string, routeConfigurationsName: string, loadDataFlag: boolean) => {
    return useQuery({
        queryKey: ['routeConfigurations', nodeID, routeConfigurationsName, loadDataFlag],
        queryFn: () => GetRouteConfigurationsService.getRouteConfiguration(nodeID, routeConfigurationsName),
        enabled: !!nodeID && loadDataFlag,
        select: (data) => data
    })
}