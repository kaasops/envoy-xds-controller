import { IAllDomainsResponse } from '../../common/types/allDomainsApiTypes'
import axiosClient from '../axiosApiClient'

const GetAllDomainsService = {
	getAllDomains: async (nodeId: string) => {
		try {
			const { data } = await axiosClient.get<IAllDomainsResponse>(`/domains?node_id=${nodeId}`)
			return data
		} catch (error: any) {
			console.error('Error fetching data:', error)

			if (error.response && error.response.status === 500) {
				// Возвращаем пустой объект в случае ошибки 500
				return { domains: [] }
			} else {
				// Возвращаем пустой объект или другие данные по умолчанию в случае других ошибок
				return { domains: [] }
			}
		}
	}
}

export default GetAllDomainsService
