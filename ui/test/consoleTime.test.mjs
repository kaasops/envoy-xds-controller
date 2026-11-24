import assert from 'node:assert/strict'
import test from 'node:test'
import {
	createTimeFormatter,
	isoTime,
	isValidTimeZone,
	readTimeZone,
	storeTimeZone
} from '../src/pages/beta/time/format.ts'

const instant = '2026-10-25T00:30:12.123456789Z'

test('UTC uses a 24-hour clock and preserves the full timestamp for inspection', () => {
	const format = createTimeFormatter('UTC')
	assert.equal(format(instant), '2026-10-25 00:30:12 UTC')
	assert.equal(isoTime(instant), instant)
	assert.equal(isoTime('2026-10-25T03:30:12.123456+03:00'), '2026-10-25T03:30:12.123456+03:00')
	assert.equal(format(0), '1970-01-01 00:00:00 UTC')
	assert.equal(isoTime(0), '1970-01-01T00:00:00.000Z')
	assert.equal(format(null), '')
	assert.equal(format('invalid'), '')
})

test('named zones use the offset at the event, including DST and fractional offsets', () => {
	const format = createTimeFormatter('Europe/Berlin')
	assert.equal(format(instant), '2026-10-25 02:30:12 UTC+02:00')
	assert.equal(format('2026-10-25T01:30:12Z'), '2026-10-25 02:30:12 UTC+01:00')
	assert.equal(createTimeFormatter('Asia/Kathmandu')(instant), '2026-10-25 06:15:12 UTC+05:45')
})

test('Browser local uses the system time zone', t => {
	const before = process.env.TZ
	t.after(() => {
		if (before === undefined) delete process.env.TZ
		else process.env.TZ = before
	})
	process.env.TZ = 'America/New_York'
	assert.equal(createTimeFormatter('local')(instant), '2026-10-24 20:30:12 UTC-04:00')
})

test('preferences round-trip and fall back to Browser local if storage is inaccessible or corrupt', t => {
	const original = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
	t.after(() => {
		if (original) Object.defineProperty(globalThis, 'localStorage', original)
		else delete globalThis.localStorage
	})
	const data = new Map()
	Object.defineProperty(globalThis, 'localStorage', {
		configurable: true,
		value: { getItem: key => data.get(key) ?? null, setItem: (key, value) => data.set(key, value) }
	})
	assert.equal(readTimeZone(), 'local')
	storeTimeZone('Europe/Berlin')
	assert.equal(readTimeZone(), 'Europe/Berlin')
	storeTimeZone('UTC')
	assert.equal(readTimeZone(), 'UTC')
	const key = [...data.keys()][0]
	data.set(key, 'Invalid/Zone')
	assert.equal(readTimeZone(), 'local')
	assert.equal(isValidTimeZone('+03:00'), false)
	Object.defineProperty(globalThis, 'localStorage', {
		configurable: true,
		get() {
			throw new Error('Blocked')
		}
	})
	assert.equal(readTimeZone(), 'local')
	assert.doesNotThrow(() => storeTimeZone('UTC'))
})
