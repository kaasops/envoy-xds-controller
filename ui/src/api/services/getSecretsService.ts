import { ISecretsResponse } from "../../common/types/getSecretsApiTypes";
import axiosClient from "../axiosApiClient";

const GetSecretsService = {
    getSecrets: async (nodeId: string, secretName: string) => {
        try {
            const { data } = await axiosClient.get<ISecretsResponse>(`/secrets?node_id=${nodeId}&secret_name=${secretName}`);
            return data
        } catch (error: any) {
            console.error('Error: ', error)
        }
    }
}

export default GetSecretsService;