import { IBuildInfoResponse } from '../../common/types/buildInfoTypes'
import axiosClient from '../axiosApiClient'

const GetBuildInfoService = {
	getBuildInfo: async () => {
		try {
			const { data } = await axiosClient.get<IBuildInfoResponse>('/buildinfo')
			return data
		} catch (error: any) {
			console.error('Error fetching build info: ', error)
		}
	}
}

export default GetBuildInfoService
