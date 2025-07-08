import { create } from 'zustand'
import { devtools } from 'zustand/middleware'

type IUseSetDomainStore = {
	domain: string
	setDomainValue: (domain: string) => void
}

const useSetDomainStore = create<IUseSetDomainStore>()(
	devtools(
		set => ({
			domain: '',
			setDomainValue: domain => set({ domain })
		}),
		{ name: 'domainStore' }
	)
)

export default useSetDomainStore
