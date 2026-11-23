import { NodeOverviewResponse } from '../../common/types/overviewApiTypes'
import axiosClient from '../axiosApiClient'

const GetOverviewService = {
	getOverview: async (nodeId: string, signal?: AbortSignal): Promise<NodeOverviewResponse> => {
		const { data } = await axiosClient.get<NodeOverviewResponse>('/overview', {
			params: { node_id: nodeId },
			signal
		})
		return data
	}
}

export default GetOverviewService
