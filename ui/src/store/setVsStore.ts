import { create } from 'zustand'

interface IVirtualServiceState {
	virtualServiceMap: Map<string, string>
	setVirtualService: (uid: string, name: string) => void
}

export const useVirtualServiceStore = create<IVirtualServiceState>(set => ({
	virtualServiceMap: new Map(),
	setVirtualService: (uid, name) =>
		set(state => {
			const newMap = new Map(state.virtualServiceMap)
			newMap.set(uid, name)
			return { virtualServiceMap: newMap }
		})
}))
