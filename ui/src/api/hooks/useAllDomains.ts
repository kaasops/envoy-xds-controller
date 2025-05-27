import { useQuery } from '@tanstack/react-query'
import GetAllDomainsService from '../services/getAllDomainsService'

export const useAllDomains = (nodeId: string) => {
	return useQuery({
		queryKey: ['domains', nodeId],
		queryFn: () => GetAllDomainsService.getAllDomains(nodeId),
		enabled: !!nodeId,
		select: ({ domains }) => domains
	})
}
