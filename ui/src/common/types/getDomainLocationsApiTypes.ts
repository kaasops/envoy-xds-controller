export interface IGetDomainLocationsResponse {
	locations: IDomainLocationsResponse[]
}

export interface IDomainLocationsResponse {
	filter: string
	filter_chain: string
	listener: string
	route_configuration: string

	[key: string]: string // для того что бы использовать map по ключам
}
