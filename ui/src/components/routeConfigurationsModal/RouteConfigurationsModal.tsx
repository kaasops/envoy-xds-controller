import { Autocomplete, Box, Button, Modal, TextField, Typography } from '@mui/material'
import { IModalProps } from '../../common/types/modalProps'
import { styleModalSetting } from '../../utils/helpers/styleModalSettings'
import { useParams } from 'react-router-dom'
import { useCallback, useEffect, useState } from 'react'
import { useGetRouteConfigurations } from '../../api/hooks/useRouteConfigurations'
import { IRouteConfigurationResponse } from '../../common/types/getRouteConfigurationApiTypes'
import { convertToYaml } from '../../utils/helpers/convertToYaml'
import CodeBlock from '../codeBlock/CodeBlock'
import FullscreenExitIcon from '@mui/icons-material/FullscreenExit'
import FullscreenIcon from '@mui/icons-material/Fullscreen'

function RouteConfigurationsModal({ open, onClose }: IModalProps) {
	const { nodeID } = useParams()
	const [loadDataFlag, setLoadDataFlag] = useState(false)
	const [routeConfigurationName, setRouteConfigurationName] = useState('')
	const [routeConfigurationsList, setRouteConfigurationsList] = useState<string[]>([])
	const [yamlData, setYamlData] = useState('')

	const [isFullscreen, setIsFullscreen] = useState(false)
	const fullscreenStyles = {
		width: '95%',
		height: '95%'
	}

	const { data, isFetching } = useGetRouteConfigurations(nodeID as string, routeConfigurationName, loadDataFlag)

	const getRouteConfigurationsNames = useCallback((data: IRouteConfigurationResponse | undefined) => {
		if (data) {
			setRouteConfigurationsList(prevRouteConfigurationsList => [
				...prevRouteConfigurationsList,
				...data.routeConfigurations.map(routeConfiguration => routeConfiguration.name)
			])
		}
	}, [])

	useEffect(() => {
		if (open) {
			setLoadDataFlag(true)
		}
		if (!open) {
			setRouteConfigurationName('')
			setIsFullscreen(false)
		}
	}, [open])

	useEffect(() => {
		if (data) {
			const yamlString = convertToYaml(data)
			setYamlData(yamlString)
		}

		if (!isFetching && !routeConfigurationsList.length) {
			getRouteConfigurationsNames(data as IRouteConfigurationResponse)
		}
	}, [data, isFetching, getRouteConfigurationsNames, routeConfigurationsList.length])

	const handleChangeRouteConfiguration = (value: string | null): void => {
		setRouteConfigurationName(value ?? '')
	}

	return (
		<Modal open={open} onClose={onClose}>
			<Box
				className='RouteConfigurationBox'
				sx={{ ...styleModalSetting, ...(isFullscreen ? fullscreenStyles : {}) }}
			>
				<Box gap={2} sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
					<Box display='flex' justifyContent='space-between' alignItems='flex-start'>
						<Typography variant='h6' component='h2'>
							Route Configurations Modal
						</Typography>
						<Button onClick={() => setIsFullscreen(!isFullscreen)}>
							{isFullscreen ? <FullscreenExitIcon /> : <FullscreenIcon />}
						</Button>
					</Box>
					<Autocomplete
						disablePortal
						id='combo-box-demo'
						options={routeConfigurationsList}
						sx={{ width: '100%', height: 'auto' }}
						onChange={(_event, value) => handleChangeRouteConfiguration(value)}
						renderInput={params => <TextField {...params} label='RouteConfigurations' />}
					/>
					{data && <CodeBlock jsonData={data} yamlData={yamlData} heightCodeBox={100} />}
				</Box>
			</Box>
		</Modal>
	)
}

export default RouteConfigurationsModal
