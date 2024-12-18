import React, { useState } from 'react'
import { ISecretsResponse } from '../../common/types/getSecretsApiTypes.ts'
import { Box, List, ListItem, ListItemButton, ListItemText, Typography } from '@mui/material'
import { useParams } from 'react-router-dom'
import { useGetSecretCerts } from '../../api/hooks/useSecrets.ts'
import { AccordionTlsCerts } from '../accordionTlsCerts/accordionTlsCerts.tsx'

interface ISecretsTlsBlockProps {
	data: ISecretsResponse
}

const SecretsTlsBlock: React.FC<ISecretsTlsBlockProps> = ({ data }) => {
	const { nodeID } = useParams()
	const secretsTls = data.secrets.filter(secret => 'TlsCertificate' in secret.Type)

	const [openItem, setOpenItem] = useState<string | null>(null)
	const [spaceName, setSpaceName] = useState<string | null>(null)
	const [name, setName] = useState<string | null>(null)

	const { data: certs } = useGetSecretCerts(spaceName, name)

	const handleClick = (value: string) => {
		const [newSpaceName, newName] = value.split('/')
		if (openItem === value) {
			setOpenItem(null)
			setSpaceName(null)
			setName(null)
		} else {
			setSpaceName(newSpaceName)
			setName(newName)
			setOpenItem(prev => (prev === value ? null : value))
		}
	}

	const renderListItems = (value: string) => {
		const isOpen = openItem === value
		return (
			<Box key={value}>
				<ListItem key={value} sx={{ paddingX: 0 }}>
					<ListItemButton onClick={() => handleClick(value)} sx={{ paddingX: 1 }}>
						<ListItemText id={value} primary={value} />
					</ListItemButton>
				</ListItem>
				{isOpen && certs ? (
					<Box overflow='auto' className='boxForAccordion'>
						{certs.certs.map(cert => (
							<AccordionTlsCerts key={`${cert.serialNumber}-accordion`} cert={cert} />
						))}
					</Box>
				) : null}
			</Box>
		)
	}

	if (secretsTls.length === 0) {
		return (
			<Box display='flex' alignItems='center' justifyContent='center' gap={1} mt={5}>
				<Typography variant='h4'>
					No TLS certificates were found for this node: "{nodeID?.toUpperCase()}"
				</Typography>
			</Box>
		)
	}

	return (
		<Box className='secretsListBox'>
			<List sx={{ pl: 0, overflow: 'auto' }} className='mainList'>
				{secretsTls.map(secret => renderListItems(secret.name))}
			</List>
		</Box>
	)
}

export default SecretsTlsBlock
