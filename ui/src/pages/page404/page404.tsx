import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Container from '@mui/material/Container'
import SvgIcon from '@mui/material/SvgIcon'
import Typography from '@mui/material/Typography'

import ArrowBackOutlinedIcon from '@mui/icons-material/ArrowBackOutlined'
import { useNavigate } from 'react-router-dom'
import errorImg from '../../assets/errors/error-404.png'

const Page404 = () => {
	const navigate = useNavigate()

	return (
		<Box
			sx={{
				alignItems: 'center',
				display: 'flex',
				flexGrow: 1,
				height: '100vh'
			}}
		>
			<Container maxWidth='md'>
				<Box
					sx={{
						alignItems: 'center',
						display: 'flex',
						flexDirection: 'column'
					}}
				>
					<Box
						sx={{
							mb: 3,
							textAlign: 'center'
						}}
					>
						<img
							alt='Under development'
							src={errorImg}
							style={{
								display: 'inline-block',
								maxWidth: '100%',
								width: 400
							}}
						/>
					</Box>
					<Typography align='center' sx={{ mb: 3 }} variant='h3'>
						404: Page not found
					</Typography>
					<Typography align='center' color='text.secondary' variant='body1'>
						You have gone to a non-existent page
					</Typography>
					<Button
						color='primary'
						variant='contained'
						startIcon={
							<SvgIcon fontSize='medium'>
								<ArrowBackOutlinedIcon color='secondary' />
							</SvgIcon>
						}
						sx={{ mt: 3 }}
						onClick={() => navigate('/nodeIds')}
					>
						Go back
					</Button>
				</Box>
			</Container>
		</Box>
	)
}

export default Page404
