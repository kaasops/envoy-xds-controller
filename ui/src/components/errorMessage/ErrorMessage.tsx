import Typography from '@mui/material/Typography'
import errorInfo from '../../assets/errors/error.gif'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import ReplayIcon from '@mui/icons-material/Replay'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import { useNavigate } from 'react-router-dom'

interface IErrorMessage {
	error: string
}

function ErrorMessage({ error }: IErrorMessage) {
	const navigate = useNavigate()

	return (
		<Box display='flex' alignItems='center' justifyContent='center' height='100vh' flexDirection='column' gap={2}>
			<Typography variant='h4' gutterBottom>
				OOPS Something went wrong
			</Typography>

			<Typography variant='body1' color='textSecondary' mb={2}>
				An unexpected error occurred. Please try again.
			</Typography>

			<img
				src={errorInfo}
				alt='Error'
				style={{
					display: 'block',
					width: '250px',
					height: '250px',
					objectFit: 'contain',
					margin: '0 auto'
				}}
			/>

			<Typography variant='caption' color='error' sx={{ wordBreak: 'break-word', maxWidth: 400 }}>
				{error}
			</Typography>

			<Box display='flex' alignItems='center' gap={2}>
				<Button
					onClick={() => navigate(-1)}
					color='primary'
					variant='contained'
					startIcon={<ArrowBackIcon color='secondary' />}
				>
					Go back
				</Button>
				<Button onClick={() => navigate(0)} variant='contained' endIcon={<ReplayIcon color='secondary' />}>
					Try again
				</Button>
			</Box>
		</Box>
	)
}

export default ErrorMessage
