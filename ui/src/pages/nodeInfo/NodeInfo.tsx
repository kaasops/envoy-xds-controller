import Box from '@mui/material/Box'
import { useParams } from 'react-router-dom'
import { useAllDomains } from '../../api/hooks/useAllDomains'
import DomainSection from '../../components/domainSections/DomainSection'
import Spinner from '../../components/spinner/Spinner'
import SettingsNodeSection from '../../components/settingsNodeSection/SettingsNodeSection'

function NodeInfo() {
	const { nodeID } = useParams()
	const { isFetching } = useAllDomains(nodeID as string)

	return (
		<Box component='section' className='RootBoxNodeInfo' sx={{ height: 'calc(100vh - 85px)' }}>
			{isFetching ? (
				<Spinner />
			) : (
				<Box className='NodeInfoWrapper' height='100%'>
					<DomainSection />
					<SettingsNodeSection />
				</Box>
			)}
		</Box>
	)
}

export default NodeInfo
