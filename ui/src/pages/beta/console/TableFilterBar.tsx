import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { Autocomplete, Box, Button, ButtonBase, Chip, Stack, TextField, Typography } from '@mui/material'
import CloseOutlined from '@mui/icons-material/CloseOutlined'
import SearchOutlined from '@mui/icons-material/SearchOutlined'
import { consoleMonospace } from '../theme'
import {
	formatFilters,
	operatorsFor,
	parseFilters,
	type FilterField,
	type FilterOperator,
	type FilterToken
} from './tableFilters'

interface Props {
	fields: readonly FilterField[]
	query: string
	onChange: (value: string, replace?: boolean) => void
	suggestions: (field: string, search: string) => readonly string[]
	activeFilters?: readonly { label: string; onDelete: () => void }[]
	onClear?: () => void
}

const signs: Record<FilterOperator, string> = { ':': '=', '!=': '≠', '>': '>', '<': '<', '>=': '≥', '<=': '≤' }
const comparisons: Record<FilterOperator, string> = {
	':': 'is',
	'!=': 'is not',
	'>': 'greater than',
	'<': 'less than',
	'>=': 'at least',
	'<=': 'at most'
}
const dateComparisons: Record<FilterOperator, string> = {
	':': 'on',
	'!=': 'not on',
	'>': 'after',
	'<': 'before',
	'>=': 'on or after',
	'<=': 'on or before'
}

// Keep preceding search words when the last word(s) were used to find a field.
function textBeforeField(text: string, field: FilterField) {
	const names = [field.key.toLowerCase(), field.label.toLowerCase()]
	const words = text.trimEnd()
	const from = Math.max(0, words.length - Math.max(...names.map(name => name.length)))
	for (let i = from; i < words.length; i++) {
		if (i && !/\s/.test(words[i - 1])) continue
		const suffix = words.slice(i).toLowerCase()
		if (names.some(name => name.includes(suffix))) return words.slice(0, i).trimEnd()
	}
}

export default function TableFilterBar({ fields, query, onChange, suggestions, activeFilters = [], onClear }: Props) {
	const parsed = useMemo(() => parseFilters(query, fields), [fields, query])
	const tokens = parsed.error ? [] : parsed.tokens
	const freeText = parsed.error ? query : parsed.text
	const [draft, setDraft] = useState<string | null>(null)
	const [fieldKey, setFieldKey] = useState('')
	const [operator, setOperator] = useState<FilterOperator>()
	const [entry, setEntry] = useState('')
	const [error, setError] = useState('')
	const [open, setOpen] = useState(false)
	const expectedQuery = useRef(query)
	const liveEdit = useRef(false)
	const input = useRef<HTMLInputElement>(null)
	const inputId = useId()
	const field = fields.find(f => f.key === fieldKey)
	const value = field ? entry : draft ?? freeText
	const operatorNames = field?.filterType === 'date' ? dateComparisons : comparisons
	const heading = !field ? 'Filter by field' : field.label + (operator ? ' · Value' : ' · Operator')

	useEffect(() => {
		// Browser navigation can replace the search while the operator is composing a filter.
		if (query !== expectedQuery.current) {
			setDraft(null)
			setFieldKey('')
			setOperator(undefined)
			setEntry('')
			setOpen(false)
			setError('')
			expectedQuery.current = query
			liveEdit.current = false
		}
	}, [query])

	function write(next: string, replace = false) {
		if (query === next) return
		expectedQuery.current = next
		onChange(next, replace)
	}

	function close() {
		liveEdit.current = false
		setFieldKey('')
		setOperator(undefined)
		setEntry('')
		setError('')
		setOpen(false)
	}

	function commit(nextTokens: readonly FilterToken[], text: string, finishEdit = false) {
		write(formatFilters(nextTokens, text), finishEdit && liveEdit.current)
		setDraft(null)
		close()
	}

	function type(text: string) {
		setError('')
		setOpen(true)
		if (field) {
			setEntry(text)
			return
		}
		setDraft(text)
		const typed = parseFilters(text, fields)
		const next = formatFilters(tokens, text)
		if (!typed.tokens.length && !typed.error && next !== query) {
			// One history entry per edit, including any text typed before the first operator.
			write(next, liveEdit.current)
			liveEdit.current = true
		}
	}

	function applyText() {
		const typed = parseFilters(value, fields)
		if (typed.error) setError(typed.error)
		else commit([...tokens, ...typed.tokens], typed.text, true)
	}

	function choose(option: string) {
		if (!field) {
			setFieldKey(option)
			setEntry('')
		} else if (!operator) {
			setOperator(option as FilterOperator)
			setEntry('')
		} else {
			const candidate = formatFilters([{ field: field.key, operator, value: option }], '')
			const checked = parseFilters(candidate, fields)
			if (checked.error) setError(checked.error)
			else
				commit(
					[...tokens, ...checked.tokens],
					(draft == null ? undefined : textBeforeField(draft, field)) ?? freeText,
					true
				)
		}
	}

	function optionLabel(option: string) {
		if (!field) return fields.find(f => f.key === option)?.label ?? option
		return operator ? option : operatorNames[option as FilterOperator] ?? option
	}
	const options = useMemo(() => {
		if (!open) return []
		const term = value.trim().toLowerCase()
		if (!field) {
			const matches = fields.filter(f => textBeforeField(value, f) !== undefined)
			return (matches.length ? matches : fields).map(f => f.key)
		}
		if (!operator)
			return operatorsFor(field).filter(
				op => op.includes(term) || signs[op].includes(term) || operatorNames[op].includes(term)
			)
		const items = field.filterValues ?? suggestions(field.key, term)
		return items.filter(v => v.toLowerCase().includes(term)).slice(0, 40)
	}, [field, fields, open, operator, operatorNames, suggestions, value])

	function remove(index: number) {
		commit(
			tokens.filter((_, i) => i !== index),
			freeText
		)
		input.current?.focus()
	}

	return (
		<Stack spacing={0.5} sx={{ flex: '1 1 18rem', minWidth: 0 }}>
			<Typography component='label' htmlFor={inputId} variant='caption' fontWeight={600}>
				Search and filter
			</Typography>
			<Autocomplete<string, false, true, true>
				id={inputId}
				freeSolo
				disableClearable
				disableCloseOnSelect
				openOnFocus
				selectOnFocus={false}
				open={open}
				onOpen={() => setOpen(true)}
				onClose={(_, reason) => {
					if (reason !== 'selectOption') close()
				}}
				options={options}
				value=''
				inputValue={value}
				onInputChange={(_, text, reason) => {
					if (reason === 'input') type(text)
				}}
				onChange={(_, item, reason) => {
					if (reason === 'selectOption') choose(item)
					else if (reason === 'createOption') {
						if (field && operator) choose(item)
						else if (!field) applyText()
					}
				}}
				groupBy={() => heading}
				getOptionLabel={optionLabel}
				filterOptions={items => items}
				onKeyDown={event => {
					if (event.target !== input.current || event.nativeEvent.isComposing) return
					const highlighted = input.current?.getAttribute('aria-activedescendant')
					if (event.key === 'Enter' && !highlighted) {
						event.defaultMuiPrevented = true
						event.preventDefault()
						if (field && operator && value) choose(value)
						else if (!field && value) applyText()
						else if (open && options[0]) choose(options[0])
						else if (field && !operator) setError('Choose a listed operator.')
					} else if (event.key === 'Backspace' && !value) {
						if (operator) setOperator(undefined)
						else if (field) setFieldKey('')
						else if (tokens.length) remove(tokens.length - 1)
						else if (activeFilters.length) activeFilters[activeFilters.length - 1].onDelete()
						else return
						setError('')
						event.preventDefault()
					}
				}}
				slotProps={{
					popper: { sx: { maxWidth: 'min(24rem, calc(100vw - 2rem))' } },
					listbox: { sx: { py: 0.5, '& .MuiAutocomplete-option': { minHeight: 32 } } }
				}}
				renderOption={({ key, ...props }, option) => (
					<li key={key} {...props}>
						<Box component='span' sx={{ flex: 1, overflowWrap: 'anywhere' }}>
							{optionLabel(option)}
						</Box>
						<Typography
							variant='caption'
							color='text.secondary'
							sx={{ ml: 1, textTransform: 'capitalize' }}
						>
							{!field
								? fields.find(f => f.key === option)?.filterType ?? 'text'
								: !operator
									? signs[option as FilterOperator]
									: ''}
						</Typography>
					</li>
				)}
				renderInput={params => (
					<TextField
						{...params}
						size='small'
						inputRef={input}
						placeholder={
							field
								? operator
									? field.filterType === 'date'
										? 'YYYY-MM-DD'
										: 'Enter a value'
									: 'Choose an operator'
								: 'Search all fields or add a filter'
						}
						error={Boolean(error || parsed.error)}
						helperText={error || parsed.error}
						InputProps={{
							...params.InputProps,
							startAdornment: (
								<>
									<SearchOutlined fontSize='small' color='action' aria-hidden />
									{activeFilters.map((active, index) => (
										<Chip
											key={index + active.label}
											size='small'
											label={active.label}
											onDelete={active.onDelete}
										/>
									))}
									{tokens.map((token, index) => {
										const expression = formatFilters([token], '')
										return (
											<ButtonBase
												key={expression + index}
												aria-label={'Remove ' + expression}
												title={'Remove ' + expression}
												onClick={() => remove(index)}
												sx={{
													minHeight: 26,
													minWidth: 0,
													maxWidth: '100%',
													gap: 0.5,
													px: 0.75,
													border: 1,
													borderColor: 'divider',
													borderRadius: 0.75,
													fontFamily: consoleMonospace,
													fontSize: 12,
													bgcolor: 'action.hover',
													'&:focus-visible': {
														outline: '2px solid',
														outlineColor: 'primary.main'
													}
												}}
											>
												<Box
													component='span'
													sx={{
														overflow: 'hidden',
														textOverflow: 'ellipsis',
														whiteSpace: 'nowrap'
													}}
												>
													{fields.find(f => f.key === token.field)?.label}{' '}
													{signs[token.operator]}{' '}
													<Box component='span' sx={{ color: 'primary.main' }}>
														{token.value}
													</Box>
												</Box>
												<CloseOutlined sx={{ fontSize: 14 }} />
											</ButtonBase>
										)
									})}
									{field && (
										<Typography variant='caption' fontWeight={600}>
											{field.label} {operator && signs[operator]}
										</Typography>
									)}
								</>
							),
							endAdornment: Boolean(query || value || field || activeFilters.length) && (
								<Button
									aria-label='Clear filters'
									size='small'
									sx={{ minWidth: 0 }}
									onClick={() => {
										if (onClear) {
											setDraft(null)
											close()
											onClear()
										} else commit([], '')
										input.current?.focus()
									}}
								>
									Clear
								</Button>
							)
						}}
					/>
				)}
				sx={{
					'& .MuiAutocomplete-inputRoot': { gap: 0.5, flexWrap: 'wrap' },
					'& .MuiAutocomplete-input': { minWidth: '9em !important' }
				}}
			/>
		</Stack>
	)
}
