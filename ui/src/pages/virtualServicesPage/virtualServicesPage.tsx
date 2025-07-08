import VirtualServicesTable from '../../components/virtualServicesTable/virtualServicesTable.tsx'
import Box from '@mui/material/Box'
import { styleBox, styleRootBoxVirtualService } from './style.ts'
import { useColors } from '../../utils/hooks/useColors.ts'
import { useParams } from 'react-router-dom'

function VirtualServicesPage() {
	const { colors } = useColors()
	const { groupId } = useParams()

	return (
		<Box
			className='RootBoxVirtualServices'
			component='section'
			sx={{ ...styleRootBoxVirtualService, backgroundColor: colors.primary[800] }}
		>
			<Box sx={{ ...styleBox }}>
				<VirtualServicesTable groupId={groupId as string} />
			</Box>
		</Box>
	)
}

export default VirtualServicesPage
