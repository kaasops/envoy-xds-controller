import DomainDisabledTwoToneIcon from '@mui/icons-material/DomainDisabledTwoTone'
import { Box, Typography } from '@mui/material'
import { styleNotDomainsBox } from './style'

interface INotDomainCard {
	nodeID: string
}

function NotDomainsCard({ nodeID }: INotDomainCard) {
	return (
		<Box padding={3} width='100%' height='100%'>
			<Box sx={{ ...styleNotDomainsBox }}>
				<DomainDisabledTwoToneIcon sx={{ fontSize: 60 }} />
				<Typography variant='h4'>No domains were found for node {nodeID.toLocaleUpperCase()}</Typography>
			</Box>
		</Box>
	)
}

export default NotDomainsCard
