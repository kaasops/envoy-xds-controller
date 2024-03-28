import NodeIDsApiService from '../services/nodeIDService'
import { useQuery } from '@tanstack/react-query'

export const useNodeIDs = () => {
	return useQuery({
		queryKey: ['nodeIDs'],
		queryFn: NodeIDsApiService.getNodeIDs,
		select: ({ data }) => data
	})
}
