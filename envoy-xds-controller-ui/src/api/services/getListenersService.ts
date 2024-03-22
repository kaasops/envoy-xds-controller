import { IListenersResponse } from "../../common/types/getListenerDomainApiTypes";
import axiosClient from "../axiosApiClient";

const GetListenersService = {
    getListeners: async (nodeId: string, listenerName: string) => {
        try {
            const { data } = await axiosClient.get<IListenersResponse>(`/listeners?node_id=${nodeId}&listener_name=${listenerName}`);
            return data
        } catch (error: any) {
            console.error('Error: ', error)
        }

    }
}

export default GetListenersService;