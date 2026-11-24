import assert from 'node:assert/strict'
import test from 'node:test'
import { groupSnapshotHistory } from '../src/pages/beta/snapshotHistory.ts'

test('repeated publications produce one rollback target per snapshot, newest first', () => {
	const history = [
		{ id: 'a', publishedAt: '2026-01-01T13:32:00Z' },
		{ id: 'b', publishedAt: '2026-01-01T13:25:00Z' },
		{ id: 'a', publishedAt: '2026-01-01T13:18:00Z' },
		{ id: 'b', publishedAt: '2026-01-01T13:10:00Z' },
		{ id: 'c', publishedAt: '2026-01-01T13:00:00Z' }
	]
	assert.deepEqual(groupSnapshotHistory(history), [
		{ ...history[0], publications: 2 },
		{ ...history[1], publications: 2 },
		{ ...history[4], publications: 1 }
	])
})
