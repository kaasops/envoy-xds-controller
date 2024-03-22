import { Autocomplete, Box, Modal, TextField, Typography } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { useGetClustersApi } from "../../api/hooks/useClusters";
import { IClustersResponse } from "../../common/types/getClustersApiTypes";
import { IModalProps } from "../../common/types/modalProps";
import { convertToYaml } from "../../utils/helpers/convertToYaml";
import { styleModalSetting } from "../../utils/helpers/styleModalSettings";
import CodeBlock from "../codeBlock/CodeBlock";

function ClustersModal({ open, onClose }: IModalProps) {
	const { nodeID } = useParams();
	const [loadDataFlag, setLoadDataFlag] = useState(false);
	const [clusterName, setClusterName] = useState('');
	const [clustersNamesList, setClustersNamesList] = useState<string[]>([]);
	const [yamlData, setYamlData] = useState('');

	const { data, isFetching } = useGetClustersApi(nodeID as string, clusterName, loadDataFlag);

	const getClustersNames = useCallback((data: IClustersResponse | undefined) => {
		if (data) {
			setClustersNamesList(prevClustersList => [
				...prevClustersList,
				...data.clusters.map(cluster => cluster.name)
			]);
		}
	}, []);

	useEffect(() => {
		if (open === true) {
			setLoadDataFlag(true)
		}
		if (!open) {
			setClusterName('')
		}
	}, [open])

	useEffect(() => {
		if (data) {
			const yamlString = convertToYaml(data);
			setYamlData(yamlString);
		}

		if (!isFetching && !clustersNamesList.length) {
			getClustersNames(data as IClustersResponse);
		}
	}, [data, isFetching, getClustersNames, clustersNamesList.length]);

	const handleChangeCluster = (value: string) => {
		value === null ? setClusterName('') : setClusterName(value)
	}


	return (
		<Modal open={open} onClose={onClose}>
			<Box className='ClustersModalBox' sx={styleModalSetting}>
				<Typography id="modal-modal-title" variant="h6" component="h2">
					Clusters Modal
				</Typography>
				<Autocomplete disablePortal
					id="combo-box-demo"
					options={clustersNamesList}
					sx={{ width: '100%', height: 'auto', paddingY: 2 }}
					onChange={(_event, value) => handleChangeCluster(value as string)}
					renderInput={(params) => <TextField {...params} label="Clusters" />}
				/>
				{data && (
					<CodeBlock jsonData={data} yamlData={yamlData} heighCodeBox={91} />
				)}
			</Box>
		</Modal>
	)
}

export default ClustersModal