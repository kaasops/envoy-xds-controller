import { IRouteConfigurationResponse } from "../../common/types/getRouteConfigurationApiTypes";
import axiosClient from "../axiosApiClient";

const GetRouteConfigurationsService = {
    getRouteConfiguration: async (nodeID: string, routeConfigurationsName: string) => {
        try {
            const { data } = await axiosClient.get<IRouteConfigurationResponse>(`/routeConfigurations?node_id=${nodeID}&route_configuration_name=${routeConfigurationsName}`);
            return data;
        } catch (error: any) {
            console.error("Error fetching data:", error);
            if (error.response && error.response.status === 500) {
                return { routeConfigurations: [] };
            } else {
                return { routeConfigurations: [] }
            }
        }
    }
}

export default GetRouteConfigurationsService;