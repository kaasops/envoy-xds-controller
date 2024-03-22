import axiosClient from "../axiosApiClient";

const NodeIDsApiService = {
    getNodeIDs: async () => {
        return await axiosClient.get<string[]>('/nodeIDs')
    }
}

export default NodeIDsApiService;