import { Navigate, useLocation } from 'react-router-dom'

export default function ConsoleIndex() {
	const { search, hash } = useLocation()
	return <Navigate to={{ pathname: '/beta/nodes', search, hash }} replace />
}
