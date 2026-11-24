import type { SnapshotInfo } from './api'

type Publication = Pick<SnapshotInfo, 'id' | 'publishedAt'>

export function groupSnapshotHistory(history: readonly Publication[]) {
	const byId = new Map<string, Publication & { publications: number }>()
	// The API returns previous publications newest first, excluding the currently served publication.
	for (const { id, publishedAt } of history) {
		const row = byId.get(id)
		if (row) {
			row.publications++
		} else {
			byId.set(id, { id, publishedAt, publications: 1 })
		}
	}
	return Array.from(byId.values())
}
