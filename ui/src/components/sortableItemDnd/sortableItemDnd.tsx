import { useSortable } from '@dnd-kit/sortable'
import { useColors } from '../../utils/hooks/useColors.ts'
import { ListItem } from '@mui/material'
import { CSS } from '@dnd-kit/utilities'

export const SortableItemDnd = ({ uid, name }: { uid: string; name: string }) => {
	const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id: uid })
	const { colors, theme } = useColors()

	return (
		<ListItem
			ref={setNodeRef}
			{...attributes}
			{...listeners}
			sx={{
				padding: '8px',
				marginBottom: '4px',
				backgroundColor: theme.palette.mode === 'light' ? colors.gray[300] : colors.primary[200],
				borderRadius: '4px',
				cursor: 'grab',
				transform: CSS.Transform.toString(transform),
				transition
			}}
		>
			{name}
		</ListItem>
	)
}
