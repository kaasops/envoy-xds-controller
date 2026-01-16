import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import { useParams, useNavigate } from 'react-router-dom'
import { useAllDomains } from '../../api/hooks/useAllDomains'
import DomainSection from '../../components/domainSections/DomainSection'
import Spinner from '../../components/spinner/Spinner'
import SettingsNodeSection from '../../components/settingsNodeSection/SettingsNodeSection'
import DashboardIcon from '@mui/icons-material/Dashboard'

function NodeInfo() {
	const { nodeID } = useParams()
	const navigate = useNavigate()
	const { isFetching } = useAllDomains(nodeID as string)

	const handleOverviewClick = () => {
		navigate(`/nodeIDs/${nodeID}/overview`)
	}

	return (
		<Box component='section' className='RootBoxNodeInfo' sx={{ height: 'calc(100vh - 85px)' }}>
			{isFetching ? (
				<Spinner />
			) : (
				<Box className='NodeInfoWrapper' height='100%'>
					<Box sx={{ p: 2, display: 'flex', justifyContent: 'flex-end' }}>
						<Button
							variant='outlined'
							color='primary'
							startIcon={<DashboardIcon />}
							onClick={handleOverviewClick}
							size='small'
						>
							Overview
						</Button>
					</Box>
					<DomainSection />
					<SettingsNodeSection />
				</Box>
			)}
		</Box>
	)
}

export default NodeInfo
