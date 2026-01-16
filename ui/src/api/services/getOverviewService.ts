import { NodeOverviewResponse } from '../../common/types/overviewApiTypes'
import axiosClient from '../axiosApiClient'

const GetOverviewService = {
	getOverview: async (nodeId: string): Promise<NodeOverviewResponse | undefined> => {
		try {
			const { data } = await axiosClient.get<NodeOverviewResponse>(`/overview?node_id=${nodeId}`)
			return data
		} catch (error: unknown) {
			console.error('Error fetching overview: ', error)
			throw error
		}
	}
}

export default GetOverviewService
