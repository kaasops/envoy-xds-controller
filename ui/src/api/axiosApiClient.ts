import axios from 'axios'
import { env } from '../env.ts'

const axiosClient = axios.create({
	baseURL: env.VITE_ROOT_API_URL,
	headers: { 'Content-Type': 'application/json' }
})

export default axiosClient
