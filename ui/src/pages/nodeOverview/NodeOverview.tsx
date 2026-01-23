import { useState } from 'react'
import { useParams, useNavigate, Navigate } from 'react-router-dom'
import { Box, Typography, Tab, Tabs, IconButton, CircularProgress, Alert } from '@mui/material'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import RefreshIcon from '@mui/icons-material/Refresh'
import { useOverview } from '../../api/hooks/useOverview'
import { OverviewSummary } from '../../components/overviewSummary'
import { ResourceVersions } from '../../components/resourceVersions'
import { EndpointsTable } from '../../components/endpointsTable'
import { CertificatesTable } from '../../components/certificatesTable'
import { CustomTabPanel } from '../../components/customTabPanel'

function a11yProps(index: number) {
	return {
		id: `overview-tab-${index}`,
		'aria-controls': `minimal-tabpanel-${index}`
	}
}

const NodeOverview = () => {
	const { nodeID } = useParams<{ nodeID: string }>()
	const navigate = useNavigate()
	const [tabValue, setTabValue] = useState(0)

	// Hook must be called unconditionally, enabled flag handles missing nodeID
	const { data: overview, isLoading, isError, error, refetch } = useOverview(nodeID ?? '')

	const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
		setTabValue(newValue)
	}

	const handleBack = () => {
		navigate(`/nodeIDs/${nodeID}`)
	}

	// Redirect if nodeID is missing (after hooks)
	if (!nodeID) {
		return <Navigate to='/nodeIDs' replace />
	}

	if (isLoading) {
		return (
			<Box display='flex' justifyContent='center' alignItems='center' minHeight='50vh'>
				<CircularProgress />
			</Box>
		)
	}

	if (isError) {
		return (
			<Box sx={{ p: 3 }}>
				<Alert severity='error'>
					Error loading overview: {error instanceof Error ? error.message : 'Unknown error'}
				</Alert>
			</Box>
		)
	}

	if (!overview) {
		return (
			<Box sx={{ p: 3 }}>
				<Alert severity='warning'>No overview data available</Alert>
			</Box>
		)
	}

	return (
		<Box sx={{ p: 3 }}>
			{/* Header */}
			<Box display='flex' alignItems='center' justifyContent='space-between' mb={3}>
				<Box display='flex' alignItems='center' gap={2}>
					<IconButton onClick={handleBack} size='small'>
						<ArrowBackIcon />
					</IconButton>
					<Typography variant='h5' component='h1'>
						Node Overview: <code>{nodeID}</code>
					</Typography>
				</Box>
				<IconButton onClick={() => refetch()} size='small' title='Refresh'>
					<RefreshIcon />
				</IconButton>
			</Box>

			{/* Resource Versions */}
			<Box sx={{ mb: 2 }}>
				<ResourceVersions versions={overview.resourceVersions} />
			</Box>

			{/* Summary Cards */}
			<OverviewSummary summary={overview.summary} />

			{/* Tabs */}
			<Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
				<Tabs value={tabValue} onChange={handleTabChange} aria-label='Overview tabs'>
					<Tab label={`Endpoints (${overview.endpoints.length})`} {...a11yProps(0)} />
					<Tab label={`Certificates (${overview.certificates.length})`} {...a11yProps(1)} />
				</Tabs>
			</Box>

			{/* Tab Panels */}
			<CustomTabPanel value={tabValue} index={0} variant='minimal'>
				<EndpointsTable endpoints={overview.endpoints} isLoading={isLoading} />
			</CustomTabPanel>
			<CustomTabPanel value={tabValue} index={1} variant='minimal'>
				<CertificatesTable certificates={overview.certificates} isLoading={isLoading} />
			</CustomTabPanel>
		</Box>
	)
}

export default NodeOverview
