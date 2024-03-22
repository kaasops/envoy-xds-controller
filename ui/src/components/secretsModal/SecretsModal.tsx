import { Autocomplete, Box, Modal, TextField, Typography } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { useGetSecretsApi } from "../../api/hooks/useSecrets";
import { ISecretsResponse } from "../../common/types/getSecretsApiTypes";
import { IModalProps } from "../../common/types/modalProps";
import { convertToYaml } from "../../utils/helpers/convertToYaml";
import { styleModalSetting } from "../../utils/helpers/styleModalSettings";
import CodeBlock from "../codeBlock/CodeBlock";

function SecretsModal({ open, onClose }: IModalProps) {
    const { nodeID } = useParams();
    const [loadDataFlag, setLoadDataFlag] = useState(false);
    const [secretName, setSecretName] = useState('');
    const [secretNamesList, setSecretNamesList] = useState<string[]>([]);
    const [yamlData, setYamlData] = useState('');

    const { data, isFetching } = useGetSecretsApi(nodeID as string, secretName, loadDataFlag);

    const getSecretNames = useCallback((data: ISecretsResponse | undefined) => {
        if (data) {
            setSecretNamesList(prevSecretList => [
                ...prevSecretList,
                ...data.secrets.map(secret => secret.name)
            ]);
        }
    }, []);

    useEffect(() => {
        if (open === true) {
            setLoadDataFlag(true)
        }
        if (!open) {
            setSecretName('')
        }
    }, [open])

    useEffect(() => {
        if (data) {
            const yamlString = convertToYaml(data);
            setYamlData(yamlString);
        }

        if (!isFetching && !secretNamesList.length) {
            getSecretNames(data as ISecretsResponse);
        }
    }, [data, isFetching, getSecretNames, secretNamesList.length]);

    const handleChangeSecret = (value: string) => {
        value === null ? setSecretName('') : setSecretName(value)
    }

    return (
        <Modal open={open} onClose={onClose}>
            <Box className='SecretsModalBox' sx={styleModalSetting}>
                <Typography id="modal-modal-title" variant="h6" component="h2">
                    Secrets Modal
                </Typography>
                <Autocomplete disablePortal
                    id="combo-box-demo"
                    options={secretNamesList}
                    sx={{ width: '100%', height: 'auto', paddingY: 2 }}
                    onChange={(_event, value) => handleChangeSecret(value as string)}
                    renderInput={(params) => <TextField {...params} label="Secrets" />}
                />
                {data && (
                    <CodeBlock jsonData={data} yamlData={yamlData} heighCodeBox={91} />
                )}
            </Box>
        </Modal>
    )
}

export default SecretsModal