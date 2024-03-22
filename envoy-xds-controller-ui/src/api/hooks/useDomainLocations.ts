import { useQuery } from "@tanstack/react-query"
import GetDomainLocationsService from "../services/getDomainLocations"
import { IGetDomainLocationsResponse } from "../../common/types/getDomainLocationsApiTypes"

export const useGetDomainLocations = (nodeID: string, domain: string) => {
    return useQuery({
        queryKey: ['domainLocations', nodeID, domain],
        queryFn: () => GetDomainLocationsService.getDomainLocations(nodeID, domain),
        enabled: !!nodeID && !!domain && domain !== '',
        select: (data) => (data as IGetDomainLocationsResponse).locations
    })
}