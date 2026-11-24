import { useState } from 'react'
import { Box, IconButton, Link, Tooltip } from '@mui/material'
import ContentCopyOutlined from '@mui/icons-material/ContentCopyOutlined'
import { Link as RouterLink } from 'react-router-dom'
import copy from 'clipboard-copy'

export default function VersionValue({ value, to, label }: { value: string; to?: string; label?: string }) {
	const [notice, setNotice] = useState('')
	const short = value.length > 14
	const text = short ? value.slice(0, 5) + '…' + value.slice(-6) : value
	async function copyValue() {
		try {
			await copy(value)
			setNotice('Version copied')
		} catch {
			setNotice('Clipboard unavailable')
		}
	}
	return (
		<Box
			component='span'
			sx={{
				display: 'inline-flex',
				alignItems: 'center',
				gap: 0.25,
				fontFamily: 'monospace',
				whiteSpace: 'nowrap'
			}}
		>
			<Tooltip title={short ? value : ''} describeChild>
				{to ? (
					<Link component={RouterLink} to={to} aria-label={label}>
						{text}
					</Link>
				) : (
					<span tabIndex={short ? 0 : undefined} aria-label={short ? value : undefined}>
						{text}
					</span>
				)}
			</Tooltip>
			{short && (
				<Tooltip title={notice || 'Copy full version'} onClose={() => setNotice('')}>
					<IconButton size='small' aria-label={'Copy version ' + value} onClick={() => void copyValue()}>
						<ContentCopyOutlined sx={{ fontSize: 14 }} />
					</IconButton>
				</Tooltip>
			)}
			{notice && (
				<Box
					component='span'
					role='status'
					sx={{ position: 'absolute', clipPath: 'inset(50%)', width: 1, height: 1, overflow: 'hidden' }}
				>
					{notice}
				</Box>
			)}
		</Box>
	)
}
