import { Box, Paper, Typography } from '@mui/material'
import type { ReactNode } from 'react'

export default function InventorySummary({ fields }: { fields: { label: string; value: ReactNode }[] }) {
	return (
		<Paper
			component='dl'
			variant='outlined'
			sx={{ m: 0, display: 'flex', flexWrap: 'wrap', gap: '1px', overflow: 'hidden', bgcolor: 'divider' }}
		>
			{fields.map(({ label, value }) => (
				<Box key={label} sx={{ flex: '1 1 10rem', minWidth: 0, px: 2, py: 1.25, bgcolor: 'background.paper' }}>
					<Typography component='dt' variant='caption' color='text.secondary' sx={{ fontWeight: 600 }}>
						{label}
					</Typography>
					<Typography
						component='dd'
						sx={{
							m: 0,
							mt: 0.25,
							fontWeight: 650,
							fontVariantNumeric: 'tabular-nums',
							overflowWrap: 'anywhere'
						}}
					>
						{value}
					</Typography>
				</Box>
			))}
		</Paper>
	)
}
