import { Card, CardActionArea, CardContent, Typography } from '@mui/material';
import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';

interface INodeCard {
	node: string
}

function NodeCard({ node }: INodeCard): JSX.Element {
	const navigate = useNavigate();

	const openNodeZoneInfo = useCallback((nodeID: string) => {
		navigate(`${nodeID}`)
	}, [navigate])

	return (
		<Card key={node} >
			<CardActionArea onClick={() => openNodeZoneInfo(node)} sx={{ height: '100%' }}>
				<CardContent>
					<Typography gutterBottom variant="h5" component="div" margin={0}>
						{node}
					</Typography>
				</CardContent>
			</CardActionArea>
		</Card>
	)
}

export default NodeCard