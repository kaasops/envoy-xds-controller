import { Box, Card, CardContent, Typography, useTheme, alpha } from '@mui/material'
import LanguageIcon from '@mui/icons-material/Language'
import DnsIcon from '@mui/icons-material/Dns'
import SecurityIcon from '@mui/icons-material/Security'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline'
import { OverviewSummary as OverviewSummaryType } from '../../common/types/overviewApiTypes'

interface OverviewSummaryProps {
	summary: OverviewSummaryType
}

interface SummaryCardProps {
	title: string
	value: number
	icon: React.ReactNode
	color: string
}

const SummaryCard = ({ title, value, icon, color }: SummaryCardProps) => (
	<Card sx={{ height: '100%' }}>
		<CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
			<Box display='flex' alignItems='center' justifyContent='space-between'>
				<Box>
					<Typography color='text.secondary' gutterBottom variant='body2' sx={{ mb: 0.5 }}>
						{title}
					</Typography>
					<Typography variant='h5' component='div' fontWeight='bold'>
						{value}
					</Typography>
				</Box>
				<Box
					sx={{
						backgroundColor: alpha(color, 0.1),
						borderRadius: '50%',
						p: 1,
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'center'
					}}
				>
					{icon}
				</Box>
			</Box>
		</CardContent>
	</Card>
)

export const OverviewSummary = ({ summary }: OverviewSummaryProps) => {
	const theme = useTheme()

	const cards = [
		{
			title: 'Domains',
			value: summary.totalDomains,
			icon: <LanguageIcon sx={{ fontSize: 28, color: 'secondary.main' }} />,
			color: theme.palette.secondary.main
		},
		{
			title: 'Endpoints',
			value: summary.totalEndpoints,
			icon: <DnsIcon sx={{ fontSize: 28, color: 'primary.main' }} />,
			color: theme.palette.primary.main
		},
		{
			title: 'Certificates',
			value: summary.totalCertificates,
			icon: <SecurityIcon sx={{ fontSize: 28, color: 'success.main' }} />,
			color: theme.palette.success.main
		},
		{
			title: 'Expiring Soon',
			value: summary.certificatesWarning + summary.certificatesCritical,
			icon: <WarningAmberIcon sx={{ fontSize: 28, color: 'warning.main' }} />,
			color: theme.palette.warning.main
		},
		{
			title: 'Expired',
			value: summary.certificatesExpired,
			icon: <ErrorOutlineIcon sx={{ fontSize: 28, color: 'error.main' }} />,
			color: theme.palette.error.main
		}
	]

	return (
		<Box sx={{ mb: 3 }}>
			<Box
				sx={{
					display: 'flex',
					gap: 2,
					flexWrap: { xs: 'wrap', md: 'nowrap' }
				}}
			>
				{cards.map(card => (
					<Box key={card.title} sx={{ flex: { xs: '1 1 45%', md: '1 1 0' }, minWidth: 0 }}>
						<SummaryCard {...card} />
					</Box>
				))}
			</Box>
		</Box>
	)
}

export default OverviewSummary
