import type { AuthContextProps } from 'react-oidc-context'

let _auth: AuthContextProps | null = null

export const provideAuth = (auth: AuthContextProps) => {
	_auth = auth
}

export const getAuth = (): AuthContextProps => {
	if (!_auth) {
		throw new Error('[authBridge] Auth is not initialized. Call provideAuth() first.')
	}
	return _auth
}
