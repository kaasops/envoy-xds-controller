import { useMemo } from 'react'
import { MaterialReactTable, useMaterialReactTable, type MRT_ColumnDef } from 'material-react-table'
import { Box, Chip, Tooltip } from '@mui/material'
import { CertificateInfo } from '../../common/types/overviewApiTypes'
import { StatusBadge } from '../statusBadge'

interface CertificatesTableProps {
	certificates: CertificateInfo[]
	isLoading?: boolean
}

const DNSNamesCell = ({ dnsNames }: { dnsNames: string[] }) => {
	if (!dnsNames || dnsNames.length === 0) {
		return <Box sx={{ color: 'text.disabled' }}>-</Box>
	}

	const displayNames = dnsNames.slice(0, 2)
	const remaining = dnsNames.length - 2

	return (
		<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
			{displayNames.map(name => (
				<Chip key={name} label={name} size='small' variant='outlined' sx={{ fontSize: '0.75rem' }} />
			))}
			{remaining > 0 && (
				<Tooltip title={dnsNames.slice(2).join(', ')}>
					<Chip label={`+${remaining}`} size='small' color='default' sx={{ fontSize: '0.75rem' }} />
				</Tooltip>
			)}
		</Box>
	)
}

const UsedByCell = ({ domains }: { domains: string[] }) => {
	if (!domains || domains.length === 0) {
		return <Box sx={{ color: 'text.disabled' }}>Not used</Box>
	}

	const displayDomains = domains.slice(0, 2)
	const remaining = domains.length - 2

	return (
		<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
			{displayDomains.map(domain => (
				<Chip
					key={domain}
					label={domain}
					size='small'
					color='primary'
					variant='outlined'
					sx={{ fontSize: '0.75rem' }}
				/>
			))}
			{remaining > 0 && (
				<Tooltip title={domains.slice(2).join(', ')}>
					<Chip label={`+${remaining}`} size='small' color='primary' sx={{ fontSize: '0.75rem' }} />
				</Tooltip>
			)}
		</Box>
	)
}

export const CertificatesTable = ({ certificates, isLoading = false }: CertificatesTableProps) => {
	const columns = useMemo<MRT_ColumnDef<CertificateInfo>[]>(
		() => [
			{
				accessorKey: 'name',
				header: 'Name',
				size: 150
			},
			{
				accessorKey: 'namespace',
				header: 'Namespace',
				size: 120
			},
			{
				accessorKey: 'dnsNames',
				header: 'DNS Names',
				size: 250,
				Cell: ({ cell }) => <DNSNamesCell dnsNames={cell.getValue<string[]>()} />
			},
			{
				accessorKey: 'notAfter',
				header: 'Expires',
				size: 180,
				Cell: ({ cell }) => {
					const value = cell.getValue<string>()
					if (!value) return <Box sx={{ color: 'text.disabled' }}>-</Box>
					try {
						return new Date(value).toLocaleDateString('en-US', {
							year: 'numeric',
							month: 'short',
							day: 'numeric'
						})
					} catch {
						return value
					}
				}
			},
			{
				accessorKey: 'daysUntilExpiry',
				header: 'Days Left',
				size: 100,
				Cell: ({ row }) => (
					<StatusBadge status={row.original.status} daysUntilExpiry={row.original.daysUntilExpiry} />
				)
			},
			{
				accessorKey: 'status',
				header: 'Status',
				size: 100,
				Cell: ({ row }) => <StatusBadge status={row.original.status} showDays={false} />
			},
			{
				accessorKey: 'usedByDomains',
				header: 'Used By',
				size: 250,
				Cell: ({ cell }) => <UsedByCell domains={cell.getValue<string[]>()} />
			},
			{
				accessorKey: 'subject',
				header: 'Subject',
				size: 200,
				Cell: ({ cell }) => {
					const value = cell.getValue<string>()
					return (
						<Tooltip title={value}>
							<Box
								sx={{
									maxWidth: 200,
									overflow: 'hidden',
									textOverflow: 'ellipsis',
									whiteSpace: 'nowrap'
								}}
							>
								{value}
							</Box>
						</Tooltip>
					)
				}
			},
			{
				accessorKey: 'issuer',
				header: 'Issuer',
				size: 200,
				Cell: ({ cell }) => {
					const value = cell.getValue<string>()
					return (
						<Tooltip title={value}>
							<Box
								sx={{
									maxWidth: 200,
									overflow: 'hidden',
									textOverflow: 'ellipsis',
									whiteSpace: 'nowrap'
								}}
							>
								{value}
							</Box>
						</Tooltip>
					)
				}
			}
		],
		[]
	)

	const table = useMaterialReactTable({
		columns,
		data: certificates,
		enableColumnResizing: true,
		enableStickyHeader: true,
		enableGlobalFilter: true,
		enableColumnFilters: true,
		enablePagination: true,
		enableSorting: true,
		initialState: {
			showGlobalFilter: true,
			density: 'compact',
			pagination: { pageSize: 25, pageIndex: 0 },
			sorting: [{ id: 'daysUntilExpiry', desc: false }],
			columnVisibility: {
				subject: false,
				issuer: false
			}
		},
		state: {
			isLoading
		},
		muiTableContainerProps: {
			sx: { maxHeight: 'calc(100vh - 400px)' }
		},
		muiSearchTextFieldProps: {
			placeholder: 'Search certificates...',
			sx: { minWidth: '300px' },
			variant: 'outlined',
			size: 'small'
		}
	})

	return <MaterialReactTable table={table} />
}

export default CertificatesTable
