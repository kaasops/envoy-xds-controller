import { IFilterChainResponse } from "../../common/types/getFilterChainApiTypes";
import axiosClient from "../axiosApiClient";


const GetFilterChainService = {
    getFilterChain: async (nodeID: string, listenerName: string, filterChainName: string) => {
        try {
            const { data } = await axiosClient.get<IFilterChainResponse>(`/filters?node_id=${nodeID}&listener_name=${listenerName}&filter_chain_name=${filterChainName}`
            );
            return data
        } catch (error: any) {
            console.error("Error fetching data:", error);
            if (error.response && error.response.status === 500) {
                // Возвращаем пустой объект в случае ошибки 500
                return { filterChain: [] };
            } else {
                // Возвращаем пустой объект или другие данные по умолчанию в случае других ошибок
                return { filterChain: [] };
            }
        }
    }
}

export default GetFilterChainService;