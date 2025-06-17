import React, { useState } from 'react'
import {
	Control,
	Controller,
	FieldErrors,
	UseFormClearErrors,
	UseFormSetError,
	UseFormSetValue,
	useWatch
} from 'react-hook-form'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import IconButton from '@mui/material/IconButton'
import TextField from '@mui/material/TextField'

import { validationRulesVsForm } from '../../utils/helpers/validationRulesVsForm.ts'
import Card from '@mui/material/Card'
import DeleteIcon from '@mui/icons-material/Delete'
import { CustomCardContent, styleBox } from './style.ts'
import FormControl from '@mui/material/FormControl'
import FormHelperText from '@mui/material/FormHelperText'
import { useViewModeStore } from '../../store/viewModeVsStore.ts'
import Checkbox from '@mui/material/Checkbox'
import { useVerifyDomains } from '../../api/grpc/hooks/useVirtualService.ts'
import { TooltipVhDomains } from './tooltipVhDomains.tsx'
import { AddOrReplaceButtons } from './addOrReplaceButtons.tsx'

interface IVirtualHostDomainsProps {
	control: Control<IVirtualServiceForm>
	setValue: UseFormSetValue<IVirtualServiceForm>
	errors: FieldErrors<IVirtualServiceForm>
	setError: UseFormSetError<IVirtualServiceForm>
	clearErrors: UseFormClearErrors<IVirtualServiceForm>
}

export const VirtualHostDomains: React.FC<IVirtualHostDomainsProps> = ({
	control,
	errors,
	setError,
	clearErrors,
	setValue
}) => {
	const nameField = 'virtualHostDomains'
	const [newDomain, setNewDomain] = useState('')
	const [selectedDomains, setSelectedDomains] = useState<number[]>([])

	const readMode = useViewModeStore(state => state.viewMode) === 'read'

	// const domains = watch(nameField)
	const domainsWatch = useWatch({ control, name: nameField })
	// const stableDomains = useMemo(() => [...domainsWatch], [domainsWatch])
	const { data: verifyDomains } = useVerifyDomains(domainsWatch)

	const addDomain = () => {
		if (newDomain.trim() === '') return

		const inputDomains = newDomain
			.split('\n')
			.map(d => d.trim())
			.filter(Boolean)
		const currentDomains = domainsWatch
		const uniqueNewDomains = inputDomains.filter(d => !currentDomains.includes(d))

		if (uniqueNewDomains.length === 0) {
			setError(nameField, { type: 'manual', message: 'No new valid domains to add.' })
			return
		}

		const errorMessage = validationRulesVsForm.virtualHostDomains(uniqueNewDomains)

		if (errorMessage === true) {
			setValue(nameField, [...currentDomains, ...uniqueNewDomains])
			setNewDomain('')
			clearErrors(nameField)
		} else {
			setError(nameField, { type: 'manual', message: errorMessage })
			setNewDomain('')
		}
	}

	const removeDomain = (index: number) => {
		const domains = domainsWatch
		domains.splice(index, 1)
		setValue(nameField, [...domains])
	}

	const handleKeyPress = (e: React.KeyboardEvent<HTMLInputElement>) => {
		if (e.key === 'Enter') {
			e.preventDefault()
			addDomain()
		}
	}

	const toggleSelectDomain = (index: number) => {
		setSelectedDomains(prevState =>
			prevState.includes(index) ? prevState.filter(i => i !== index) : [...prevState, index]
		)
	}

	const removeSelectedDomains = () => {
		const remaining = domainsWatch.filter((_, i) => !selectedDomains.includes(i))
		setValue(nameField, remaining)
		setSelectedDomains([])
	}

	const toggleSelectAll = () => {
		const domains = domainsWatch
		if (selectedDomains.length === domains.length) {
			setSelectedDomains([])
		} else {
			setSelectedDomains(domains.map((_, i) => i))
		}
	}

	const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
		const file = e.target.files?.[0]
		if (!file) return
		e.target.value = ''

		const reader = new FileReader()
		reader.onload = () => {
			const content = reader.result as string

			const domainsFromFile = content
				.split('\n')
				.map(domain => domain.trim())
				.filter(Boolean)

			const currentDomains = domainsWatch

			const newDomains = [
				...currentDomains,
				...domainsFromFile.filter(domain => !currentDomains.includes(domain))
			]

			clearErrors(nameField)
			setValue(nameField, newDomains)
		}
		reader.readAsText(file)
	}

	return (
		<Box sx={{ ...styleBox }}>
			<Box display='flex' justifyContent='space-between' alignItems='center'>
				<TooltipVhDomains />
				<AddOrReplaceButtons control={control} setValue={setValue} mode={'virtualHostDomainsMode'} />
			</Box>

			<Box display='flex' width='100%' alignItems='flex-start'>
				<Controller
					name={nameField}
					control={control}
					render={({ field }) => (
						<FormControl style={{ flex: 1 }}>
							<TextField
								{...field}
								value={newDomain}
								onChange={e => setNewDomain(e.target.value)}
								variant='standard'
								onKeyDown={handleKeyPress}
								error={!!errors.virtualHostDomains}
								disabled={readMode}
								multiline
								// minRows={3}
								maxRows={15}
							/>
							<FormHelperText error={!!errors.virtualHostDomains} sx={{ ml: 0 }}>
								{errors.virtualHostDomains?.message}
							</FormHelperText>
						</FormControl>
					)}
				/>
				<Button
					variant='contained'
					onClick={addDomain}
					disabled={readMode}
					sx={{ flexShrink: 0, marginLeft: '10px', marginRight: '10px' }}
				>
					Add Domain
				</Button>
				<Button variant='outlined' component='label' sx={{ flexShrink: 0 }} disabled={readMode}>
					Upload Domains
					<input type='file' accept='.txt' style={{ display: 'none' }} onChange={handleFileUpload} />
				</Button>
				{selectedDomains.length > 0 && !readMode && (
					<Button
						variant='outlined'
						color='error'
						onClick={removeSelectedDomains}
						sx={{ flexShrink: 0, marginLeft: '10px' }}
					>
						Remove selected
					</Button>
				)}
				{!readMode && domainsWatch.length > 0 && (
					<Button variant='outlined' onClick={toggleSelectAll} sx={{ flexShrink: 0, marginX: '10px' }}>
						{selectedDomains.length === domainsWatch.length ? 'Deselect all' : 'Select all'}
					</Button>
				)}
			</Box>

			<Box
				mt={1}
				display='flex'
				flexDirection='column'
				gap={0.7}
				sx={{ overflowY: 'auto', padding: '1px 10px 10px 1px' }}
			>
				{domainsWatch.map((domain, index) => {
					const isVerified = verifyDomains?.results?.[index]?.validCertificate
					const errorText = verifyDomains?.results?.[index]?.error

					return (
						<Card key={index} sx={{ flexShrink: 0 }}>
							<CustomCardContent sx={{ display: 'flex', alignItems: 'center' }}>
								<Checkbox
									checked={selectedDomains.includes(index)}
									onChange={() => toggleSelectDomain(index)}
									disabled={readMode}
								/>
								{isVerified ? ' ✅' : ' ❌'}
								<Typography padding={1.2} sx={{ flexGrow: 1 }}>
									{domain}
								</Typography>
								<Typography padding={1.2} sx={{ flexGrow: 1 }} color='warning'>
									{errorText}
								</Typography>
								<IconButton onClick={() => removeDomain(index)} disabled={readMode}>
									<DeleteIcon color={readMode ? 'disabled' : 'primary'} />
								</IconButton>
							</CustomCardContent>
						</Card>
					)
				})}
			</Box>
		</Box>
	)
}
