import { ConsoleTime } from '../time/ConsoleTime'
import { useConsoleTime } from '../time/useConsoleTime'
import TableFilterBar from './TableFilterBar'
import { createRowFilter, parseFilters, type FilterColumn } from './tableFilters'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { useNavigate, useOutletContext, useSearchParams } from 'react-router-dom'
import {
	Alert,
	Box,
	Button,
	Dialog,
	IconButton,
	LinearProgress,
	Paper,
	Stack,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TablePagination,
	TableRow,
	TableSortLabel,
	Typography,
	useMediaQuery,
	useTheme
} from '@mui/material'
import ChevronRightOutlined from '@mui/icons-material/ChevronRightOutlined'
import CloseOutlined from '@mui/icons-material/CloseOutlined'
import RefreshOutlined from '@mui/icons-material/RefreshOutlined'
import { Code, ConnectError } from '@connectrpc/connect'
import { apiError, httpStatus } from '../api'
import { consoleMonospace } from '../theme'

export interface Column<T> extends FilterColumn<T> {
	identifier?: boolean
	filterable?: boolean
	sortValue?: (row: T) => number
	render?: (row: T) => ReactNode
	align?: 'left' | 'right'
	filterOnly?: boolean
}

export interface SectionTableProps<T> {
	title: string
	tableTitle?: string
	tableHeading?: string
	summary?: ReactNode
	rows: readonly T[]
	columns: Column<T>[]
	rowId: (row: T) => string
	rowHref?: (row: T) => string
	detail?: (row: T) => ReactNode
	loading?: boolean
	error?: unknown
	updatedAt?: number
	onRefresh?: () => void
	description?: ReactNode
	actions?: ReactNode
	filters?: ReactNode
	emptyText?: string
	localState?: boolean
}

const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })
const pageSizes = [10, 25, 50, 100]
const interactiveElements = 'a, button, input, select, textarea, [role="button"]'
const values = (value: string | number | readonly string[]): readonly string[] =>
	typeof value === 'string' || typeof value === 'number' ? [String(value)] : value

export default function SectionTable<T>({
	title,
	tableTitle,
	tableHeading = tableTitle,
	summary,
	rows,
	columns,
	rowId,
	rowHref,
	detail,
	loading = false,
	error,
	updatedAt,
	onRefresh,
	description,
	actions,
	filters,
	emptyText,
	localState = false
}: SectionTableProps<T>) {
	const navigate = useNavigate()
	const [urlParams, setUrlParams] = useSearchParams()
	const [privateParams, setPrivateParams] = useState(() => new URLSearchParams())
	const params = localState ? privateParams : urlParams
	const wide = useMediaQuery(useTheme().breakpoints.up('lg'))
	const panelTitle = useId()
	const closeButton = useRef<HTMLButtonElement>(null)
	const opener = useRef<HTMLButtonElement | null>(null)
	const query = params.get('q') || ''
	const { timeZone } = useConsoleTime()
	const fields = columns.filter(column => column.filterable !== false)
	const parsed = parseFilters(query, fields)
	const term = parsed.text.toLowerCase().trim()
	const matchesFilters = createRowFilter(parsed.tokens, fields, timeZone)
	const property = params.get('by') || ''
	const value = params.get('value') || ''
	const selectedId = params.get('selected')
	const sortKey = params.get('sort')
	const direction = params.get('direction') === 'desc' ? 'desc' : 'asc'
	const propertyColumn = columns.find(c => c.key === property || c.label === property)
	const sortColumn = columns.find(c => c.key === sortKey)
	const tableColumns = columns.filter(column => !column.filterOnly)
	const requestedSize = Number(params.get('size'))
	const size = pageSizes.includes(requestedSize) ? requestedSize : 25
	const requestedPage = Number(params.get('page') || 1)
	const namedFilters = ['node', 'status'].flatMap(key => {
		const filter = params.get(key)
		const column = columns.find(c => c.key === key || (key === 'node' && c.key === 'nodeId'))
		return filter && column ? [{ key, column, filter }] : []
	})
	const forbidden =
		[401, 403].includes(httpStatus(error) || 0) ||
		(error instanceof ConnectError && (error.code === Code.PermissionDenied || error.code === Code.Unauthenticated))
	const filtered =
		forbidden || parsed.error
			? []
			: rows.filter(row => {
					if (value && (!propertyColumn || !values(propertyColumn.value(row)).includes(value))) return false
					if (
						namedFilters.some(({ key, column, filter }) => {
							if (key === 'node' && column.appliesToAllNodes?.(row)) return false
							return !values(column.value(row)).includes(filter)
						})
					)
						return false
					if (!matchesFilters(row)) return false
					return (
						!term || columns.some(c => values(c.value(row)).some(item => item.toLowerCase().includes(term)))
					)
				})
	if (sortColumn) {
		filtered.sort((a, b) => {
			const sortValue = sortColumn.sortValue ?? sortColumn.value
			const first = sortValue(a)
			const second = sortValue(b)
			const order =
				typeof first === 'number' && typeof second === 'number'
					? first - second
					: collator.compare(values(first).join(', '), values(second).join(', '))
			return direction === 'desc' ? -order : order
		})
	}
	const pages = Math.max(1, Math.ceil(filtered.length / size))
	const page = Number.isSafeInteger(requestedPage) ? Math.min(Math.max(requestedPage, 1), pages) - 1 : 0
	const selected =
		detail && selectedId !== null && !forbidden ? rows.find(row => rowId(row) === selectedId) : undefined
	const showPanel = selected !== undefined && Boolean(detail)
	const stale = Boolean(error && !forbidden && (rows.length || updatedAt))

	const reportCount = useOutletContext<((count?: number) => void) | null>()
	const count = loading || error || !updatedAt || localState ? undefined : filtered.length
	useEffect(() => {
		reportCount?.(count)
	}, [reportCount, count])

	function update(changes: Record<string, string | null>, replace = false) {
		const next = new URLSearchParams(params)
		for (const [key, entry] of Object.entries(changes)) {
			if (entry) next.set(key, entry)
			else next.delete(key)
		}
		if (localState) setPrivateParams(next)
		else setUrlParams(next, { replace, preventScrollReset: true })
	}

	function filterBy(changes: Record<string, string | null>, replace = false) {
		update({ ...changes, page: null, selected: null }, replace)
	}

	const legacyFilters = namedFilters.map(({ key, column, filter }) => ({
		label: column.label + ': ' + filter,
		onDelete: () => filterBy({ [key]: null })
	}))
	if (value) {
		legacyFilters.unshift({
			label: (propertyColumn?.label || property) + ': ' + value,
			onDelete: () => filterBy({ by: null, value: null })
		})
	}
	function clearSearch() {
		const changes: Record<string, null> = { q: null, by: null, value: null }
		for (const { key } of namedFilters) changes[key] = null
		filterBy(changes)
	}

	function suggestValues(key: string, search: string): string[] {
		const column = fields.find(field => field.key === key)
		if (!column || column.filterType === 'date') return []
		const needle = search.trim().toLowerCase()
		const suggestions = new Set<string>()
		for (const row of rows) {
			const raw = column.filterValue ? column.filterValue(row) : column.value(row)
			if (raw !== undefined) {
				for (const value of values(raw)) {
					if (value && value.toLowerCase().includes(needle)) suggestions.add(value)
					if (suggestions.size === 100) return [...suggestions]
				}
			}
		}
		return [...suggestions]
	}

	function dismiss() {
		update({ selected: null })
		opener.current?.focus()
	}

	useEffect(() => {
		if (wide && showPanel) closeButton.current?.focus()
	}, [selectedId, wide, showPanel])

	const panel = showPanel && detail && selected !== undefined && (
		<>
			<Stack
				direction='row'
				alignItems='center'
				justifyContent='space-between'
				sx={{ p: 1.5, borderBottom: 1, borderColor: 'divider' }}
			>
				<Typography id={panelTitle} component='h2' variant='h6' sx={{ overflowWrap: 'anywhere' }}>
					{tableColumns.length ? values(tableColumns[0].value(selected)).join(', ') : selectedId}
				</Typography>
				<IconButton ref={closeButton} aria-label='Close details' onClick={dismiss}>
					<CloseOutlined />
				</IconButton>
			</Stack>
			<Box sx={{ p: 2, overflow: 'auto' }}>{detail(selected)}</Box>
		</>
	)

	const updated = updatedAt !== undefined && updatedAt > 0 && (
		<Typography variant='caption' color='text.secondary'>
			Updated <ConsoleTime value={updatedAt} />
		</Typography>
	)

	return (
		<Stack spacing={2}>
			<Stack direction='row' gap={2} justifyContent='space-between' alignItems='flex-start' flexWrap='wrap'>
				<Box>
					<Typography component='h1' variant='h4'>
						{title}
					</Typography>
					{description && !forbidden && (
						<Box sx={{ mt: 0.5, color: 'text.secondary', typography: 'body2' }}>{description}</Box>
					)}
				</Box>
				<Stack direction='row' gap={1} flexWrap='wrap'>
					{!forbidden && actions}
					{onRefresh && (
						<Button
							variant='outlined'
							disabled={loading}
							onClick={onRefresh}
							startIcon={<RefreshOutlined />}
						>
							Refresh
						</Button>
					)}
				</Stack>
			</Stack>
			{!forbidden && summary}
			{error != null && (
				<Alert severity={stale ? 'warning' : 'error'}>
					{stale ? 'Refresh failed. Showing the last response. ' : 'Could not load this section. '}
					{apiError(error)}
				</Alert>
			)}
			{loading && <LinearProgress aria-label={'Loading ' + title} />}
			{!forbidden && (!error || stale) && (
				<>
					<Box
						sx={{
							display: 'grid',
							alignItems: 'start',
							gap: 2,
							gridTemplateColumns:
								wide && showPanel ? 'minmax(0, 3fr) minmax(18rem, 2fr)' : 'minmax(0, 1fr)'
						}}
					>
						<Paper variant='outlined' sx={{ overflow: 'hidden' }}>
							{tableHeading && (
								<Stack
									direction='row'
									alignItems='center'
									gap={1}
									flexWrap='wrap'
									sx={{ px: 2, py: 1.5, borderBottom: 1, borderColor: 'divider' }}
								>
									<Typography component='h2' variant='h6'>
										{tableHeading}
									</Typography>
									{count !== undefined && (
										<Box
											component='span'
											aria-live='polite'
											title={count + ' matching rows'}
											sx={{
												px: 0.75,
												borderRadius: '4px',
												bgcolor: 'action.hover',
												color: 'text.secondary',
												fontSize: 12,
												fontVariantNumeric: 'tabular-nums'
											}}
										>
											{query || value || namedFilters.length
												? count + ' of ' + rows.length
												: count}
										</Box>
									)}
									{updated && (
										<Box sx={{ ml: { sm: 'auto' }, flexBasis: { xs: '100%', sm: 'auto' } }}>
											{updated}
										</Box>
									)}
								</Stack>
							)}
							<Stack
								direction='row'
								alignItems='center'
								gap={1.25}
								flexWrap='wrap'
								sx={{ px: 2, py: 1.5, borderBottom: 1, borderColor: 'divider' }}
							>
								{filters}
								<TableFilterBar
									fields={fields}
									query={query}
									onChange={(next, replace) => filterBy({ q: next }, replace)}
									suggestions={suggestValues}
									activeFilters={legacyFilters}
									onClear={clearSearch}
								/>
							</Stack>
							<TableContainer>
								<Table size='small' aria-label={tableTitle ?? title}>
									<TableHead sx={{ whiteSpace: 'nowrap', overflowWrap: 'normal' }}>
										<TableRow sx={{ bgcolor: 'action.hover' }}>
											{tableColumns.map(c => (
												<TableCell
													key={c.key}
													align={c.align}
													sortDirection={sortKey === c.key ? direction : false}
												>
													<TableSortLabel
														active={sortKey === c.key}
														direction={sortKey === c.key ? direction : 'asc'}
														onClick={() =>
															update({
																sort: c.key,
																direction:
																	sortKey === c.key && direction === 'asc'
																		? 'desc'
																		: null,
																page: null
															})
														}
													>
														{c.label}
													</TableSortLabel>
												</TableCell>
											))}
											{detail && <TableCell align='right'>Details</TableCell>}
										</TableRow>
									</TableHead>
									<TableBody>
										{filtered.slice(page * size, (page + 1) * size).map(row => {
											const id = rowId(row)
											return (
												<TableRow
													hover
													key={id}
													selected={selectedId === id}
													tabIndex={rowHref ? 0 : undefined}
													sx={
														rowHref
															? {
																	cursor: 'pointer',
																	'&:focus-visible': {
																		outline: '2px solid',
																		outlineColor: 'primary.main',
																		outlineOffset: -2
																	}
																}
															: undefined
													}
													onClick={
														rowHref
															? event => {
																	const target = event.target
																	if (
																		target instanceof Element &&
																		event.currentTarget.contains(target) &&
																		!target.closest(interactiveElements)
																	)
																		navigate(rowHref(row))
																}
															: undefined
													}
													onKeyDown={
														rowHref
															? event => {
																	if (
																		event.target === event.currentTarget &&
																		event.key === 'Enter'
																	) {
																		event.preventDefault()
																		navigate(rowHref(row))
																	}
																}
															: undefined
													}
												>
													{tableColumns.map(c => (
														<TableCell
															key={c.key}
															align={c.align}
															sx={{
																fontFamily: c.identifier ? consoleMonospace : undefined,
																minWidth: c.identifier ? '16ch' : '9ch',
																maxWidth: '34ch',
																overflowWrap: 'anywhere'
															}}
														>
															{c.render ? c.render(row) : values(c.value(row)).join(', ')}
														</TableCell>
													))}
													{detail && (
														<TableCell align='right'>
															<IconButton
																size='small'
																aria-label={'Details for ' + id}
																onClick={e => {
																	opener.current = e.currentTarget
																	update({ selected: id })
																}}
															>
																<ChevronRightOutlined fontSize='small' />
															</IconButton>
														</TableCell>
													)}
												</TableRow>
											)
										})}
										{!filtered.length && !loading && (
											<TableRow>
												<TableCell
													colSpan={tableColumns.length + (detail ? 1 : 0)}
													sx={{ py: 3 }}
												>
													{rows.length
														? 'No records match these filters.'
														: emptyText || 'No records available.'}
												</TableCell>
											</TableRow>
										)}
									</TableBody>
								</Table>
							</TableContainer>
							{filtered.length > 10 && (
								<TablePagination
									component='div'
									count={filtered.length}
									page={page}
									rowsPerPage={size}
									rowsPerPageOptions={pageSizes}
									onPageChange={(_, next) => update({ page: String(next + 1) })}
									onRowsPerPageChange={e => update({ size: e.target.value, page: null })}
								/>
							)}
						</Paper>
						{wide && showPanel && (
							<Paper
								component='aside'
								aria-labelledby={panelTitle}
								variant='outlined'
								sx={{ minWidth: 0 }}
								onKeyDown={e => {
									if (e.key === 'Escape' && !e.defaultPrevented) {
										e.preventDefault()
										dismiss()
									}
								}}
							>
								{panel}
							</Paper>
						)}
					</Box>
					{!wide && showPanel && (
						<Dialog open fullWidth maxWidth='md' aria-labelledby={panelTitle} onClose={dismiss}>
							{panel}
						</Dialog>
					)}
					{!loading && detail && selectedId && !showPanel && (
						<Alert severity='info' action={<Button onClick={dismiss}>Dismiss</Button>}>
							The selected record is no longer available.
						</Alert>
					)}

					{!tableHeading && updated}
				</>
			)}
		</Stack>
	)
}
