import { Autocomplete, Box, Button, Modal, TextField, Typography } from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useGetClustersApi } from '../../api/hooks/useClusters'
import { IClustersResponse } from '../../common/types/getClustersApiTypes'
import { IModalProps } from '../../common/types/modalProps'
import { convertToYaml } from '../../utils/helpers/convertToYaml'
import { styleModalSetting } from '../../utils/helpers/styleModalSettings'
import CodeBlock from '../codeBlock/CodeBlock'
import FullscreenExitIcon from '@mui/icons-material/FullscreenExit'
import FullscreenIcon from '@mui/icons-material/Fullscreen'

function ClustersModal({ open, onClose }: IModalProps) {
	const { nodeID } = useParams()
	const [loadDataFlag, setLoadDataFlag] = useState(false)
	const [clusterName, setClusterName] = useState('')
	const [clustersNamesList, setClustersNamesList] = useState<string[]>([])
	const [yamlData, setYamlData] = useState('')

	const [isFullscreen, setIsFullscreen] = useState(false)
	const fullscreenStyles = {
		width: '95%',
		height: '95%'
	}

	const { data, isFetching } = useGetClustersApi(nodeID as string, clusterName, loadDataFlag)

	const getClustersNames = useCallback((data: IClustersResponse | undefined) => {
		if (data) {
			setClustersNamesList(prevClustersList => [
				...prevClustersList,
				...data.clusters.map(cluster => cluster.name)
			])
		}
	}, [])

	useEffect(() => {
		if (open) {
			setLoadDataFlag(true)
		}
		if (!open) {
			setClusterName('')
			setIsFullscreen(false)
		}
	}, [open])

	useEffect(() => {
		if (data) {
			const yamlString = convertToYaml(data)
			setYamlData(yamlString)
		}

		if (!isFetching && !clustersNamesList.length) {
			getClustersNames(data as IClustersResponse)
		}
	}, [data, isFetching, getClustersNames, clustersNamesList.length])

	const handleChangeCluster = (value: string | null): void => {
		setClusterName(value ?? '')
	}

	return (
		<Modal open={open} onClose={onClose}>
			<Box className='ClustersModalBox' sx={{ ...styleModalSetting, ...(isFullscreen ? fullscreenStyles : {}) }}>
				<Box gap={2} sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
					<Box display='flex' justifyContent='space-between' alignItems='flex-start'>
						<Typography variant='h6' component='h2'>
							Clusters Modal
						</Typography>
						<Button onClick={() => setIsFullscreen(!isFullscreen)}>
							{isFullscreen ? <FullscreenExitIcon /> : <FullscreenIcon />}
						</Button>
					</Box>
					<Autocomplete
						disablePortal
						id='combo-box-demo'
						options={clustersNamesList}
						sx={{ width: '100%', height: 'auto' }}
						onChange={(_event, value) => handleChangeCluster(value)}
						renderInput={params => <TextField {...params} label='Clusters' />}
					/>
					{data && <CodeBlock jsonData={data} yamlData={yamlData} heightCodeBox={100} />}
				</Box>
			</Box>
		</Modal>
	)
}

export default ClustersModal
