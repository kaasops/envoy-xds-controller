import React, { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useGetFilterChain } from '../../api/hooks/useFilterChain'
import { useGetFilterName } from '../../api/hooks/useFilterName'
import { useGetListenerApi } from '../../api/hooks/useListenersApi'
import { useGetRouteConfigurations } from '../../api/hooks/useRouteConfigurations'
import { IDomainLocationsResponse } from '../../common/types/getDomainLocationsApiTypes'
import useSetDomainStore from '../../store/setDomainStore'
import ModalEnvoyConfig from '../modalEnvoyConfig/ModalEnvoyConfig'
import { styleBoxCard, styleListItemButton } from './style'
import Box from '@mui/material/Box'
import Divider from '@mui/material/Divider'
import IconButton from '@mui/material/IconButton'
import List from '@mui/material/List'
import ListItemButton from '@mui/material/ListItemButton'
import Typography from '@mui/material/Typography'
import ContentCopyTwoToneIcon from '@mui/icons-material/ContentCopyTwoTone'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'

import copy from 'clipboard-copy'

const LocationCard: React.FC<{ domain: IDomainLocationsResponse }> = ({ domain }) => {
	const { nodeID } = useParams()
	const currentDomain = useSetDomainStore(state => state.domain)

	const [loadListener, setLoadListener] = useState(false)
	const [loadFilterChain, setLoadFilterChain] = useState(false)
	const [loadFilterName, setLoadFilterName] = useState(false)
	const [loadRouteConfig, setLoadRouteConfig] = useState(false)

	const [modalData, setModalData] = useState({ data: {}, isFetching: false })
	const [openModal, setOpenModal] = useState(false)
	const [modalConfigName, setModalConfigName] = useState('')

	const { data: listenerData, isFetching: listenerIsFetching } = useGetListenerApi(
		nodeID as string,
		domain.listener,
		loadListener
	)

	const { data: filterChainData, isFetching: filterChainIsFetching } = useGetFilterChain(
		nodeID as string,
		domain.listener,
		domain.filter_chain,
		loadFilterChain
	)

	const { data: filterNameData, isFetching: filterNameIsFetching } = useGetFilterName(
		nodeID as string,
		domain.listener,
		domain.filter_chain,
		domain.filter,
		loadFilterName
	)

	const { data: routeConfigurationsData, isFetching: routeConfigurationsIsFetching } = useGetRouteConfigurations(
		nodeID as string,
		domain.route_configuration,
		loadRouteConfig
	)

	useEffect(() => {
		if (!listenerIsFetching && listenerData && modalConfigName === 'Listener') {
			setModalData({ data: listenerData, isFetching: listenerIsFetching })
		}
		if (!filterChainIsFetching && filterChainData && modalConfigName === 'Filter Chain') {
			setModalData({ data: filterChainData, isFetching: filterChainIsFetching })
		}
		if (!filterNameIsFetching && filterNameData && modalConfigName === 'Filter') {
			setModalData({ data: filterNameData, isFetching: filterNameIsFetching })
		}
		if (!routeConfigurationsIsFetching && routeConfigurationsData && modalConfigName === 'Route Configuration') {
			setModalData({ data: routeConfigurationsData, isFetching: routeConfigurationsIsFetching })
		}
		if (!openModal) {
			// Сбрасываем данные при закрытии модального окна
			setModalData({ data: {}, isFetching: false })
			return
		}
	}, [
		listenerData,
		listenerIsFetching,
		filterChainData,
		filterChainIsFetching,
		filterNameData,
		filterNameIsFetching,
		routeConfigurationsData,
		routeConfigurationsIsFetching,
		modalConfigName,
		openModal
	])

	const handleClick = async (key: string) => {
		setOpenModal(true)
		const dataTypeApi = getTitlesCard(key)
		setModalConfigName(dataTypeApi)
		setModalData({ ...modalData, isFetching: true })

		switch (dataTypeApi) {
			case 'Listener':
				setLoadListener(true)
				break
			case 'Filter Chain':
				setLoadFilterChain(true)
				break
			case 'Filter':
				setLoadFilterName(true)
				break
			case 'Route Configuration':
				setLoadRouteConfig(true)
				break
			default:
				break
		}
	}

	const handleCloseModal = () => {
		setOpenModal(false)
		setModalData({ data: {}, isFetching: false })
	}

	const getTitlesCard = (title: string) => {
		return title
			.split('_')
			.map(char => char[0].toLocaleUpperCase() + char.slice(1))
			.join(' ')
	}

	const renderLocationItems = Object.entries(domain).map(([key, value], index) => (
		<ListItemButton sx={styleListItemButton} className='CardRow' key={index} onClick={() => handleClick(key)}>
			<Box display='flex' justifyContent='space-between' width='100%'>
				<Box sx={{ width: '25%' }}>
					<Typography variant='body2' fontWeight='bold'>
						{key}
					</Typography>
				</Box>
				<Box sx={{ width: '75%' }}>
					<Typography variant='body2' sx={{ wordWrap: 'break-word' }}>
						{value}
					</Typography>
				</Box>
			</Box>

			<IconButton
				className='CardRowCopy'
				onClick={event => {
					event.stopPropagation()
					copy(value)
				}}
			>
				<ContentCopyTwoToneIcon />
			</IconButton>
		</ListItemButton>
	))

	return (
		<>
			<Card sx={{ ...styleBoxCard }} className='cardBox'>
				<CardContent>
					<Typography sx={{ fontSize: 14, paddingLeft: 1 }} color='text.secondary' gutterBottom>
						{currentDomain}
					</Typography>
					<Divider />
					<List>{renderLocationItems}</List>
				</CardContent>
			</Card>

			<ModalEnvoyConfig
				open={openModal}
				onClose={handleCloseModal}
				modalData={modalData.data}
				configName={modalConfigName}
			/>
		</>
	)
}

export default LocationCard
