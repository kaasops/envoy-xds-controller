import assert from 'node:assert/strict'
import test from 'node:test'
import {
	createRowFilter,
	formatFilters,
	matchesToken,
	operatorsFor,
	parseFilters
} from '../src/pages/beta/console/tableFilters.ts'

const node = { key: 'node', label: 'Node ID' }
const status = { key: 'status', label: 'Sync status', filterType: 'enum', filterValues: ['ACK', 'Awaiting ACK'] }
const streams = { key: 'streams', label: 'xDS streams', filterType: 'number' }
const connected = { key: 'connected', label: 'Connected', filterType: 'date' }
const fields = [node, status, streams, connected]

test('accepts any exact value of a field while requiring other fields, exclusions and bounds', () => {
	const columns = fields.map(field => ({ ...field, value: row => row[field.key] }))
	const rows = [
		{ node: 'a', streams: 3, status: ['ACK', 'Awaiting ACK'] },
		{ node: 'b', streams: 2, status: ['ACK'] },
		{ node: 'c', streams: 1, status: ['Awaiting ACK'] }
	]
	const matching = query => {
		const parsed = parseFilters(query, columns)
		assert.equal(parsed.error, undefined)
		return rows.filter(createRowFilter(parsed.tokens, columns, 'UTC')).map(row => row.node)
	}
	assert.deepEqual(matching('node:a node:b'), ['a', 'b'])
	assert.deepEqual(matching('node:a status:"Awaiting ACK" node:b'), ['a'])
	assert.deepEqual(matching('node:a node:b -node:a -node:c'), ['b'])
	assert.deepEqual(matching('node:a node:b streams>=2 streams<3'), ['b'])
	assert.deepEqual(matching('status:ACK status:"Awaiting ACK"'), ['a', 'b', 'c'])
})
const token = (field, operator, value) => ({ field, operator, value })
const match = (query, field, value, zone = 'UTC') => {
	const parsed = parseFilters(query, fields)
	assert.equal(parsed.error, undefined)
	return matchesToken(parsed.tokens[0], field, value, zone)
}

test('combines typed and negated tokens with the existing free-text search', () => {
	const raw = 'edge -status:"Awaiting ACK" streams>=2'
	const { tokens, text, error } = parseFilters(raw, fields)
	assert.equal(error, undefined)
	assert.equal(text, 'edge')
	assert.deepEqual(tokens, [token('status', '!=', 'Awaiting ACK'), token('streams', '>=', '2')])
	assert.equal(formatFilters(tokens, text), '-status:"Awaiting ACK" streams>=2 edge')
})

test('preserves punctuation, phrases and unknown field names in existing searches', () => {
	const text = 'eds://sample.test  10.0.0.1:443 [::1] unknown:value plain'
	assert.deepEqual(parseFilters(text, fields), { tokens: [], text })
	assert.deepEqual(parseFilters('prefix "message node:example" node:edge', fields), {
		tokens: [token('node', ':', 'edge')],
		text: 'prefix "message node:example"'
	})
})

test('retains quotes, spaces and backslashes through a URL round trip', () => {
	const tokens = [token('node', ':', 'an edge "blue" \\ west'), token('node', '!=', '2001:db8::1')]
	const url = new URL('https://example.test/beta/nodes')
	url.searchParams.set('q', formatFilters(tokens, 'site  http'))
	assert.deepEqual(parseFilters(new URL(url).searchParams.get('q'), fields), { tokens, text: 'site  http' })
})

test('rejects malformed tokens and unsupported operators for a known field', () => {
	for (const expression of [
		'node:',
		'node:""',
		'node:"  "',
		'node:"open',
		'node:"a"b',
		'node>edge',
		'status:other',
		'streams:no',
		'streams>=',
		'-streams>2',
		'connected:2026-02-30',
		'connected:2026-11'
	]) {
		assert.ok(parseFilters(expression, fields).error, expression)
	}
	assert.deepEqual(operatorsFor(node), [':', '!='])
	assert.deepEqual(operatorsFor(status), [':', '!='])
	assert.ok(operatorsFor(connected).includes('>='))
	assert.ok(operatorsFor(streams).includes('<='))
})

test('negative matches exclude a row if any value equals the operand', () => {
	assert.equal(match('-status:ACK', status, ['Awaiting ACK', 'ACK']), false)
	assert.equal(match('-status:ACK', status, ['Awaiting ACK']), true)
	assert.equal(match('status:ack', status, ['ACK']), true)
	assert.equal(match('node:edge', node, 'Edge-1'), false)
})

test('compares numbers numerically and excludes missing counts from ranges', () => {
	assert.equal(match('streams>2', streams, 10), true)
	assert.equal(match('streams>=10', streams, '10'), true)
	assert.equal(match('streams<10', streams, 10), false)
	assert.equal(match('streams<=0', streams, 0), true)
	for (const absent of ['', ' ', undefined, 'No snapshot', NaN]) {
		assert.equal(match('streams<=0', streams, absent), false)
	}
})

test('compares the calendar day in Preferences, including at a DST boundary', () => {
	const instant = '2026-03-28T23:30:00Z'
	assert.equal(match('connected<2026-03-29', connected, instant), true)
	assert.equal(match('connected:2026-03-29', connected, instant, 'Europe/Berlin'), true)
	assert.equal(match('connected>=2026-03-29', connected, Date.parse(instant), 'Europe/Berlin'), true)
	assert.equal(match('connected<2026-03-29', connected, undefined), false)
	assert.equal(match('connected<2026-03-29', connected, 'Unavailable'), false)
	assert.equal(match('connected:1970-01-01', connected, 0), true)
})
