import axios from 'axios'
import { env } from '../env.ts'

const axiosClient = axios.create({
	baseURL: env.VITE_ROOT_API_URL || '/api/v1',
	headers: { 'Content-Type': 'application/json' }
})

export default axiosClient

export function setAccessToken(token: string | undefined) {
	axiosClient.interceptors.request.use(
		config => {
			config.headers['Authorization'] = `Bearer ${token}`
			return config
		},
		error => {
			return Promise.reject(error)
		}
	)
}
