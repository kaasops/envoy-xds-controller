declare global {
	interface Window {
		env: any
	}
}

type EnvType = {
	VITE_ROOT_API_URL: string
}

export const env: EnvType = { ...import.meta.env, ...window.env }
