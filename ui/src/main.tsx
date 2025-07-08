import React from 'react'
import ReactDOM from 'react-dom/client'
import { AuthProvider } from 'react-oidc-context'
import App from './App.tsx'
import './index.css'
import './utils/monacoEditorSettings/loaderConfig.ts'

import '@fontsource/roboto/300.css'
import '@fontsource/roboto/400.css'
import '@fontsource/roboto/500.css'
import '@fontsource/roboto/700.css'
import { QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import queryClient from './utils/queryClient/queryClient.ts'
import { TransportProvider } from '@connectrpc/connect-query'
import { env } from './env.ts'
import { transport } from './api/grpc/client.ts'

function createApp() {
	const app = (
		<TransportProvider transport={transport}>
			<QueryClientProvider client={queryClient}>
				<BrowserRouter>
					<React.StrictMode>
						<App />
					</React.StrictMode>
				</BrowserRouter>
			</QueryClientProvider>
		</TransportProvider>
	)
	if (env.VITE_OIDC_ENABLED === 'true') {
		const oidcConfig = {
			authority: env.VITE_OIDC_AUTHORITY,
			client_id: env.VITE_OIDC_CLIENT_ID,
			redirect_uri: env.VITE_OIDC_REDIRECT_URI || document.location.origin,
			scope: env.VITE_OIDC_SCOPE
		}
		return <AuthProvider {...oidcConfig}>{app}</AuthProvider>
	}

	// if (import.meta.env.MODE === 'development') {
	// 	import('./api/axiosMock')
	// }

	return app
}

ReactDOM.createRoot(document.getElementById('root')!).render(createApp())
