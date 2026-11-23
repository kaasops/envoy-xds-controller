// The state a sign-in hands back to /callback.
export interface SigninState {
	returnTo?: string
}

const home = '/nodeIDs'

// signinState records the page the user opened, so that a direct link survives the sign-in.
export const signinState = (
	location: Pick<Location, 'pathname' | 'search' | 'hash'> = window.location
): SigninState => ({
	returnTo: location.pathname + location.search + location.hash
})

// returnPath gives the page saved before the sign-in. Anything that is not a path of this application
// leads home instead: another origin, "//host" (browsers read "\" as "/" and drop tabs and newlines),
// or the callback itself.
export const returnPath = (state: unknown): string => {
	const returnTo = (state as SigninState | null | undefined)?.returnTo
	if (
		typeof returnTo !== 'string' ||
		!returnTo.startsWith('/') ||
		returnTo.startsWith('//') ||
		Array.from(returnTo).some(ch => ch === '\\' || ch.charCodeAt(0) < 0x20)
	) {
		return home
	}
	if (returnTo === '/' || /^\/callback(?:[/?#]|$)/.test(returnTo)) {
		return home
	}
	return returnTo
}
