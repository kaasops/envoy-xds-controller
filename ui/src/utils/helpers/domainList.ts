export const filterData = (query: string, data: string[]) => {
	if (!query) {
		return data;
	} else {
		return data.filter((domain) => domain.includes(query.toLowerCase()));
	}
};