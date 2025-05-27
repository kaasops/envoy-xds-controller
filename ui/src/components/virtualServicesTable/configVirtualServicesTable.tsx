import React, { useCallback, useMemo, useState } from 'react'
import { MRT_ColumnDef, MRT_Row, MRT_VisibilityState, useMaterialReactTable } from 'material-react-table'
import { VirtualServiceListItem } from '../../gen/virtual_service/v1/virtual_service_pb.ts'
import { NodeIdsChip } from '../nodeIdsChip/nodeIdsChip.tsx'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import IconButton from '@mui/material/IconButton'
import Tooltip from '@mui/material/Tooltip'
import RefreshIcon from '@mui/icons-material/Refresh'
import { ListVirtualServicesResponse } from '../../gen/virtual_service/v1/virtual_service_pb'
import { QueryObserverResult, RefetchOptions } from '@tanstack/react-query'
import { Delete, Edit } from '@mui/icons-material'
import { useNavigate } from 'react-router-dom'
import { useVirtualServiceStore } from '../../store/setVsStore.ts'
import TravelExploreIcon from '@mui/icons-material/TravelExplore'
import RemoveRedEyeIcon from '@mui/icons-material/RemoveRedEye'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import { useTabStore } from '../../store/tabIndexStore.ts'
import { usePermissionsStore } from '../../store/permissionsStore.ts'
import { PermissionAction } from '../../utils/helpers/permissionsActions.ts'

interface IConfigVirtualServicesTable {
	groupId: string
	virtualServices: ListVirtualServicesResponse | undefined
	refetch: (options?: RefetchOptions | undefined) => Promise<QueryObserverResult<ListVirtualServicesResponse, Error>>
	isError: boolean
	isFetching: boolean
	setOpenDialog: React.Dispatch<React.SetStateAction<boolean>>
	setNameForDialog: React.Dispatch<React.SetStateAction<string>>
	setSelectedUid: React.Dispatch<React.SetStateAction<string>>
}

export const useConfigTable = ({
	groupId,
	virtualServices,
	refetch,
	isFetching,
	isError,
	setOpenDialog,
	setNameForDialog,
	setSelectedUid
}: IConfigVirtualServicesTable) => {
	const navigate = useNavigate()

	const canView = usePermissionsStore(state => state.hasPermission(groupId, PermissionAction.GetVirtualService))
	const canEdit = usePermissionsStore(state => state.hasPermission(groupId, PermissionAction.UpdateVirtualService))
	const canDelete = usePermissionsStore(state => state.hasPermission(groupId, PermissionAction.DeleteVirtualService))
	const canCreate = usePermissionsStore(state => state.hasPermission(groupId, PermissionAction.CreateVirtualService))

	const setVsInfo = useVirtualServiceStore(state => state.setVirtualService)
	const setViewMode = useViewModeStore(state => state.setViewMode)
	const setTabIndex = useTabStore(state => state.setTabIndex)
	const [columnVisibility, setColumnVisibility] = useState<MRT_VisibilityState>({ uid: false })

	const handleDeleteVS = useCallback(
		(row: MRT_Row<VirtualServiceListItem>) => {
			setNameForDialog(row.original.name)
			setOpenDialog(true)
			setSelectedUid(row.original.uid)
		},
		[setNameForDialog, setOpenDialog, setSelectedUid]
	)

	const handeCreateVs = useCallback(() => {
		setViewMode('edit')
		setTabIndex(0)
		navigate(`/accessGroups/${groupId}/virtualServices/createVs`)
	}, [navigate, setViewMode, setTabIndex, groupId])

	const openEditVsPage = useCallback(
		(row: MRT_Row<VirtualServiceListItem>) => {
			setVsInfo(row.original.uid, row.original.name)
			setViewMode('edit')
			setTabIndex(0)
			navigate(`/accessGroups/${groupId}/virtualServices/${row.original.uid}`)
		},
		[navigate, setVsInfo, setViewMode, setTabIndex, groupId]
	)

	const openEditDomainVsPage = useCallback(
		(row: MRT_Row<VirtualServiceListItem>) => {
			setVsInfo(row.original.uid, row.original.name)
			setViewMode('edit')
			setTabIndex(1)
			navigate(`/accessGroups/${groupId}/virtualServices/${row.original.uid}`)
		},
		[navigate, setVsInfo, setViewMode, setTabIndex, groupId]
	)

	const openReadOnlyVsPage = useCallback(
		(row: MRT_Row<VirtualServiceListItem>) => {
			setVsInfo(row.original.uid, row.original.name)
			setViewMode('read')
			setTabIndex(0)
			navigate(`/accessGroups/${groupId}/virtualServices/${row.original.uid}`)
		},
		[navigate, setVsInfo, setViewMode, setTabIndex, groupId]
	)

	const columns = useMemo<MRT_ColumnDef<VirtualServiceListItem>[]>(
		() => [
			{
				accessorKey: 'uid',
				header: 'UID',
				minSize: 350
			},
			{
				accessorKey: 'name',
				header: 'Name',
				minSize: 200
			},
			{
				accessorKey: 'nodeIds',
				header: 'Node IDs',
				minSize: 250,
				size: 300,
				Cell: ({ renderedCellValue }) =>
					Array.isArray(renderedCellValue) && <NodeIdsChip nodeIsData={renderedCellValue} />
			},
			{
				accessorFn: row => row?.template?.name || '',
				header: 'Template',
				minSize: 250
			},
			{
				accessorFn: row => row?.description || '',
				header: 'Description',
				minSize: 500
			}
		],
		[]
	)

	const table = useMaterialReactTable({
		columns,
		data: virtualServices?.items || [],

		enableRowActions: true,
		enableSorting: false,

		enableColumnResizing: true,

		enableStickyHeader: true,
		enableStickyFooter: true,
		enableFullScreenToggle: false,
		enableDensityToggle: false,

		state: {
			showGlobalFilter: true,
			isLoading: isFetching,
			showAlertBanner: isError,
			showProgressBars: isFetching,
			showSkeletons: isFetching,
			columnVisibility: columnVisibility
		},

		onColumnVisibilityChange: setColumnVisibility,
		enableGlobalFilterModes: true,
		globalFilterModeOptions: ['fuzzy', 'startsWith'],

		globalFilterFn: 'myCustomFilterFn',

		muiTableContainerProps: {
			sx: { overflow: 'auto', height: 'calc(100% - 120px)', minHeight: 'calc(80% - 120px)' }
		},
		muiTablePaperProps: {
			sx: { overflow: 'auto', height: '100%' }
		},
		muiTopToolbarProps: {
			sx: {
				'button[aria-label="Show/Hide search"]': {
					display: 'none'
				}
			}
		},
		displayColumnDefOptions: {
			'mrt-row-actions': {
				size: 210
				// grow: false
			}
		},

		renderRowActions: ({ row }) => (
			<Box display='flex' gap={1}>
				{row.original.isEditable ? (
					<>
						{canView && (
							<Tooltip placement='top-end' title='View Virtual Service'>
								<IconButton onClick={() => openReadOnlyVsPage(row)}>
									<RemoveRedEyeIcon />
								</IconButton>
							</Tooltip>
						)}
						{canEdit && (
							<>
								<Tooltip placement='top-end' title='Edit Virtual Service'>
									<IconButton onClick={() => openEditVsPage(row)}>
										<Edit />
									</IconButton>
								</Tooltip>

								<Tooltip placement='top-end' title='Edit Domain Virtual Service'>
									<IconButton onClick={() => openEditDomainVsPage(row)}>
										<TravelExploreIcon color='warning' />
									</IconButton>
								</Tooltip>
							</>
						)}

						{canDelete && (
							<Tooltip placement='top-end' title='Remove Virtual Service'>
								<IconButton onClick={() => handleDeleteVS(row)}>
									<Delete color='error' />
								</IconButton>
							</Tooltip>
						)}
					</>
				) : (
					<>
						{canView && (
							<Tooltip placement='top-end' title='View Virtual Service'>
								<IconButton onClick={() => openReadOnlyVsPage(row)}>
									<RemoveRedEyeIcon />
								</IconButton>
							</Tooltip>
						)}
					</>
				)}
			</Box>
		),

		renderTopToolbarCustomActions: () => (
			<Box display='flex' gap={1} alignItems='center'>
				<Tooltip arrow title='Re-request data'>
					<IconButton onClick={() => refetch()}>
						<RefreshIcon />
					</IconButton>
				</Tooltip>

				{canCreate && (
					<Button
						color='primary'
						onClick={handeCreateVs}
						variant='contained'
						disabled={isFetching}
						size='small'
						sx={{ fontSize: 15, height: 36 }}
					>
						Create VirtualService
					</Button>
				)}
			</Box>
		)
	})

	return { columns, table }
}
