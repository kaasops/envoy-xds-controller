import { ThemeProvider } from '@emotion/react'
import { CssBaseline } from '@mui/material'
import { Suspense, lazy } from 'react'
import { Route, Routes } from 'react-router-dom'
import ErrorBoundary from './components/errorBoundary/ErrorBoundary'
import Spinner from './components/spinner/Spinner'
import Layout from './layout/layout'
import { ColorModeContext } from './theme/theme'
import useThemeMode from './utils/hooks/useThemeMode'

const HomePage = lazy(() => import('./pages/home/Home'))
const NodeInfoPage = lazy(() => import('./pages/nodeInfo/NodeInfo'))
const KuberPage = lazy(() => import('./pages/kuber/KuberPage'))
const Page404 = lazy(() => import('./pages/page404/page404'))

function App() {
	const [theme, colorMode] = useThemeMode()

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

							<Route path='kuber' element={<Layout />}>
								<Route index element={<KuberPage />} />
							</Route>
							<Route path='*' element={<Page404 />} />
						</Routes>
					</ErrorBoundary>
				</Suspense>
			</ThemeProvider>
		</ColorModeContext.Provider>
	)
}

export default App
