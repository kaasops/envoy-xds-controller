import create from 'zustand'

interface TabStore {
	tabIndex: number
	setTabIndex: (index: number) => void
}

export const useTabStore = create<TabStore>(set => ({
	tabIndex: 0,
	setTabIndex: index => set({ tabIndex: index })
}))
