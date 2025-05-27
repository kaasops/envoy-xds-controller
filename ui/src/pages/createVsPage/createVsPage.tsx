import React from 'react'
import { useColors } from '../../utils/hooks/useColors.ts'
import { Box } from '@mui/material'
import { styleBox, styleRootBoxCreateVS } from './style.ts'
import { VirtualServiceForm } from '../../components/virtualServiceForm/virtualServiceForm.tsx'

interface ICreateVsProps {
	title?: string
}

const CreateVsPage: React.FC<ICreateVsProps> = () => {
	const { colors } = useColors()

	return (
		<Box
			className='RootBoxVirtualServices'
			component='section'
			sx={{ ...styleRootBoxCreateVS, backgroundColor: colors.primary[800] }}
		>
			<Box sx={{ ...styleBox }}>
				<VirtualServiceForm />
			</Box>
		</Box>
	)
}

export default CreateVsPage
