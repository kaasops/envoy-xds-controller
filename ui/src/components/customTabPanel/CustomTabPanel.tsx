import Box from '@mui/material/Box'
import React from 'react'
import { SxProps, Theme } from '@mui/material/styles'

interface ICustomTabPanelProps {
	children?: React.ReactNode
	index: number
	value: number
	variant?: 'simple' | 'vertical' | 'minimal'
	style?: React.CSSProperties
	sx?: SxProps<Theme>
}

function CustomTabPanel(props: ICustomTabPanelProps) {
	const { children, value, index, variant = 'simple', sx, ...other } = props

	const getVariantStyles = (): SxProps<Theme> => {
		switch (variant) {
			case 'vertical':
				return {
					display: 'flex',
					flexDirection: 'column',
					gap: 2,
					pl: 1,
					height: '100%'
				}
			case 'minimal':
				return { pt: 2 }
			default:
				return {}
		}
	}

	return (
		<div
			role='tabpanel'
			hidden={value !== index}
			id={`${variant}-tabpanel-${index}`}
			aria-labelledby={`${variant}-tab-${index}`}
			style={{ width: '100%', flexGrow: 1, overflow: 'auto' }}
			{...other}
		>
			{value === index && (
				<Box
					sx={{ p: variant === 'minimal' ? 0 : 1, ...sx }}
					height={variant === 'minimal' ? 'auto' : '100%'}
					display='flex'
					flexDirection='column'
				>
					<Box sx={getVariantStyles()}>{children}</Box>
				</Box>
			)}
		</div>
	)
}

export default CustomTabPanel
