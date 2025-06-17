import React from 'react'
import { closestCenter, DndContext, DragEndEvent } from '@dnd-kit/core'
import { arrayMove, SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import Box from '@mui/material/Box'
import Tooltip from '@mui/material/Tooltip'
import { styleTooltip } from './style.ts'
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward'
import List from '@mui/material/List'
import { SortableItemDnd } from '../sortableItemDnd/sortableItemDnd.tsx'
import { Control, UseFormSetValue, useWatch } from 'react-hook-form'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import { nameFieldKeys } from './dNdSelectFormVs.tsx'
import { ListHTTPFiltersResponse } from '../../gen/http_filter/v1/http_filter_pb.ts'
import { ListRoutesResponse } from '../../gen/route/v1/route_pb.ts'

interface IDNdElementsBoxProps {
	titleMessage: 'HTTP filter' | 'Route'
	nameField: nameFieldKeys
	control: Control<IVirtualServiceForm>
	setValue: UseFormSetValue<IVirtualServiceForm>
	data: ListHTTPFiltersResponse | ListRoutesResponse | undefined
}

export const DNdElements: React.FC<IDNdElementsBoxProps> = ({ titleMessage, nameField, control, setValue, data }) => {
	const selectedUids = useWatch({ control, name: nameField })

	const onDragEnd = (e: DragEndEvent) => {
		const { active, over } = e
		if (!over || active.id === over.id) return

		const oldIndex = selectedUids.indexOf(active.id.toString())
		const newIndex = selectedUids.indexOf(over.id.toString())

		setValue(nameField, arrayMove(selectedUids, oldIndex, newIndex))
	}

	return (
		<DndContext collisionDetection={closestCenter} onDragEnd={onDragEnd}>
			<SortableContext items={selectedUids} strategy={verticalListSortingStrategy}>
				<Box sx={{ display: 'flex', alignItems: 'center' }}>
					<Tooltip
						title={`Arrange the ${titleMessage} from top to bottom..`}
						placement='bottom-start'
						enterDelay={500}
						slotProps={{ ...styleTooltip }}
					>
						<ArrowDownwardIcon sx={{ fontSize: 19, color: 'gray' }} />
					</Tooltip>
					<List sx={{ padding: 1, borderRadius: '4px', width: '100%' }}>
						{selectedUids.map(uid => {
							const item = (data?.items || []).find(el => el.uid === uid)
							return item ? (
								<SortableItemDnd key={uid} uid={uid} name={item.name} description={item.description} />
							) : null
						})}
					</List>
				</Box>
			</SortableContext>
		</DndContext>
	)
}
