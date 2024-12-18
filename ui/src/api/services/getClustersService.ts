import { IClustersResponse } from '../../common/types/getClustersApiTypes'
import axiosClient from '../axiosApiClient'

const GetClustersService = {
	getClusters: async (nodeId: string, clustersName: string) => {
		try {
			const { data } = await axiosClient.get<IClustersResponse>(
				`/clusters?node_id=${nodeId}&cluster_name=${clustersName}`
			)
			return data
		} catch (error: any) {
			console.error('Error: ', error)
		}
	}
}

export default GetClustersService
