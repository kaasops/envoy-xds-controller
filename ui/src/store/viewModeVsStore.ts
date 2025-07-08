import { create } from 'zustand'

export type ViewMode = 'read' | 'edit'

interface IViewModeState {
	viewMode: ViewMode
	setViewMode: (viewMode: ViewMode) => void
}

export const useViewModeStore = create<IViewModeState>(set => ({
	viewMode: 'read',
	setViewMode: mode => set({ viewMode: mode })
}))
