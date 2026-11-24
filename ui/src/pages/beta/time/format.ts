export type TimeValue = string | number | Date | null | undefined

const storageKey = 'beta.timeZone'

export function isValidTimeZone(zone: string) {
	if (!/^[A-Za-z]/.test(zone)) return false
	try {
		new Intl.DateTimeFormat('en', { timeZone: zone })
		return true
	} catch {
		return false
	}
}

export function readTimeZone(): string {
	try {
		const zone = localStorage.getItem(storageKey)
		return zone && zone !== 'local' && isValidTimeZone(zone) ? zone : 'local'
	} catch {
		return 'local'
	}
}

export function storeTimeZone(zone: string) {
	try {
		localStorage.setItem(storageKey, zone)
	} catch {
		// The selected zone still applies to the current console session.
	}
}

function asDate(value: TimeValue) {
	if (value == null || value === '') return undefined
	const date = value instanceof Date ? value : new Date(value)
	return Number.isNaN(date.getTime()) ? undefined : date
}

export function isoTime(value: TimeValue): string {
	const date = asDate(value)
	if (!date) return ''
	// Keep the precision of protobuf and Go timestamps beyond JavaScript's milliseconds.
	if (typeof value === 'string' && /^\d{4}-\d{2}-\d{2}T.+(?:Z|[+-]\d{2}:\d{2})$/i.test(value)) return value
	return date.toISOString()
}

export function createTimeFormatter(zone: string) {
	const formatter = new Intl.DateTimeFormat('en-GB', {
		timeZone: zone === 'local' ? undefined : zone,
		hourCycle: 'h23',
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit',
		timeZoneName: 'longOffset'
	})
	return (value: TimeValue): string => {
		const date = asDate(value)
		if (!date) return ''
		const parts = Object.fromEntries(formatter.formatToParts(date).map(part => [part.type, part.value]))
		const offset = parts.timeZoneName.replace(/^GMT/, 'UTC').replace(/[+-]00:00$/, '')
		return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}:${parts.second} ${offset}`
	}
}
