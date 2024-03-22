import { Autocomplete, Box, Modal, TextField, Typography } from "@mui/material"
import { IModalProps } from "../../common/types/modalProps"
import { styleModalSetting } from "../../utils/helpers/styleModalSettings";
import { useParams } from "react-router-dom";
import { useGetAllListenersApi } from "../../api/hooks/useListenersApi";
import { useCallback, useEffect, useState } from "react";
import CodeBlock from "../codeBlock/CodeBlock";
import { convertToYaml } from "../../utils/helpers/convertToYaml";
import { IListenersResponse } from "../../common/types/getListenerDomainApiTypes";


function ListenersModal({ open, onClose }: IModalProps) {
    const { nodeID } = useParams();
    const [loadDataFlag, setLoadDataFlag] = useState(false);
    const [listenerName, setListenerName] = useState('');
    const [listenersList, setListenersList] = useState<string[]>([]);
    const [yamlData, setYamlData] = useState('');
    const { data, isFetching } = useGetAllListenersApi(nodeID as string, listenerName, loadDataFlag)

    const getListenersNames = useCallback((data: IListenersResponse | undefined) => {
        if (data) {
            setListenersList(prevListenersList => [
                ...prevListenersList,
                ...data.listeners.map(listener => listener.name)
            ]);
        }
    }, []);

    useEffect(() => {
        if (open === true) {
            setLoadDataFlag(true)
        }
        if (!open) {
            setListenerName('')
        }
    }, [open])

    useEffect(() => {
        if (data) {
            const yamlString = convertToYaml(data);
            setYamlData(yamlString);
        }

        if (!isFetching && !listenersList.length) {
            getListenersNames(data as IListenersResponse);
        }
    }, [data, isFetching, getListenersNames, listenersList.length]);

    const handleChangeListener = (value: string) => {
        value === null ? setListenerName('') : setListenerName(value)
    }

    return (
        <Modal open={open} onClose={onClose}>
            <Box className='ListenersModalBox' sx={styleModalSetting}>
                <Typography id="modal-modal-title" variant="h6" component="h2">
                    Listeners Modal
                </Typography>
                <Autocomplete disablePortal
                    id="combo-box-demo"
                    options={listenersList}
                    sx={{ width: '100%', height: 'auto', paddingY: 2 }}
                    onChange={(_event, value) => handleChangeListener(value as string)}
                    renderInput={(params) => <TextField {...params} label="Listeners" />}
                />
                {data && (
                    <CodeBlock jsonData={data} yamlData={yamlData} heighCodeBox={91}/>
                )}
            </Box>
        </Modal>
    )
}

export default ListenersModal;
