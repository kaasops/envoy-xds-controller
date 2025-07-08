import React from 'react'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { useNavigate, useParams } from 'react-router-dom'
import { usePermissionsStore } from '../../store/permissionsStore.ts'
import { PermissionAction } from '../../utils/helpers/permissionsActions.ts'
import { actionButtonsVsBlock, actionButtonsVsWrapper } from './style.ts'

interface IActionButtonsVsProps {
	isCreateMode: boolean
	isEditable: boolean
	isFetchingCreateVs: boolean
	isFetchingUpdateVs: boolean

	handleResetForm(): void
}

export const ActionButtonsVs: React.FC<IActionButtonsVsProps> = ({
	isCreateMode,
	isEditable,
	isFetchingCreateVs,
	isFetchingUpdateVs,
	handleResetForm
}) => {
	const viewMode = useViewModeStore(state => state.viewMode)
	const setViewMode = useViewModeStore(state => state.setViewMode)

	const navigate = useNavigate()
	const { groupId } = useParams()

	const canEdit = usePermissionsStore(state =>
		state.hasPermission(groupId as string, PermissionAction.UpdateVirtualService)
	)

	return (
		<Box className='actionButtonsVsBlock' sx={{ ...actionButtonsVsBlock }}>
			<Box className='actionButtonsVsWrapper' sx={{ ...actionButtonsVsWrapper }}>
				<Button
					variant='outlined'
					loading={isFetchingCreateVs || isFetchingUpdateVs}
					disabled={!isEditable || viewMode === 'read'}
					onClick={() => navigate(-1)}
				>
					Back to Table
				</Button>
				<Button
					variant='contained'
					type='submit'
					loading={isFetchingCreateVs || isFetchingUpdateVs}
					disabled={!isEditable || viewMode === 'read'}
				>
					{isCreateMode
						? 'Create Virtual Service'
						: viewMode === 'read'
							? 'Read-Only Virtual Service'
							: 'Update Virtual Service'}
				</Button>
				<Button
					variant='outlined'
					color='warning'
					loading={isFetchingCreateVs || isFetchingUpdateVs}
					disabled={!isEditable || viewMode === 'read'}
					onClick={handleResetForm}
				>
					Reset form
				</Button>
			</Box>

			{viewMode === 'read' && canEdit && isEditable && (
				<Button variant='outlined' color='warning' onClick={() => setViewMode('edit')}>
					Enable Edit Form
				</Button>
			)}
		</Box>
	)
}
