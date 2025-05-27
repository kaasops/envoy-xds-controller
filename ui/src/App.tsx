import { ThemeProvider } from '@emotion/react'
import { useAuth } from 'react-oidc-context'
import { CssBaseline } from '@mui/material'
import { lazy, Suspense, useEffect } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import ErrorBoundary from './components/errorBoundary/ErrorBoundary'
import Spinner from './components/spinner/Spinner'
import Layout from './layout/layout'

import { ColorModeContext } from './theme/theme'
import useThemeMode from './utils/hooks/useThemeMode'
import { env } from './env.ts'
import { provideAuth } from './utils/helpers/authBridge.ts'
import ErrorMessage from './components/errorMessage/ErrorMessage.tsx'
import { ThemedWrapper } from './components/themeWrapper/themeWrapper.tsx'
import { useGetPermissions } from './api/grpc/hooks/useVirtualService.ts'
import { usePermissionsStore } from './store/permissionsStore.ts'

const HomePage = lazy(() => import('./pages/home/Home'))
const NodeInfoPage = lazy(() => import('./pages/nodeInfo/NodeInfo'))
const AccessGroupsPage = lazy(() => import('./pages/accessGroupsPage/accessGroupsPage'))
const VirtualServicesPage = lazy(() => import('./pages/virtualServicesPage/virtualServicesPage'))
const EditVsPage = lazy(() => import('./pages/editVsPage/editVsPage'))
const CreateVsPage = lazy(() => import('./pages/createVsPage/createVsPage'))
const Page404 = lazy(() => import('./pages/page404/page404'))

function App() {
	const [theme, colorMode] = useThemeMode()
	const auth = useAuth()
	const { getPermissions } = useGetPermissions()
	const setPermissions = usePermissionsStore(state => state.setPermissions)

	useEffect(() => {
		if (env.VITE_OIDC_ENABLED === 'true' && auth.isAuthenticated && auth.user) {
			const fetchPermissions = async () => {
				provideAuth(auth)
				try {
					const permissions = await getPermissions()
					setPermissions(permissions.items)
				} catch (error) {
					console.error('Error while getting permissions:', error)
				}
			}

			void fetchPermissions()
		}
	}, [auth, auth?.isAuthenticated, auth?.user, getPermissions, setPermissions])

	if (env.VITE_OIDC_ENABLED === 'true') {
		if (auth.isLoading) {
			return (
				<ThemedWrapper theme={theme} colorMode={colorMode}>
					<Spinner />
				</ThemedWrapper>
			)
		}

		if (auth.error) {
			return (
				<ThemedWrapper theme={theme} colorMode={colorMode}>
					<ErrorMessage error={auth.error.message} />
				</ThemedWrapper>
			)
		}

		if (!auth.isAuthenticated) {
			void auth.signinRedirect()
			return (
				<ThemedWrapper theme={theme} colorMode={colorMode}>
					<div>Redirect to login...</div>
				</ThemedWrapper>
			)
		}
	}

	return (
		<ColorModeContext.Provider value={colorMode}>
			<ThemeProvider theme={theme}>
				<CssBaseline enableColorScheme />
				<Suspense fallback={<Spinner />}>
					<ErrorBoundary>
						<Routes>
							<Route path='nodeIDs' element={<Layout />}>
								<Route index element={<HomePage />} />
								<Route path=':nodeID' element={<NodeInfoPage />} />
							</Route>

							<Route path='accessGroups' element={<Layout />}>
								<Route index element={<AccessGroupsPage />} />

								<Route path=':groupId'>
									<Route index element={<Navigate to='virtualServices' replace />} />

									<Route path='virtualServices'>
										<Route index element={<VirtualServicesPage />} />
										<Route path='createVs' element={<CreateVsPage />} />
										<Route path=':uid' element={<EditVsPage />} />
									</Route>
								</Route>
							</Route>

							<Route path='callback' element={<Navigate to='/nodeIDs' replace />} />
							<Route path='*' element={<Page404 />} />
						</Routes>
					</ErrorBoundary>
				</Suspense>
			</ThemeProvider>
		</ColorModeContext.Provider>
	)
}

export default App
