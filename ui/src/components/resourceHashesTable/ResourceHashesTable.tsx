import { useMemo, useState } from 'react'
import { MaterialReactTable, useMaterialReactTable, type MRT_ColumnDef } from 'material-react-table'
import { Box, Chip, Tooltip, Snackbar } from '@mui/material'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import { ResourceHashVersions, ResourceHashVersion } from '../../common/types/overviewApiTypes'

interface ResourceHashesTableProps {
	data: ResourceHashVersions | undefined
	isLoading?: boolean
}

interface FlatResourceHash {
	type: 'cluster' | 'listener' | 'route' | 'secret'
	name: string
	version: string
}

const truncateHash = (hash: string): string => {
	if (hash.length <= 19) return hash
	return `${hash.slice(0, 8)}...${hash.slice(-8)}`
}

const CopyableHash = ({ hash }: { hash: string }) => {
	const [showSnackbar, setShowSnackbar] = useState(false)

	const handleCopy = async () => {
		try {
			await navigator.clipboard.writeText(hash)
			setShowSnackbar(true)
		} catch (err) {
			console.error('Failed to copy:', err)
		}
	}

	return (
		<>
			<Tooltip title={`Click to copy: ${hash}`} arrow placement='top'>
				<Box
					component='code'
					onClick={handleCopy}
					sx={{
						fontFamily: 'monospace',
						fontSize: '0.85rem',
						backgroundColor: 'action.hover',
						px: 1,
						py: 0.5,
						borderRadius: 1,
						cursor: 'pointer',
						display: 'inline-flex',
						alignItems: 'center',
						gap: 0.5,
						'&:hover': {
							backgroundColor: 'action.selected'
						}
					}}
				>
					{truncateHash(hash)}
					<ContentCopyIcon sx={{ fontSize: '0.75rem', opacity: 0.5 }} />
				</Box>
			</Tooltip>
			<Snackbar
				open={showSnackbar}
				autoHideDuration={2000}
				onClose={() => setShowSnackbar(false)}
				message='Hash copied to clipboard'
				anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
			/>
		</>
	)
}

const TypeChip = ({ type }: { type: string }) => {
	const config: Record<string, { color: 'primary' | 'secondary' | 'success' | 'warning' }> = {
		listener: { color: 'primary' },
		cluster: { color: 'secondary' },
		route: { color: 'success' },
		secret: { color: 'warning' }
	}

	const { color } = config[type] || { color: 'primary' as const }

	return <Chip label={type} color={color} size='small' variant='outlined' />
}

export const ResourceHashesTable = ({ data, isLoading = false }: ResourceHashesTableProps) => {
	const flatData = useMemo<FlatResourceHash[]>(() => {
		if (!data) return []

		const result: FlatResourceHash[] = []

		const addResources = (
			resources: ResourceHashVersion[] | null,
			type: FlatResourceHash['type']
		) => {
			if (resources) {
				resources.forEach(r => result.push({ type, name: r.name, version: r.version }))
			}
		}

		addResources(data.listeners, 'listener')
		addResources(data.clusters, 'cluster')
		addResources(data.routes, 'route')
		addResources(data.secrets, 'secret')

		return result
	}, [data])

	const columns = useMemo<MRT_ColumnDef<FlatResourceHash>[]>(
		() => [
			{
				accessorKey: 'type',
				header: 'Type',
				size: 120,
				Cell: ({ cell }) => <TypeChip type={cell.getValue<string>()} />
			},
			{
				accessorKey: 'name',
				header: 'Resource Name',
				size: 350
			},
			{
				accessorKey: 'version',
				header: 'Hash',
				size: 180,
				Cell: ({ cell }) => <CopyableHash hash={cell.getValue<string>()} />
			}
		],
		[]
	)

	const table = useMaterialReactTable({
		columns,
		data: flatData,
		enableColumnActions: false,
		enableColumnFilters: true,
		enablePagination: true,
		enableSorting: true,
		enableDensityToggle: false,
		enableFullScreenToggle: false,
		enableHiding: false,
		initialState: {
			density: 'compact',
			sorting: [{ id: 'type', desc: false }],
			pagination: { pageSize: 25, pageIndex: 0 }
		},
		state: {
			isLoading
		},
		muiTableContainerProps: {
			sx: { maxHeight: '500px' }
		},
		muiTablePaperProps: {
			elevation: 0,
			sx: { border: '1px solid', borderColor: 'divider' }
		}
	})

	return <MaterialReactTable table={table} />
}

export default ResourceHashesTable
