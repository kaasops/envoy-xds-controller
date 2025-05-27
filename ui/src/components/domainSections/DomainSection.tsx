import Box from '@mui/material/Box'
import Grid from '@mui/material/Grid2'
import { useParams } from 'react-router-dom'
import { useAllDomains } from '../../api/hooks/useAllDomains'
import { useGetDomainLocations } from '../../api/hooks/useDomainLocations'
import { IDomainLocationsResponse } from '../../common/types/getDomainLocationsApiTypes'
import useSetDomainStore from '../../store/setDomainStore'
import DomainLocations from '../domainLocations/DomainLocations'
import DomainsList from '../domainsList/DomainsList'
import NotDomainsCard from '../notDomainsCard/NotDomainsCard'
import { styleDomainSection } from './style'
import { useColors } from '../../utils/hooks/useColors'
import { useEffect } from 'react'

function DomainSection() {
	const { colors } = useColors()
	const { nodeID } = useParams()

	//При переходе по NodeIds, что бы сбрасывался выбранный домен ранее
	const setSelectDomain = useSetDomainStore(state => state.setDomainValue)
	useEffect(() => {
		setSelectDomain('')
	}, [setSelectDomain])

	const domain = useSetDomainStore(state => state.domain)
	const { data: domains } = useAllDomains(nodeID as string)
	const { data: domainLocations, isFetching: domainLocationsFetching } = useGetDomainLocations(
		nodeID as string,
		domain as string
	)

	return (
		<Box className='DomainSection' sx={{ ...styleDomainSection, backgroundColor: colors.primary[800] }}>
			{domains?.length !== 0 ? (
				<Grid container height='100%'>
					<Grid size={{ xs: 4, md: 4.5, lg: 4 }} className='DomainSelectSection'>
						<Box padding={3} height='100%'>
							<DomainsList domains={domains as string[]} />
						</Box>
					</Grid>
					<Grid size={{ xs: 8, md: 7.5, lg: 8 }} className='DomainLocationSection' height={'100%'}>
						<Box padding={3} paddingLeft={0} height='100%'>
							<DomainLocations
								locations={domainLocations as IDomainLocationsResponse[]}
								isFetching={domainLocationsFetching}
							/>
						</Box>
					</Grid>
				</Grid>
			) : (
				<NotDomainsCard nodeID={nodeID as string} />
			)}
		</Box>
	)
}

export default DomainSection
