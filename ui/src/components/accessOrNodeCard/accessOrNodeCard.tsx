import Card from '@mui/material/Card'
import CardActionArea from '@mui/material/CardActionArea'
import CardContent from '@mui/material/CardContent'
import Typography from '@mui/material/Typography'

import { useCallback } from 'react'
import { useNavigate } from 'react-router-dom'

interface IAccessOrNodeCard {
	entity: string
}

function AccessOrNodeCard({ entity }: IAccessOrNodeCard) {
	const navigate = useNavigate()

	const openEntityPage = useCallback(
		(entity: string) => {
			navigate(`${entity}`)
		},
		[navigate]
	)

	return (
		<Card key={entity}>
			<CardActionArea onClick={() => openEntityPage(entity)} sx={{ height: '100%', padding: 2 }}>
				<CardContent>
					<Typography gutterBottom variant='h5' component='div' margin={0}>
						{entity}
					</Typography>
				</CardContent>
			</CardActionArea>
		</Card>
	)
}

export default AccessOrNodeCard
