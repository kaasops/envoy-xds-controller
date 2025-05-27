import Box from '@mui/material/Box'
import CircularProgress from '@mui/material/CircularProgress'
import Container from '@mui/material/Container'

import { IDomainLocationsResponse } from '../../common/types/getDomainLocationsApiTypes'
import useSetDomainStore from '../../store/setDomainStore'
import LocationCard from '../locationCard/LocationCard'
import LocationState from '../locationState/LocationState'
import { styleDomainLocationsBox, styleList } from './style'
import List from '@mui/material/List'

interface IDomainLocationsProps {
	locations: IDomainLocationsResponse[]
	isFetching: boolean
}

function DomainLocations({ locations, isFetching }: IDomainLocationsProps) {
	const selectedDomain = useSetDomainStore(state => state.domain)

	return (
		<Box sx={{ ...styleDomainLocationsBox }}>
			{selectedDomain === '' ? (
				<LocationState isEmpty={false} />
			) : (
				<>
					{isFetching ? (
						<CircularProgress size={100} />
					) : locations?.length !== 0 ? (
						<Container
							style={{
								height: '100%',
								width: '100%',
								margin: 0,
								overflow: 'auto',
								padding: 1
								// display: 'flex',
							}}
						>
							<List sx={{ ...styleList }} className='ListLocationsCard'>
								{locations?.map((domain, index) => <LocationCard key={index} domain={domain} />)}
							</List>
						</Container>
					) : (
						<LocationState isEmpty={true} />
					)}
				</>
			)}
		</Box>
	)
}

export default DomainLocations
