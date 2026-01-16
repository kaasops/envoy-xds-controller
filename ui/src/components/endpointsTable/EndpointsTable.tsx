import { useMemo } from 'react'
import { MaterialReactTable, useMaterialReactTable, type MRT_ColumnDef } from 'material-react-table'
import { Box, Chip } from '@mui/material'
import HttpIcon from '@mui/icons-material/Http'
import HttpsIcon from '@mui/icons-material/Https'
import LanIcon from '@mui/icons-material/Lan'
import { EndpointInfo } from '../../common/types/overviewApiTypes'
import { StatusBadge } from '../statusBadge'

interface EndpointsTableProps {
	endpoints: EndpointInfo[]
	isLoading?: boolean
}

const ProtocolChip = ({ protocol }: { protocol: string }) => {
	const config = {
		HTTP: { icon: <HttpIcon fontSize='small' />, color: 'warning' as const },
		HTTPS: { icon: <HttpsIcon fontSize='small' />, color: 'success' as const },
		TCP: { icon: <LanIcon fontSize='small' />, color: 'default' as const }
	}

	const { icon, color } = config[protocol as keyof typeof config] || config.TCP

	return <Chip icon={icon} label={protocol} color={color} size='small' variant='outlined' />
}

export const EndpointsTable = ({ endpoints, isLoading = false }: EndpointsTableProps) => {
	const columns = useMemo<MRT_ColumnDef<EndpointInfo>[]>(
		() => [
			{
				accessorKey: 'domain',
				header: 'Domain',
				size: 250
			},
			{
				accessorKey: 'port',
				header: 'Port',
				size: 80
			},
			{
				accessorKey: 'protocol',
				header: 'Protocol',
				size: 120,
				Cell: ({ cell }) => <ProtocolChip protocol={cell.getValue<string>()} />
			},
			{
				accessorKey: 'certificate.name',
				header: 'Certificate',
				size: 150,
				Cell: ({ row }) => {
					const cert = row.original.certificate
					return cert ? cert.name : <Box sx={{ color: 'text.disabled' }}>-</Box>
				}
			},
			{
				accessorKey: 'certificate.daysUntilExpiry',
				header: 'Expires',
				size: 100,
				Cell: ({ row }) => {
					const cert = row.original.certificate
					if (!cert) return <Box sx={{ color: 'text.disabled' }}>-</Box>
					return <StatusBadge status={cert.status} daysUntilExpiry={cert.daysUntilExpiry} />
				}
			},
			{
				accessorKey: 'listenerName',
				header: 'Listener',
				size: 200
			},
			{
				accessorKey: 'routeConfigName',
				header: 'Route Config',
				size: 200,
				Cell: ({ cell }) => {
					const value = cell.getValue<string>()
					return value || <Box sx={{ color: 'text.disabled' }}>-</Box>
				}
			}
		],
		[]
	)

	const table = useMaterialReactTable({
		columns,
		data: endpoints,
		enableColumnResizing: true,
		enableStickyHeader: true,
		enableGlobalFilter: true,
		enableColumnFilters: true,
		enablePagination: true,
		enableSorting: true,
		initialState: {
			showGlobalFilter: true,
			density: 'compact',
			pagination: { pageSize: 25, pageIndex: 0 }
		},
		state: {
			isLoading
		},
		muiTableContainerProps: {
			sx: { maxHeight: 'calc(100vh - 400px)' }
		},
		muiSearchTextFieldProps: {
			placeholder: 'Search domains...',
			sx: { minWidth: '300px' },
			variant: 'outlined',
			size: 'small'
		}
	})

	return <MaterialReactTable table={table} />
}

export default EndpointsTable
