import { useState, type ReactNode } from 'react'
import {
	Box,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TablePagination,
	TableRow,
	TextField
} from '@mui/material'

interface Column<T> {
	label: string
	value: (row: T) => string
	render?: (row: T) => ReactNode
}

interface Props<T> {
	label: string
	rows: T[]
	columns: Column<T>[]
	rowId: (row: T) => string
	selected?: string
}

export default function ConfigurationTable<T>({ label, rows, columns, rowId, selected }: Props<T>) {
	const [term, setTerm] = useState('')
	const [page, setPage] = useState(0)
	const [size, setSize] = useState(25)
	const needle = term.toLowerCase().trim()
	const visible = needle ? rows.filter(row => columns.some(c => c.value(row).toLowerCase().includes(needle))) : rows
	const current = Math.min(page, Math.max(0, Math.ceil(visible.length / size) - 1))

	return (
		<Box>
			{(rows.length > 10 || term) && (
				<TextField
					type='search'
					size='small'
					label={'Filter ' + label.toLowerCase()}
					value={term}
					onChange={e => {
						setTerm(e.target.value)
						setPage(0)
					}}
					sx={{ m: 1.5 }}
				/>
			)}
			<TableContainer>
				<Table size='small' aria-label={label}>
					<TableHead sx={{ whiteSpace: 'nowrap', bgcolor: 'action.hover' }}>
						<TableRow>
							{columns.map(c => (
								<TableCell key={c.label}>{c.label}</TableCell>
							))}
						</TableRow>
					</TableHead>
					<TableBody>
						{visible.slice(current * size, (current + 1) * size).map(row => (
							<TableRow key={rowId(row)} selected={Boolean(selected) && rowId(row) === selected}>
								{columns.map(c => (
									<TableCell
										key={c.label}
										sx={{ minWidth: '7ch', maxWidth: '38ch', overflowWrap: 'anywhere' }}
									>
										{c.render ? c.render(row) : c.value(row)}
									</TableCell>
								))}
							</TableRow>
						))}
						{visible.length === 0 && (
							<TableRow>
								<TableCell colSpan={columns.length}>
									{rows.length
										? 'No matching records.'
										: 'No ' + label.toLowerCase() + ' in this configuration.'}
								</TableCell>
							</TableRow>
						)}
					</TableBody>
				</Table>
			</TableContainer>
			{visible.length > 25 && (
				<TablePagination
					component='div'
					count={visible.length}
					page={current}
					rowsPerPage={size}
					rowsPerPageOptions={[25, 50, 100]}
					onPageChange={(_, next) => setPage(next)}
					onRowsPerPageChange={e => {
						setSize(Number(e.target.value))
						setPage(0)
					}}
				/>
			)}
		</Box>
	)
}
