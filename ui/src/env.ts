declare global {
	interface Window {
		env: any
	}
}

type EnvType = {
	VITE_ROOT_API_URL: string
	VITE_GRPC_API_URL: string
	VITE_OIDC_ENABLED: string
	VITE_OIDC_CLIENT_ID: string
	VITE_OIDC_AUTHORITY: string
	VITE_OIDC_REDIRECT_URI: string
	VITE_OIDC_SCOPE: string
}

export const env: EnvType = { ...import.meta.env, ...window.env }
