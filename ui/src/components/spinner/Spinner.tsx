import { Box } from '@mui/material'
import CircularProgress from '@mui/material/CircularProgress'
import { useColors } from '../../utils/hooks/useColors'

function Spinner() {
	const { colors } = useColors()

	return (
		<Box
			sx={{
				display: 'flex',
				justifyContent: 'center',
				alignItems: 'center',
				width: '100vw',
				height: '100vh',
				backgroundColor: colors.primary.DEFAULT
			}}
		>
			<CircularProgress size={100} />
		</Box>
	)
}

export default Spinner
