import { useId, useState } from 'react'
import {
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	FormControl,
	FormControlLabel,
	FormLabel,
	IconButton,
	Radio,
	RadioGroup,
	Stack,
	TextField,
	Tooltip
} from '@mui/material'
import SettingsOutlined from '@mui/icons-material/SettingsOutlined'
import { useConsoleTime } from './useConsoleTime'
import { isValidTimeZone } from './format'

function supportedZones() {
	const intl = Intl as typeof Intl & { supportedValuesOf?: (key: 'timeZone') => string[] }
	return intl.supportedValuesOf?.('timeZone') ?? []
}

function PreferencesDialog({ onClose }: { onClose: () => void }) {
	const { timeZone, setTimeZone } = useConsoleTime()
	const title = useId()
	const zonesId = useId()
	const [mode, setMode] = useState(timeZone === 'local' || timeZone === 'UTC' ? timeZone : 'iana')
	const [zoneName, setZoneName] = useState(mode === 'iana' ? timeZone : '')
	const [zones] = useState(supportedZones)
	const selected = mode === 'iana' ? zoneName.trim() : mode
	const valid = mode !== 'iana' || isValidTimeZone(selected)
	const invalidName = Boolean(zoneName.trim()) && !valid
	return (
		<Dialog open fullWidth maxWidth='xs' aria-labelledby={title} onClose={onClose}>
			<form
				onSubmit={e => {
					e.preventDefault()
					if (valid) {
						setTimeZone(selected)
						onClose()
					}
				}}
			>
				<DialogTitle id={title}>Preferences</DialogTitle>
				<DialogContent>
					<Stack spacing={2}>
						<FormControl component='fieldset'>
							<FormLabel component='legend'>Time zone</FormLabel>
							<RadioGroup value={mode} onChange={e => setMode(e.target.value)} aria-label='Time zone'>
								<FormControlLabel
									value='local'
									control={<Radio autoFocus={timeZone === 'local'} />}
									label='Browser local'
								/>
								<FormControlLabel
									value='UTC'
									control={<Radio autoFocus={timeZone === 'UTC'} />}
									label='UTC'
								/>
								<FormControlLabel
									value='iana'
									control={<Radio autoFocus={timeZone !== 'local' && timeZone !== 'UTC'} />}
									label='IANA time zone'
								/>
							</RadioGroup>
						</FormControl>
						{mode === 'iana' && (
							<>
								<TextField
									fullWidth
									label='Time zone name'
									size='small'
									value={zoneName}
									onChange={e => setZoneName(e.target.value)}
									error={invalidName}
									helperText={
										invalidName
											? 'Enter a valid IANA time zone name.'
											: 'For example, Europe/Berlin.'
									}
									inputProps={{ list: zonesId, autoComplete: 'off', spellCheck: false }}
								/>
								<datalist id={zonesId}>
									{zones.map(zone => (
										<option key={zone} value={zone} />
									))}
								</datalist>
							</>
						)}
					</Stack>
				</DialogContent>
				<DialogActions>
					<Button onClick={onClose}>Cancel</Button>
					<Button variant='contained' type='submit' disabled={!valid}>
						Save
					</Button>
				</DialogActions>
			</form>
		</Dialog>
	)
}

export default function PreferencesButton() {
	const [open, setOpen] = useState(false)
	return (
		<>
			<Tooltip title='Preferences'>
				<IconButton
					color='inherit'
					aria-label='Preferences'
					aria-haspopup='dialog'
					onClick={() => setOpen(true)}
				>
					<SettingsOutlined fontSize='small' />
				</IconButton>
			</Tooltip>
			{open && <PreferencesDialog onClose={() => setOpen(false)} />}
		</>
	)
}
