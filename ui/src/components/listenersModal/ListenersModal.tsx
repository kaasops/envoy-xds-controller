import { Autocomplete, Box, Button, Modal, TextField, Typography } from '@mui/material'
import { IModalProps } from '../../common/types/modalProps'
import { styleModalSetting } from '../../utils/helpers/styleModalSettings'
import { useParams } from 'react-router-dom'
import { useGetAllListenersApi } from '../../api/hooks/useListenersApi'
import { useCallback, useEffect, useState } from 'react'
import CodeBlock from '../codeBlock/CodeBlock'
import { convertToYaml } from '../../utils/helpers/convertToYaml'
import { IListenersResponse } from '../../common/types/getListenerDomainApiTypes'
import FullscreenIcon from '@mui/icons-material/Fullscreen'
import FullscreenExitIcon from '@mui/icons-material/FullscreenExit'

function ListenersModal({ open, onClose }: IModalProps) {
	const { nodeID } = useParams()
	const [loadDataFlag, setLoadDataFlag] = useState(false)
	const [listenerName, setListenerName] = useState('')
	const [listenersList, setListenersList] = useState<string[]>([])
	const [yamlData, setYamlData] = useState('')
	const [isFullscreen, setIsFullscreen] = useState(false)
	const fullscreenStyles = {
		width: '95%',
		height: '95%'
	}

	const { data, isFetching } = useGetAllListenersApi(nodeID as string, listenerName, loadDataFlag)

	const getListenersNames = useCallback((data: IListenersResponse | undefined) => {
		if (data) {
			setListenersList(prevListenersList => [
				...prevListenersList,
				...data.listeners.map(listener => listener.name)
			])
		}
	}, [])

	useEffect(() => {
		if (open) {
			setLoadDataFlag(true)
		}
		if (!open) {
			setListenerName('')
			setIsFullscreen(false)
		}
	}, [open])

	useEffect(() => {
		if (data) {
			const yamlString = convertToYaml(data)
			setYamlData(yamlString)
		}

		if (!isFetching && !listenersList.length) {
			getListenersNames(data as IListenersResponse)
		}
	}, [data, isFetching, getListenersNames, listenersList.length])

	const handleChangeListener = (value: string | null): void => {
		setListenerName(value ?? '')
	}

	return (
		<Modal open={open} onClose={onClose}>
			<Box className='ListenersModalBox' sx={{ ...styleModalSetting, ...(isFullscreen ? fullscreenStyles : {}) }}>
				<Box gap={2} sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
					<Box display='flex' justifyContent='space-between' alignItems='flex-start'>
						<Typography variant='h6' component='h2'>
							Listeners Modal
						</Typography>
						<Button onClick={() => setIsFullscreen(!isFullscreen)}>
							{isFullscreen ? <FullscreenExitIcon /> : <FullscreenIcon />}
						</Button>
					</Box>
					<Autocomplete
						disablePortal
						id='combo-box-demo'
						options={listenersList}
						sx={{ width: '100%', height: 'auto' }}
						onChange={(_event, value) => handleChangeListener(value)}
						renderInput={params => <TextField {...params} label='Listeners' />}
					/>
					{data && <CodeBlock jsonData={data} yamlData={yamlData} heightCodeBox={100} />}
				</Box>
			</Box>
		</Modal>
	)
}

export default ListenersModal
