import { useLocation, useParams } from 'react-router-dom'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { useEffect } from 'react'

export const useVirtualServiceFormMeta = () => {
	const { groupId } = useParams<{ groupId?: string }>()
	const isCreate = useLocation().pathname.split('/').pop() === 'createVs'

	const setViewMode = useViewModeStore(state => state.setViewMode)

	useEffect(() => {
		if (isCreate) {
			setViewMode('edit')
		}
	}, [isCreate, setViewMode])

	return { groupId, isCreate }
}
