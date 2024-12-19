import { ICertificatesResponse, ISecretsResponse } from '../../common/types/getSecretsApiTypes'
import axiosClient from '../axiosApiClient'
import { AxiosResponse } from 'axios'

const GetSecretsService = {
	getSecrets: async (nodeId: string, secretName: string) => {
		try {
			const { data } = await axiosClient.get<ISecretsResponse>(
				`/secrets?node_id=${nodeId}&secret_name=${secretName}`
			)
			return data
		} catch (error: any) {
			console.error('Error: ', error)
		}
	},

	getSecretCerts: async (nameSpace: string | null, name: string | null) => {
		const { data } = await axiosClient.get<any, AxiosResponse<ICertificatesResponse>>(
			`/secrets/${nameSpace}/${name}`
		)
		return data
	}
}

export default GetSecretsService
