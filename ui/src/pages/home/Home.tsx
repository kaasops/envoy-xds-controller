import { Box } from '@mui/material'
import { useNodeIDs } from '../../api/hooks/useNodeIDsApi'
import NodeCard from '../../components/nodeCard/nodeCard'
import Spinner from '../../components/spinner/Spinner'
import useColors from '../../utils/hooks/useColors'
import { styleRootBox, styleWrapperCards } from './style'

const Home = () => {
	const { colors } = useColors()

	const { data: nodes, isFetching } = useNodeIDs()

	const renderCards = nodes?.map(node => <NodeCard node={node} key={node} />)

	return (
		<Box component='section' sx={{ ...styleRootBox, backgroundColor: colors.primary[800] }}>
			{!isFetching ? <Box sx={{ ...styleWrapperCards }}>{renderCards}</Box> : <Spinner />}
		</Box>
	)
}
export default Home
