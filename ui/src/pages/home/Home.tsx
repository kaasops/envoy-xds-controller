import Box from '@mui/material/Box'
import { useNodeIDs } from '../../api/hooks/useNodeIDsApi'
import AccessOrNodeCard from '../../components/accessOrNodeCard/accessOrNodeCard.tsx'
import Spinner from '../../components/spinner/Spinner'
import { useColors } from '../../utils/hooks/useColors'
import { styleRootBox, styleWrapperCards } from './style'
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline'
import Typography from '@mui/material/Typography'
import Button from '@mui/material/Button'

const Home = () => {
	const { colors } = useColors()

	const { data: nodes, isFetching: isFetchingNodeIds, isError: isErrorNodeIds, refetch, error } = useNodeIDs()

	return (
		<Box
			component='section'
			className='accessGroupPage'
			sx={{ ...styleRootBox, backgroundColor: colors.primary[800] }}
		>
			{isErrorNodeIds ? (
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
						Error loading Nodes
					</Typography>
					<Typography color='error'>{error.message}</Typography>
					<Typography variant='body2' color='text.secondary'>
						Please try again later.
					</Typography>
					<Button variant='outlined' onClick={() => refetch()}>
						Try again
					</Button>
				</Box>
			) : !isFetchingNodeIds ? (
				<Box sx={{ ...styleWrapperCards }}>
					{nodes?.map(node => <AccessOrNodeCard entity={node} key={node} />)}
				</Box>
			) : (
				<Spinner />
			)}
		</Box>
	)
}

export default Home
