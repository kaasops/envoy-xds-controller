import { Autocomplete, Box, Modal, TextField, Typography } from "@mui/material"
import { IModalProps } from "../../common/types/modalProps"
import { styleModalSetting } from "../../utils/helpers/styleModalSettings"
import { useParams } from "react-router-dom"
import { useCallback, useEffect, useState } from "react";
import { useGetRouteConfigurations } from "../../api/hooks/useRouteConfigurations";
import { IRouteConfigurationResponse } from "../../common/types/getRouteConfigurationApiTypes";
import { convertToYaml } from "../../utils/helpers/convertToYaml";
import CodeBlock from "../codeBlock/CodeBlock";

function RouteConfigurationsModal({ open, onClose }: IModalProps) {
    const { nodeID } = useParams();
    const [loadDataFlag, setLoadDataFlag] = useState(false);
    const [routeConfigurationName, setRouteConfigurationName] = useState('');
    const [routeConfigurationsList, setRouteConfigurationsList] = useState<string[]>([]);
    const [yamlData, setYamlData] = useState('');

    const { data, isFetching } = useGetRouteConfigurations(nodeID as string, routeConfigurationName, loadDataFlag);

    const getRouteConfigurationsNames = useCallback((data: IRouteConfigurationResponse | undefined) => {
        if (data) {
            setRouteConfigurationsList(prevRouteConfigurationsList => [
                ...prevRouteConfigurationsList,
                ...data.routeConfigurations.map(routeConfiguration => routeConfiguration.name)
            ]);
        }
    }, []);

    useEffect(() => {
        if (open === true) {
            setLoadDataFlag(true)
        }
        if (!open) {
            setRouteConfigurationName('')
        }
    }, [open])

    useEffect(() => {
        if (data) {
            const yamlString = convertToYaml(data);
            setYamlData(yamlString);
        }

        if (!isFetching && !routeConfigurationsList.length) {
            getRouteConfigurationsNames(data as IRouteConfigurationResponse);
        }
    }, [data, isFetching, getRouteConfigurationsNames, routeConfigurationsList.length]);

    const handleChangeRouteConfiguration = (value: string) => {
        value === null ? setRouteConfigurationName('') : setRouteConfigurationName(value)
    }

    return (
        <Modal open={open} onClose={onClose}>
            <Box className='RouteConfigurationBox' sx={styleModalSetting}>
                <Typography id="modal-modal-title" variant="h6" component="h2">
                    Route Configurations Modal
                </Typography>
                <Autocomplete disablePortal
                    id="combo-box-demo"
                    options={routeConfigurationsList}
                    sx={{ width: '100%', height: 'auto', paddingY: 2 }}
                    onChange={(_event, value) => handleChangeRouteConfiguration(value as string)}
                    renderInput={(params) => <TextField {...params} label="RouteConfigurations" />}
                />
                {data && (
                    <CodeBlock jsonData={data} yamlData={yamlData} heighCodeBox={91} />
                )}
            </Box>
        </Modal>
    )
}

export default RouteConfigurationsModal