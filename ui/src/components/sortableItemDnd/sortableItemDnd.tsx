import { useSortable } from '@dnd-kit/sortable'
import { useColors } from '../../utils/hooks/useColors.ts'
import { ListItem } from '@mui/material'
import { CSS } from '@dnd-kit/utilities'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'

export const SortableItemDnd = ({
	uid,
	name,
	description
}: {
	uid: string
	name: string
	description: string | undefined
}) => {
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
			<Box sx={{ width: '40%' }}>
				<Typography>{name}</Typography>
			</Box>
			<Box sx={{ width: '65%' }}>
				<Typography variant='body2' sx={{ wordWrap: 'break-word' }} color='textDisabled'>
					{description}
				</Typography>
			</Box>
		</ListItem>
	)
}
