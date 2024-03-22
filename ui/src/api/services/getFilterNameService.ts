import { IFilterNameResponse } from "../../common/types/getFilterNameApiTypes";
import axiosClient from "../axiosApiClient";

const GetFilterNameService = {
    getFilterName: async (nodeID: string, listenerName: string, filterChainName: string, filterName: string) => {
        try {
            const { data } = await axiosClient.get<IFilterNameResponse>(`/filters?node_id=${nodeID}&listener_name=${listenerName}&filter_chain_name=${filterChainName}&filter_name=${filterName}`);
            return data;

        } catch (error: any) {
            console.error("Error fetching data:", error);
            if (error.response && error.response.status === 500) {
                return { filterName: [] };
            } else {
                return { filterName: [] }
            }
        }

    }
}

export default GetFilterNameService;