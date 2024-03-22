import { IGetDomainLocationsResponse } from "../../common/types/getDomainLocationsApiTypes";
import axiosClient from "../axiosApiClient"

const GetDomainLocationsService = {
    getDomainLocations: async (nodeId: string, domain: string) => {
        try {
            const { data } = await axiosClient.get<IGetDomainLocationsResponse>(`/domainLocations?node_id=${nodeId}&domain_name=${domain}`);
            return data
        } catch (error: any) {
            console.error("Error fetching data:", error);

            if (error.response && error.response.status === 500) {
                // Возвращаем пустой объект в случае ошибки 500
                return { locations: [] };
            } else {
                // Возвращаем пустой объект или другие данные по умолчанию в случае других ошибок
                return { locations: [] };
            }
        }
    },
}

export default GetDomainLocationsService
