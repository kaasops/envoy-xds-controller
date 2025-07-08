import React from 'react'
import { useNavigate } from 'react-router-dom'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Button from '@mui/material/Button'
import LockIcon from '@mui/icons-material/Lock'

interface IForbiddenPageProps {
	title?: string
}

const ForbiddenPage: React.FC<IForbiddenPageProps> = () => {
	const navigate = useNavigate()

	return (
		<Box
			display='flex'
			flexDirection='column'
			alignItems='center'
			justifyContent='center'
			height='100%'
			textAlign='center'
			p={3}
		>
			<LockIcon sx={{ fontSize: 80, color: 'error.main', mb: 2 }} />
			<Typography variant='h4' gutterBottom>
				Access denied
			</Typography>
			<Typography variant='body1' color='text.secondary' mb={3}>
				You do not have permission to view this page.
			</Typography>
			<Button variant='contained' onClick={() => navigate(-1)}>
				Return to previous
			</Button>
		</Box>
	)
}

export default ForbiddenPage
