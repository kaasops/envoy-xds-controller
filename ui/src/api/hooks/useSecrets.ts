import { useQuery } from "@tanstack/react-query"
import GetSecretsService from "../services/getSecretsService"

export const useGetSecretsApi = (nodeId: string, secretName: string, loadDataFlag: boolean) => {
    return useQuery({
        queryKey: ['secrets', nodeId, secretName, loadDataFlag],
        queryFn: () => GetSecretsService.getSecrets(nodeId, secretName),
        enabled: !!nodeId && loadDataFlag,
        select: (data) => data
    })
}