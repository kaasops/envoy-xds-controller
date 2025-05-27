import React, { useEffect, useRef, useState } from 'react'
import { useListVs } from '../../api/grpc/hooks/useVirtualService.ts'
import { MaterialReactTable, MRT_VisibilityState } from 'material-react-table'
import { useConfigTable } from './configVirtualServicesTable.tsx'
import DialogDeleteVS from '../dialogDeleteVs/dialogDeleteVs.tsx'
import { useLocation, useNavigate } from 'react-router-dom'
import Snackbar from '@mui/material/Snackbar'
import { Alert } from '@mui/material'
import { usePermissionsStore } from '../../store/permissionsStore.ts'
import { PermissionAction } from '../../utils/helpers/permissionsActions.ts'
import ForbiddenPage from '../../pages/forbiddenPage/forbiddenPage.tsx'

interface VirtualServicesTable {
	groupId: string
}

const VirtualServicesTable: React.FC<VirtualServicesTable> = ({ groupId }) => {
	const canGetListVs = usePermissionsStore(state =>
		state.hasPermission(groupId, PermissionAction.ListVirtualServices)
	)
	const { data: virtualServices, isError, isFetching, refetch } = useListVs(canGetListVs, groupId)

	const isFirstRender = useRef(true)

	const [openDialog, setOpenDialog] = useState(false)
	const [nameForDialog, setNameForDialog] = useState('')
	const [selectedUid, setSelectedUid] = useState('')
	const [columnVisibility, setColumnVisibility] = useState<MRT_VisibilityState>({})

	const location = useLocation()
	const navigate = useNavigate()

	const [openSnackBar, setOpenSnackBar] = useState(false)
	const [snackMessage, setSnackMessage] = useState<string | null>(null)

	useEffect(() => {
		if (location.state?.successMessage) {
			setSnackMessage(location.state.successMessage)
			setOpenSnackBar(true)

			navigate(location.pathname, { replace: true, state: null })
		}
	}, [location, navigate])

	const { table } = useConfigTable({
		groupId,
		virtualServices,
		refetch,
		isError,
		isFetching,
		setOpenDialog,
		setNameForDialog,
		setSelectedUid
	})

	//Загрузка состояния таблицы
	useEffect(() => {
		const columnVisibility = localStorage.getItem('columnVisibility_VS')

		if (columnVisibility) {
			setColumnVisibility(JSON.parse(columnVisibility))
		}

		isFirstRender.current = false
	}, [])

	//Сохранение видимости столбцов
	useEffect(() => {
		if (!isFirstRender.current) return
		localStorage.setItem('columnVisibility_VS', JSON.stringify(columnVisibility))
	}, [columnVisibility])

	if (!canGetListVs) {
		return <ForbiddenPage />
	}

	return (
		<>
			<MaterialReactTable table={table} />

			<DialogDeleteVS
				openDialog={openDialog}
				serviceName={nameForDialog}
				setOpenDialog={setOpenDialog}
				refetchServices={refetch}
				selectedUid={selectedUid}
				setSelectedUid={setSelectedUid}
			/>
			<Snackbar
				open={openSnackBar}
				autoHideDuration={3500}
				onClose={() => setOpenSnackBar(false)}
				anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
			>
				<Alert onClose={() => setOpenSnackBar(false)} severity='success' variant='filled' sx={{ width: '50%' }}>
					{snackMessage}
				</Alert>
			</Snackbar>
		</>
	)
}

export default VirtualServicesTable
