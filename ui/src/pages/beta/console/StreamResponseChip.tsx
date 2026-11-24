import { Chip } from '@mui/material'
import StatusHelp from './StatusHelp'
import type { StreamStatus } from './streams'

const colors: Partial<Record<StreamStatus, 'error' | 'warning' | 'success'>> = {
	NACK: 'error',
	'Awaiting ACK': 'warning',
	ACK: 'success'
}

export default function StreamResponseChip({
	status,
	count,
	to,
	label
}: {
	status: StreamStatus
	count?: number
	to?: string
	label?: string
}) {
	return (
		<StatusHelp status={status} to={to} label={label}>
			<Chip
				component='span'
				data-status
				variant='outlined'
				size='small'
				color={colors[status] ?? 'default'}
				label={count === undefined ? status : count + ' ' + status}
			/>
		</StatusHelp>
	)
}
