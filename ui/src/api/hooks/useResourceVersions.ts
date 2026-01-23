import { useQuery } from '@tanstack/react-query'
import GetResourceVersionsService from '../services/getResourceVersionsService'

export const useResourceVersions = (nodeId: string) => {
	return useQuery({
		queryKey: ['resourceVersions', nodeId],
		queryFn: () => GetResourceVersionsService.getResourceVersions(nodeId),
		enabled: !!nodeId,
		staleTime: 30 * 1000
	})
}
