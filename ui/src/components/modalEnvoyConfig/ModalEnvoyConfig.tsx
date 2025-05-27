import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Modal from '@mui/material/Modal'
import Typography from '@mui/material/Typography'
import { useEffect, useState } from 'react'
import { convertToYaml } from '../../utils/helpers/convertToYaml'
import CodeBlock from '../codeBlock/CodeBlock'
import { styleModalConfigs } from './style'
import FullscreenIcon from '@mui/icons-material/Fullscreen'
import FullscreenExitIcon from '@mui/icons-material/FullscreenExit'

interface IModalEnvoyConfigProps {
	configName: string
	open: boolean
	onClose: () => void
	modalData: any
}

function ModalEnvoyConfig({ configName, onClose, open, modalData }: IModalEnvoyConfigProps) {
	const [yamlData, setYamlData] = useState('')
	const [isFullscreen, setIsFullscreen] = useState(false)
	const fullscreenStyles = {
		width: '95%',
		height: '95%'
	}

	useEffect(() => {
		if (modalData) {
			const yamlString = convertToYaml(modalData)
			setYamlData(yamlString)
		}
	}, [modalData, open])

	useEffect(() => {
		if (!open) {
			setIsFullscreen(false)
		}
	}, [open])

	return (
		<Modal open={open} onClose={onClose}>
			<Box className='ConfigModalBox' sx={{ ...styleModalConfigs, ...(isFullscreen ? fullscreenStyles : {}) }}>
				<Box display='flex' justifyContent='space-between' alignItems='flex-start'>
					<Typography variant='h6' component='h2' paddingBottom={2}>
						Config: {configName}
					</Typography>
					<Button onClick={() => setIsFullscreen(!isFullscreen)}>
						{isFullscreen ? <FullscreenExitIcon /> : <FullscreenIcon />}
					</Button>
				</Box>

				{modalData && <CodeBlock jsonData={modalData} yamlData={yamlData} heightCodeBox={100} />}
			</Box>
		</Modal>
	)
}

export default ModalEnvoyConfig
