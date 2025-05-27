import React, { useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { useGetVs } from '../../api/grpc/hooks/useVirtualService.ts'
import { useVirtualServiceStore } from '../../store/setVsStore.ts'
import { Box } from '@mui/material'
import { VirtualServiceForm } from '../../components/virtualServiceForm/virtualServiceForm.tsx'
import { useColors } from '../../utils/hooks/useColors.ts'
import { styleBox, styleRootBoxEditVS } from './style.ts'

interface IEditVsPageProps {
	title?: string
}

const EditVsPage: React.FC<IEditVsPageProps> = () => {
	const { colors } = useColors()
	const { uid } = useParams<{ uid: string }>()
	const { data: virtualService, isLoading, error } = useGetVs(uid ?? '')

	const setVirtualServices = useVirtualServiceStore(state => state.setVirtualService)

	useEffect(() => {
		if (virtualService) {
			setVirtualServices(virtualService.uid, virtualService.name)
		}
	}, [setVirtualServices, virtualService])

	if (isLoading) return <div>Loading...</div>
	if (error) return <div>Error loading virtual service</div>

	return (
		<Box
			className='RootBoxEditVirtualServices'
			component='section'
			sx={{ ...styleRootBoxEditVS, backgroundColor: colors.primary[800] }}
		>
			<Box sx={{ ...styleBox }}>
				<VirtualServiceForm virtualServiceInfo={virtualService} isEdit={true} />
			</Box>
		</Box>
	)
}

export default EditVsPage
