import React, { useRef, useState } from 'react'
import {
	Accordion,
	AccordionActions,
	AccordionDetails,
	AccordionSummary,
	Box,
	Button,
	Divider,
	Tooltip,
	Typography,
	useTheme
} from '@mui/material'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import { ICertificate } from '../../common/types/getSecretsApiTypes.ts'
import { Editor } from '@monaco-editor/react'
import { formatDate, isExpired } from '../../utils/helpers/dateService.ts'
import WarningIcon from '@mui/icons-material/Warning'

interface IAccordionTlsCertsProps {
	cert: ICertificate
}

export const AccordionTlsCerts: React.FC<IAccordionTlsCertsProps> = ({ cert }) => {
	const theme = useTheme()
	const editorRef = useRef<any>(null)
	const [tooltipOpen, setTooltipOpen] = useState(false)
	const [textTooltip, setTextTooltip] = useState('')

	function handleEditorDidMount(editor: any) {
		editorRef.current = editor
	}

	const certWithoutRaw = ({ raw, ...rest }: ICertificate) => rest

	const handleCopy = async () => {
		try {
			await navigator.clipboard.writeText(cert.raw)
			setTextTooltip('.PEM certificate successfully copied')
			setTooltipOpen(true)
			setTimeout(() => setTooltipOpen(false), 1000)
		} catch (error) {
			setTextTooltip('error with .PEM certificate')
			setTooltipOpen(true)
			setTimeout(() => setTooltipOpen(false), 1000)
			console.error('Failed to copy:', error)
		}
	}

	return (
		<>
			<Accordion sx={{ marginLeft: 2.5 }}>
				<AccordionSummary expandIcon={<ExpandMoreIcon />} aria-controls='panel3-content' id='panel3-header'>
					<Box display='flex' alignItems='center' gap={2}>
						{isExpired(cert.notAfter) && <WarningIcon color='warning' />}
						<Typography color={isExpired(cert.notAfter) ? 'error' : 'inherit'}>{cert.subject}</Typography>
						<Divider orientation='vertical' flexItem />
						<Typography color={isExpired(cert.notAfter) ? 'error' : 'inherit'}>
							expired on: {formatDate(cert.notAfter)}
						</Typography>
					</Box>
				</AccordionSummary>
				<AccordionDetails>
					<Editor
						onMount={handleEditorDidMount}
						height={180}
						defaultLanguage='json'
						value={JSON.stringify(certWithoutRaw(cert), null, 2)}
						theme={theme.palette.mode === 'light' ? 'light' : 'vs-dark'}
						options={{ readOnly: true, minimap: { enabled: false } }}
					/>
				</AccordionDetails>
				<AccordionActions>
					<Tooltip title={textTooltip} open={tooltipOpen} placement='top-end'>
						<Button onClick={handleCopy}>Copy .PEM Certificate</Button>
					</Tooltip>
				</AccordionActions>
			</Accordion>
			<Divider sx={{ marginY: 0.3, marginLeft: 2.5 }} />
		</>
	)
}
