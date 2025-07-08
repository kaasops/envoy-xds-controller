import axios from 'axios'
import { env } from '../env.ts'
import { getAuth } from '../utils/helpers/authBridge.ts'

const axiosClient = axios.create({
	baseURL: env.VITE_ROOT_API_URL || '/api/v1',
	headers: { 'Content-Type': 'application/json' }
})

let counter401Error = 0
const MAX_401_BEFORE_REDIRECT = 1

axiosClient.interceptors.request.use(config => {
	try {
		const auth = getAuth()
		const token = auth.user?.access_token
		if (token && config.headers) {
			config.headers.Authorization = `Bearer ${token}`
		}
	} catch (err) {
		console.warn('Auth not ready during request')
	}
	return config
})

axiosClient.interceptors.response.use(
	response => {
		counter401Error = 0
		return response
	},
	async error => {
		if (error.response?.status === 401) {
			counter401Error += 1

			const auth = getAuth()

			if (counter401Error < MAX_401_BEFORE_REDIRECT) {
				try {
					await auth.signinSilent()
					const newToken = auth.user?.access_token

					if (newToken) {
						error.config.headers.Authorization = `Bearer ${newToken}`
						return axiosClient(error.config)
					}
				} catch (silentErr) {
					console.warn('The number of attempts has expired, re-receive the token:', silentErr)
				}
			}

			try {
				await auth.signinRedirect()
			} catch (redirectErr) {
				console.error('Redirect to login:', redirectErr)
			}
		}

		return Promise.reject(error)
	}
)

export default axiosClient
