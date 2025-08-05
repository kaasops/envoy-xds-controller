export const getDefaultVirtualServiceValues = (isCreate: boolean, groupId?: string) => ({
	name: '',
	nodeIds: [],
	virtualHostDomains: [],
	accessGroup: isCreate ? groupId : '',
	additionalHttpFilterUids: [],
	additionalRouteUids: [],
	accessLogConfigUids: [],
	useRemoteAddress: undefined,
	templateOptions: [],
	viewTemplateMode: false,
	virtualHostDomainsMode: false,
	additionalHttpFilterMode: false,
	additionalRouteMode: false,
	additionalAccessLogConfigMode: false,
	extraFields: {}
})
