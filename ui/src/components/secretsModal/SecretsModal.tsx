import { Autocomplete, Box, Button, Modal, TextField, Typography } from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useGetSecretsApi } from '../../api/hooks/useSecrets'
import { ISecretsResponse } from '../../common/types/getSecretsApiTypes'
import { IModalProps } from '../../common/types/modalProps'
import { convertToYaml } from '../../utils/helpers/convertToYaml'
import { styleModalSetting } from '../../utils/helpers/styleModalSettings'
import FullscreenExitIcon from '@mui/icons-material/FullscreenExit'
import FullscreenIcon from '@mui/icons-material/Fullscreen'
import CodeBlockExtends from '../CodeBlockExtends/CodeBlockExtends.tsx'

function SecretsModal({ open, onClose }: IModalProps) {
	const { nodeID } = useParams()
	const [loadDataFlag, setLoadDataFlag] = useState(false)
	const [secretName, setSecretName] = useState('')
	const [secretNamesList, setSecretNamesList] = useState<string[]>([])
	const [yamlData, setYamlData] = useState('')

	const [isFullscreen, setIsFullscreen] = useState(false)
	const fullscreenStyles = {
		width: '95%',
		height: '95%'
	}

	const { data, isFetching } = useGetSecretsApi(nodeID as string, secretName, loadDataFlag)

	const getSecretNames = useCallback((data: ISecretsResponse | undefined) => {
		if (data) {
			setSecretNamesList(prevSecretList => [...prevSecretList, ...data.secrets.map(secret => secret.name)])
		}
	}, [])

	useEffect(() => {
		if (open) {
			setLoadDataFlag(true)
		}
		if (!open) {
			setSecretName('')
			setIsFullscreen(false)
		}
	}, [open])

	useEffect(() => {
		if (data) {
			const yamlString = convertToYaml(data)
			setYamlData(yamlString)
		}

		if (!isFetching && !secretNamesList.length) {
			getSecretNames(data as ISecretsResponse)
		}
	}, [data, isFetching, getSecretNames, secretNamesList.length])

	const handleChangeSecret = (value: string | null): void => {
		setSecretName(value ?? '')
	}

	return (
		<Modal open={open} onClose={onClose}>
			<Box className='SecretsModalBox' sx={{ ...styleModalSetting, ...(isFullscreen ? fullscreenStyles : {}) }}>
				<Box gap={2} sx={{ display: 'flex', flexDirection: 'column', height: '100%' }} overflow='auto'>
					<Box display='flex' justifyContent='space-between' alignItems='flex-start'>
						<Typography variant='h6' component='h2'>
							Secrets Modal
						</Typography>
						<Button onClick={() => setIsFullscreen(!isFullscreen)}>
							{isFullscreen ? <FullscreenExitIcon /> : <FullscreenIcon />}
						</Button>
					</Box>
					<Autocomplete
						disablePortal
						id='combo-box-demo'
						options={secretNamesList}
						sx={{ width: '100%', height: 'auto' }}
						onChange={(_event, value) => handleChangeSecret(value)}
						renderInput={params => <TextField {...params} label='Secrets' />}
					/>
					{data && <CodeBlockExtends jsonData={data} yamlData={yamlData} heightCodeBox={100} />}
				</Box>
			</Box>
		</Modal>
	)
}

export default SecretsModal
