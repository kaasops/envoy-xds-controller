import { Box, Modal, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import { convertToYaml } from '../../utils/helpers/convertToYaml'
import CodeBlock from '../codeBlock/CodeBlock'
import { modalBox } from './style'

interface IModalEnvoyConfigProps {
	configName: string
	open: boolean
	onClose: () => void
	modalData: any
}

function ModalEnvoyConfig({ configName, onClose, open, modalData }: IModalEnvoyConfigProps) {
	const [yamlData, setYamlData] = useState('')

	useEffect(() => {
		if (modalData) {
			const yamlString = convertToYaml(modalData)
			setYamlData(yamlString)
		}
	}, [modalData, open])

	return (
		<Modal open={open} onClose={onClose}>
			<Box className='ModalBox' sx={modalBox}>
				<Typography variant='h6' component='h2' paddingBottom={2}>
					Config: {configName}
				</Typography>
				{modalData && <CodeBlock jsonData={modalData} yamlData={yamlData} heightCodeBox={98} />}
			</Box>
		</Modal>
	)
}

export default ModalEnvoyConfig
