import { Chip } from '@mui/material'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import WarningIcon from '@mui/icons-material/Warning'
import ErrorIcon from '@mui/icons-material/Error'
import CancelIcon from '@mui/icons-material/Cancel'
import { CertificateStatus } from '../../common/types/overviewApiTypes'

interface StatusBadgeProps {
	status: CertificateStatus
	daysUntilExpiry?: number
	showDays?: boolean
}

const statusConfig = {
	ok: {
		color: 'success' as const,
		icon: <CheckCircleIcon fontSize='small' />,
		label: 'OK'
	},
	warning: {
		color: 'warning' as const,
		icon: <WarningIcon fontSize='small' />,
		label: 'Warning'
	},
	critical: {
		color: 'error' as const,
		icon: <ErrorIcon fontSize='small' />,
		label: 'Critical'
	},
	expired: {
		color: 'error' as const,
		icon: <CancelIcon fontSize='small' />,
		label: 'Expired'
	}
}

export const StatusBadge = ({ status, daysUntilExpiry, showDays = true }: StatusBadgeProps) => {
	const config = statusConfig[status]

	const label =
		showDays && daysUntilExpiry !== undefined
			? daysUntilExpiry <= 0
				? 'Expired'
				: `${daysUntilExpiry}d`
			: config.label

	return <Chip icon={config.icon} label={label} color={config.color} size='small' variant='outlined' />
}

export default StatusBadge
