import { Chip } from '@mui/material'
import type { StreamStatus } from './streams'

const colors: Partial<Record<StreamStatus, 'error' | 'warning' | 'success'>> = {
	NACK: 'error',
	'Awaiting ACK': 'warning',
	ACK: 'success'
}

export default function StreamResponseChip({ status, count }: { status: StreamStatus; count?: number }) {
	return (
		<Chip
			component='span'
			data-status
			variant='outlined'
			size='small'
			color={colors[status] ?? 'default'}
			label={count === undefined ? status : count + ' ' + status}
		/>
	)
}
