import { Box } from '@mui/material'
import { useColors } from '../../utils/hooks'
import { styleSettingsNodeSection, styleWrapperNodeSettingsCards } from './style'
import NodeSettingsCard from '../nodeSettingsCard/NodeSettingsCard'
import React, { useState } from 'react'
import { IModalProps } from '../../common/types/modalProps'
import ListenersModal from '../listenersModal/ListenersModal'
import ClustersModal from '../clustersModal/ClustersModal'
import SecretsModal from '../secretsModal/SecretsModal'
import RouteConfigurationsModal from '../routeConfigurationsModal/RouteConfigurationsModal'
import { nodeSettingsItems } from '../../utils/helpers'

function SettingsNodeSection() {
	const { colors } = useColors()
	const [openModalId, setOpenModalId] = useState<number | null>(null)

	const handleOpenModal = (id: number) => {
		setOpenModalId(id)
	}

	const handleCloseModal = () => {
		setOpenModalId(null)
	}

	const getModalComponent = (id: number): React.FC<IModalProps> | null => {
		switch (id) {
			case 1:
				return ListenersModal
			case 2:
				return ClustersModal
			case 3:
				return RouteConfigurationsModal
			case 4:
				return SecretsModal
			default:
				return null
		}
	}

	return (
		<Box className='NodeSettingsSection' sx={{ ...styleSettingsNodeSection, backgroundColor: colors.primary[800] }}>
			<Box className='WrapperNodeSettings' sx={styleWrapperNodeSettingsCards}>
				{nodeSettingsItems.map(item => (
					<NodeSettingsCard key={item.id} title={item.name} handleClick={() => handleOpenModal(item.id)} />
				))}
				{nodeSettingsItems.map(item => {
					const ModalComponent = getModalComponent(item.id)
					return (
						ModalComponent && (
							<ModalComponent key={item.id} open={openModalId === item.id} onClose={handleCloseModal} />
						)
					)
				})}
			</Box>
		</Box>
	)
}

export default SettingsNodeSection
