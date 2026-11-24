export function formatDate(dateString: string): string {
	const date = new Date(dateString)

	const day = String(date.getUTCDate()).padStart(2, '0')
	const month = String(date.getUTCMonth() + 1).padStart(2, '0')
	const year = date.getUTCFullYear()

	return `${day}-${month}-${year}`
}

export function isExpired(dateString: string): boolean {
	const inputDate = new Date(dateString)
	const currentDate = new Date()
	return inputDate < currentDate
}

export function formatUtcTimestamp(value?: string) {
	if (!value) return ''
	const date = new Date(value)
	return Number.isNaN(date.getTime()) ? '' : date.toISOString().slice(0, 19).replace('T', ' ') + ' UTC'
}
