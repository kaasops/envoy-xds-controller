import React from 'react';
import ReactDOM from 'react-dom/client';
import { AuthProvider } from 'react-oidc-context';
import App from './App.tsx';
import './index.css';

import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';
import { QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import queryClient from './utils/queryClient/queryClient.ts';
import { env } from './env.ts'

const oidcConfig = {
	authority: env.VITE_OIDC_AUTHORITY,
	client_id: env.VITE_OIDC_CLIENT_ID,
	redirect_uri: env.VITE_OIDC_REDIRECT_URI || document.location.origin,
	scope: env.VITE_OIDC_SCOPE
}

const app = (
	<QueryClientProvider client={queryClient}>
		<BrowserRouter>
			<React.StrictMode>
				<App />
			</React.StrictMode>
		</BrowserRouter>
	</QueryClientProvider>
)

ReactDOM.createRoot(document.getElementById('root')!).render(
	env.VITE_OIDC_ENABLED ? <AuthProvider {...oidcConfig}>{app}</AuthProvider> : app
)
