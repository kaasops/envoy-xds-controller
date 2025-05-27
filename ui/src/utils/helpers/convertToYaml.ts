import yaml from 'js-yaml'

export function convertToYaml(data: any) {
	return yaml.dump(data)
}

export function parseNestedJson(input: any): any {
	if (typeof input === 'string') {
		try {
			const parsed = JSON.parse(input)
			return parseNestedJson(parsed)
		} catch {
			return input
		}
	}

	if (Array.isArray(input)) {
		return input.map(parseNestedJson)
	}

	if (typeof input === 'object' && input !== null) {
		const result: Record<string, any> = {}
		for (const [key, value] of Object.entries(input)) {
			result[key] = parseNestedJson(value)
		}
		return result
	}

	return input
}

export const convertRawToFullYaml = (rawString: string) => {
	const cleaned = parseNestedJson(JSON.parse(rawString))
	return yaml.dump(cleaned)
}
