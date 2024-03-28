import ContentCopyTwoToneIcon from '@mui/icons-material/ContentCopyTwoTone'
import {
	Card,
	CardContent,
	Grid,
	IconButton,
	List,
	ListItemButton,
	Typography,
	useMediaQuery,
	useTheme
} from '@mui/material'
import copy from 'clipboard-copy'
import React, { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useGetFilterChain } from '../../api/hooks/useFilterChain'
import { useGetFilterName } from '../../api/hooks/useFilterName'
import { useGetListenerApi } from '../../api/hooks/useListenersApi'
import { useGetRouteConfigurations } from '../../api/hooks/useRouteConfigurations'
import { IDomainLocationsResponse } from '../../common/types/getDomainLocationsApiTypes'
import useSetDomainStore from '../../store/setDomainStore'
import useSideBarState from '../../store/sideBarStore'
import ModalEnvoyConfig from '../modalEnvoyConfig/ModalEnvoyConfig'
import { styleListItemButton } from './style'

const LocationCard: React.FC<{ domain: IDomainLocationsResponse }> = ({ domain }) => {
	const { nodeID } = useParams()
	const currentDomain = useSetDomainStore(state => state.domain)
	const theme = useTheme()
	const isLargeScreen = useMediaQuery(theme.breakpoints.up('lg'))
	const isExtraLargeScreen = useMediaQuery(theme.breakpoints.up('xl'))
	const isOpenSideBar = useSideBarState(state => state.isOpenSideBar)

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

	let width
	if (isExtraLargeScreen) {
		width = isOpenSideBar ? '30.8vw' : '34.7vw'
	} else if (isLargeScreen) {
		width = isOpenSideBar ? '28.5vw' : '34vw'
	} else {
		width = isOpenSideBar ? '100%' : '30.7vw'
	}

	const getTitlesCard = (title: string) => {
		return title
			.split('_')
			.map(char => char[0].toLocaleUpperCase() + char.slice(1))
			.join(' ')
	}

	const locationKeys = Object.keys(domain)

	const renderLocationItems = locationKeys.map((key, index) => (
		<ListItemButton sx={styleListItemButton} className='CardRow' key={index} onClick={() => handleClick(key)}>
			<Grid container display='flex' alignItems='center'>
				<Grid className='CardRowTitle' item xs={12} md={4.5} lg={4}>
					<Typography sx={{ fontSize: 17, paddingLeft: 1 }} fontWeight='bold'>
						{getTitlesCard(key)}
					</Typography>
				</Grid>
				<Grid className='CardRowValue' item xs={12} md={7.5} lg={8}>
					<Typography sx={{ wordWrap: 'break-word' }}>{domain[key]}</Typography>
				</Grid>
			</Grid>
			<IconButton
				className='CardRowCopy'
				onClick={event => {
					event.stopPropagation()
					copy(domain[key])
				}}
			>
				<ContentCopyTwoToneIcon />
			</IconButton>
		</ListItemButton>
	))

	return (
		<>
			<Card
				sx={{
					minHeight: '200px',
					height: '100%',
					minWidth: width,
					maxWidth: width,
					boxShadow: `0px 0px 8px 0px rgba(0,0,0,0.2),
            0px 0px 0px 0px rgba(0,0,0,0.14),
            0px 1px 3px 0px rgba(0,0,0,0.12)`
				}}
			>
				<CardContent
					sx={{
						overflow: 'auto',
						maxHeight: '100%',
						width: '100%'
					}}
				>
					<Typography sx={{ fontSize: 14, paddingLeft: 1 }} color='text.secondary' gutterBottom>
						{currentDomain}
					</Typography>
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
