import React from 'react'
import { useColors } from '../../utils/hooks/useColors.ts'
import Box from '@mui/material/Box'
import { styleRootBox, styleWrapperCards } from './style.ts'
import { useAccessGroupsVs } from '../../api/grpc/hooks/useVirtualService.ts'
import Spinner from '../../components/spinner/Spinner.tsx'
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline'
import Typography from '@mui/material/Typography'
import Button from '@mui/material/Button'
import AccessOrNodeCard from '../../components/accessOrNodeCard/accessOrNodeCard.tsx'

const AccessGroupsPage: React.FC = () => {
	const { colors } = useColors()
	const {
		data: accessGroups,
		isFetching: isFetchingAccessGroups,
		isError: isErrorAccessGroups,
		error,
		refetch
	} = useAccessGroupsVs()

	return (
		<Box
			component='section'
			className='accessGroupPage'
			sx={{ ...styleRootBox, backgroundColor: colors.primary[800] }}
		>
			{isErrorAccessGroups ? (
				<Box
					display='flex'
					flexDirection='column'
					alignItems='center'
					justifyContent='center'
					height='100%'
					textAlign='center'
					gap={2}
					sx={{ py: 5 }}
				>
					<ErrorOutlineIcon color='error' sx={{ fontSize: 60 }} />
					<Typography variant='h6' color='error'>
						Error loading Access Groups
					</Typography>
					<Typography color='error'>{error.message}</Typography>
					<Typography variant='body2' color='text.secondary'>
						Please try again later.
					</Typography>
					<Button variant='outlined' onClick={() => refetch()}>
						Try again
					</Button>
				</Box>
			) : !isFetchingAccessGroups ? (
				<Box sx={{ ...styleWrapperCards }}>
					{accessGroups?.items?.map(group => <AccessOrNodeCard entity={group.name} key={group.name} />)}
				</Box>
			) : (
				<Spinner />
			)}
		</Box>
	)
}

export default AccessGroupsPage
