import { Box, Typography } from '@mui/material'
import { ResourceVersions as ResourceVersionsType } from '../../common/types/overviewApiTypes'

interface ResourceVersionsProps {
	versions: ResourceVersionsType
}

export const ResourceVersions = ({ versions }: ResourceVersionsProps) => {
	const items = [
		{ label: 'Listeners', version: versions.listeners },
		{ label: 'Clusters', version: versions.clusters },
		{ label: 'Routes', version: versions.routes },
		{ label: 'Secrets', version: versions.secrets }
	]

	return (
		<Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
			<Typography variant='body2' color='text.secondary'>
				Versions:
			</Typography>
			{items.map((item, index) => (
				<Box key={item.label} sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
					<Typography variant='body2' color='text.secondary'>
						{item.label}
					</Typography>
					<Typography variant='body2' sx={{ fontFamily: 'monospace', fontWeight: 600 }}>
						{item.version}
					</Typography>
					{index < items.length - 1 && (
						<Typography variant='body2' color='text.disabled' sx={{ ml: 1 }}>
							|
						</Typography>
					)}
				</Box>
			))}
		</Box>
	)
}

export default ResourceVersions
