import { ResourceHashVersions } from '../../common/types/overviewApiTypes'
import axiosClient from '../axiosApiClient'

const GetResourceVersionsService = {
	getResourceVersions: async (nodeId: string): Promise<ResourceHashVersions | undefined> => {
		try {
			const { data } = await axiosClient.get<ResourceHashVersions>(
				`/resourceVersions?nodeID=${nodeId}`
			)
			return data
		} catch (error: unknown) {
			console.error('Error fetching resource versions: ', error)
			throw error
		}
	}
}

export default GetResourceVersionsService
