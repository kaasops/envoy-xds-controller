export type FilterOperator = ':' | '!=' | '>' | '<' | '>=' | '<='

export interface FilterField {
	key: string
	label: string
	filterType?: 'text' | 'enum' | 'number' | 'date'
	filterValues?: readonly string[]
}

export interface FilterToken {
	field: string
	operator: FilterOperator
	value: string
}

interface ParsedFilters {
	tokens: FilterToken[]
	text: string
	error?: string
}

const equality = [':', '!='] as const
const ordered = [...equality, '>', '<', '>=', '<='] as const

export function operatorsFor(field: FilterField): readonly FilterOperator[] {
	return field.filterType === 'date' || field.filterType === 'number' ? ordered : equality
}

const decimal = /^[+-]?(?:\d+(?:\.\d+)?|\.\d+)$/

function validateValue(field: FilterField, value: string): string | undefined {
	if (!value.trim()) return 'Enter a value for ' + field.label + '.'
	if (field.filterType === 'number' && (!decimal.test(value) || !Number.isFinite(Number(value)))) {
		return 'Enter a number for ' + field.label + '.'
	}
	if (field.filterType === 'date') {
		if (/^\d{4}-\d\d-\d\d$/.test(value)) {
			const date = new Date(value + 'T00:00:00Z')
			if (Number.isFinite(date.getTime()) && date.toISOString().startsWith(value + 'T')) return
		}
		return 'Enter a valid date (YYYY-MM-DD) for ' + field.label + '.'
	}
	if (
		field.filterType === 'enum' &&
		field.filterValues &&
		!field.filterValues.some(v => v.toLowerCase() === value.toLowerCase())
	) {
		return 'Choose a listed value for ' + field.label + '.'
	}
}

export function parseFilters(input: string, fields: readonly FilterField[]): ParsedFilters {
	const tokens: FilterToken[] = []
	const free: string[] = []
	let freeStart = 0
	const invalid = (error: string): ParsedFilters => ({ tokens: [], text: input, error })

	// Quoted values may contain spaces and escaped quotes. Keep the original spaces within free text.
	for (const term of input.matchAll(/(?:[^\s"]|"(?:\\.|[^"\\])*(?:"|\\?$))+/gs)) {
		const parts = /^(-?)([\w.-]+)(!=|[<>]=?|[:=!])(.*)$/s.exec(term[0])
		if (!parts) continue
		const [, negative, key, symbol, raw] = parts
		const field = fields.find(f => f.key === key)
		if (!field) continue
		const operator = negative && symbol === ':' ? '!=' : symbol
		if ((negative && symbol !== ':') || !operatorsFor(field).some(op => op === operator)) {
			return invalid('Choose a supported operator for ' + field.label + '.')
		}
		let value = raw
		if (raw.includes('"')) {
			try {
				if (!raw.startsWith('"')) return invalid('Quote the entire value for ' + field.label + '.')
				value = JSON.parse(raw)
			} catch {
				return invalid('Check the quoted value for ' + field.label + '.')
			}
		}
		const error = validateValue(field, value)
		if (error) return invalid(error)
		const start = term.index!
		free.push(input.slice(freeStart, start).trim())
		freeStart = start + term[0].length
		tokens.push({ field: field.key, operator: operator as FilterOperator, value })
	}
	free.push(input.slice(freeStart).trim())
	return { tokens, text: free.filter(Boolean).join(' ') }
}

export function formatFilters(tokens: readonly FilterToken[], text: string): string {
	const terms = tokens.map(({ field, operator, value }) => {
		const encoded = /[\s"\\]/.test(value) ? JSON.stringify(value) : value
		return operator === '!=' ? '-' + field + ':' + encoded : field + operator + encoded
	})
	return [...terms, text.trim()].filter(Boolean).join(' ')
}

// A table can contain thousands of timestamps. Reuse the calendar for the current preference.
let calendar: { zone: string; formatter: Intl.DateTimeFormat } | undefined

function calendarDay(value: string | number, zone: string) {
	const date = new Date(value)
	if (!Number.isFinite(date.getTime())) return undefined
	if (calendar?.zone !== zone) {
		calendar = {
			zone,
			formatter: new Intl.DateTimeFormat('sv-SE', {
				timeZone: zone === 'local' ? undefined : zone,
				year: 'numeric',
				month: '2-digit',
				day: '2-digit'
			})
		}
	}
	return calendar.formatter.format(date)
}

function comparable(field: FilterField, value: string | number, timeZone: string) {
	if (field.filterType === 'date') return calendarDay(value, timeZone)
	if (field.filterType === 'number') {
		if (typeof value === 'string' && !decimal.test(value)) return undefined
		const number = Number(value)
		return Number.isFinite(number) ? number : undefined
	}
	return String(value).toLowerCase()
}

export function matchesToken(
	token: FilterToken,
	field: FilterField,
	raw: string | number | readonly string[] | undefined,
	timeZone: string
): boolean {
	const target = field.filterType === 'number' ? Number(token.value) : token.value.toLowerCase()
	const matches = (value: string | number) => {
		const item = comparable(field, value, timeZone)
		if (item === undefined) return false
		switch (token.operator) {
			case '>':
				return item > target
			case '>=':
				return item >= target
			case '<':
				return item < target
			case '<=':
				return item <= target
			default:
				return item === target
		}
	}

	const found = typeof raw === 'string' || typeof raw === 'number' ? matches(raw) : raw?.some(matches) ?? false
	return token.operator === '!=' ? !found : found
}
