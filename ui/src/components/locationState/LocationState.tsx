import ExploreTwoToneIcon from '@mui/icons-material/ExploreTwoTone'
import ExploreOffTwoToneIcon from '@mui/icons-material/ExploreOffTwoTone'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import useSetDomainStore from '../../store/setDomainStore'

interface ILocationsStateProps {
	isEmpty: boolean
}

function LocationState({ isEmpty }: ILocationsStateProps) {
	const domain = useSetDomainStore(state => state.domain)
	return (
		<Box
			sx={{
				display: 'flex',
				justifyContent: 'center',
				alignItems: 'center',
				gap: 1
			}}
		>
			{isEmpty ? (
				<>
					<ExploreOffTwoToneIcon sx={{ fontSize: 60 }} />
					<Typography variant='h4'>No locations found for this domain {domain.toUpperCase()}</Typography>
				</>
			) : (
				<>
					<ExploreTwoToneIcon sx={{ fontSize: 60 }} />
					<Typography variant='h4'>To display locations, select the domain on the left</Typography>
				</>
			)}
		</Box>
	)
}

export default LocationState
