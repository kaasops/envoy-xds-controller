import { Component, ErrorInfo, ReactNode } from 'react'
import ErrorMessage from '../errorMessage/ErrorMessage.tsx'

interface Props {
	children?: ReactNode
}

interface State {
	hasError: boolean
	error: Error | null
}

class ErrorBoundary extends Component<Props, State> {
	state: State = {
		hasError: false,
		error: null
	}

	componentDidCatch(error: Error, errorInfo: ErrorInfo) {
		console.error('Caught by ErrorBoundary:', error, errorInfo)
		this.setState({ hasError: true, error })
	}

	render() {
		const { hasError, error } = this.state

		if (hasError && error) {
			return <ErrorMessage error={error.message} />
		}

		return this.props.children
	}
}

export default ErrorBoundary
