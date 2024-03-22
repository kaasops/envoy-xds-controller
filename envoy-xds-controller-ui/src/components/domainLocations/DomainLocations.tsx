import { Box, CircularProgress, Container, List } from '@mui/material'
import { IDomainLocationsResponse } from '../../common/types/getDomainLocationsApiTypes'
import useSetDomainStore from '../../store/setDomainStore'
import LocationCard from '../locationCard/LocationCard'
import LocationState from '../locationState/LocationState'
import { styleDomainLocationsBox, styleList } from './style'

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
                    ) : (
                        locations?.length !== 0 ? (
                            <Container style={{
                                height: '100%',
                                overflow: 'auto',
                                margin: 0,
                                display: 'flex',
                            }}>
                                <List sx={{ ...styleList }}>
                                    {locations?.map((domain, index) => (
                                        <LocationCard key={index} domain={domain} />
                                    ))}
                                </List>
                            </Container>
                        ) : (
                            <LocationState isEmpty={true} />
                        )
                    )}
                </>
            )}
        </Box>
    )
}

export default DomainLocations