import { useQuery } from '@tanstack/react-query'
import GetOverviewService from '../services/getOverviewService'

// Backend caches overview responses for 30 seconds
const OVERVIEW_STALE_TIME = 30 * 1000

export const useOverview = (nodeId: string) => {
	return useQuery({
		queryKey: ['overview', nodeId],
		queryFn: () => GetOverviewService.getOverview(nodeId),
		enabled: !!nodeId,
		staleTime: OVERVIEW_STALE_TIME
	})
}
