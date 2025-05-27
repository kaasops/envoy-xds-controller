import { create } from 'zustand'
import { devtools } from 'zustand/middleware'
import { immer } from 'zustand/middleware/immer'

interface ISideBarStore {
	isOpenSideBar: boolean
	toggleSideBar: (isOpen: boolean) => void
}

const useSideBarState = create<ISideBarStore>()(
	devtools(
		immer(set => ({
			isOpenSideBar: true,
			toggleSideBar(isOpen: boolean) {
				set(state => {
					state.isOpenSideBar = isOpen
				})
			}
		})),
		{ name: 'sideBarStore' }
	)
)

export default useSideBarState
