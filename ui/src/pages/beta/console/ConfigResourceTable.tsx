import { useEffect, useState, type ReactNode } from 'react'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Alert, Box, Button, Link, MenuItem, Stack, TextField, Typography } from '@mui/material'
import client from '../../../api/axiosApiClient'
import { httpStatus, manualQueryOptions } from '../api'
import SectionTable, { type Column } from './SectionTable'

interface Props<T extends { name: string }> {
	title: string
	columns: Column<T>[]
	load: (node: string, signal: AbortSignal) => Promise<T[]>
	detail: (row: T, node: string) => ReactNode
}

export function Fields({ entries }: { entries: [string, ReactNode][] }) {
	return (
		<Box component='dl' sx={{ m: 0 }}>
			{entries.map(([key, value]) => (
				<Box key={key} sx={{ mb: 1.25 }}>
					<Typography variant='caption' color='text.secondary' component='dt'>
						{key}
					</Typography>
					<Typography variant='body2' component='dd' sx={{ m: 0, overflowWrap: 'anywhere' }}>
						{value}
					</Typography>
				</Box>
			))}
		</Box>
	)
}

export function ResourceSelection({ name, children }: { name: string; children: ReactNode }) {
	const [params] = useSearchParams()
	const next = new URLSearchParams(params)
	next.set('selected', name)
	return (
		<Link component={RouterLink} to={{ search: next.toString() }} aria-label={'Inspect ' + name}>
			{children}
		</Link>
	)
}

function CopyMetadata({ data }: { data: Record<string, unknown> }) {
	const [message, setMessage] = useState('')
	async function copy() {
		try {
			await navigator.clipboard.writeText(JSON.stringify(data, null, 2))
			setMessage('Copied.')
		} catch {
			setMessage('Clipboard unavailable.')
		}
	}
	return (
		<Stack direction='row' alignItems='center' gap={1}>
			<Button size='small' variant='outlined' onClick={() => void copy()}>
				Copy metadata
			</Button>
			<Typography variant='caption' role='status'>
				{message}
			</Typography>
		</Stack>
	)
}

export default function ConfigResourceTable<T extends { name: string }>({ title, columns, load, detail }: Props<T>) {
	const [params, setParams] = useSearchParams()
	const ids = useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: ['beta', 'config-nodeIDs'],
		queryFn: async ({ signal }) => (await client.get<string[] | null>('/nodeIDs', { signal })).data ?? []
	})
	const denied = [401, 403].includes(httpStatus(ids.error) ?? 0)
	const nodes = denied ? [] : ids.data ?? []
	const requested = params.get('node')
	const node = requested ?? nodes[0] ?? ''
	const available = Boolean(node) && nodes.includes(node)
	useEffect(() => {
		if (!requested && node)
			setParams(
				current => {
					const next = new URLSearchParams(current)
					next.set('node', node)
					return next
				},
				{ replace: true }
			)
	}, [node, requested, setParams])
	const resource = useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		queryKey: ['beta', 'config', title, node],
		queryFn: ({ signal }) => load(node, signal),
		enabled: available
	})
	function selectNode(id: string) {
		setParams(current => {
			const next = new URLSearchParams(current)
			next.set('node', id)
			next.delete('selected')
			next.delete('page')
			return next
		})
	}
	async function refresh() {
		const latest = await ids.refetch()
		if (latest.isSuccess && latest.data.includes(node)) await resource.refetch()
	}
	return (
		<SectionTable
			title={title}
			tableHeading='Served resources'
			rows={available ? resource.data ?? [] : []}
			rowId={row => row.name}
			columns={columns}
			detail={row => (
				<Stack spacing={2}>
					{detail(row, node)}
					<CopyMetadata
						key={node + '/' + row.name}
						data={Object.fromEntries(columns.map(column => [column.label, column.value(row)]))}
					/>
				</Stack>
			)}
			updatedAt={available ? resource.dataUpdatedAt : ids.dataUpdatedAt}
			error={ids.error ?? resource.error}
			loading={ids.isFetching || resource.isFetching}
			onRefresh={() => void refresh()}
			description={
				requested && ids.isSuccess && !available ? (
					<Alert severity='warning'>This node has no visible cached snapshot. Select another node.</Alert>
				) : undefined
			}
			actions={
				<TextField
					select
					label='Node'
					size='small'
					sx={{ minWidth: 170, maxWidth: 320 }}
					value={available ? node : ''}
					onChange={event => selectNode(event.target.value)}
					disabled={!nodes.length}
				>
					{nodes.map(id => (
						<MenuItem value={id} key={id}>
							{id}
						</MenuItem>
					))}
				</TextField>
			}
			emptyText={
				node && available ? 'No ' + title.toLowerCase() + ' in this snapshot.' : 'No cached snapshot available.'
			}
		/>
	)
}
