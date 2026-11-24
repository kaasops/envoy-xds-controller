import { Box, Link, Paper, Popper, Typography } from '@mui/material'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { Link as RouterLink } from 'react-router-dom'

interface Guidance {
	meaning: string
	csds?: string
	next?: string
	href: string
}

const envoyDocs = 'https://www.envoyproxy.io/docs/envoy/latest/'
const protocol = envoyDocs + 'api-docs/xds_protocol'
const csds = envoyDocs + 'api-v3/service/status/v3/csds.proto#enum-service-status-v3-configstatus'

const guidance = {
	ACK: {
		meaning: 'Envoy acknowledged the latest responses observed on this stream.',
		csds: 'SYNCED',
		href: protocol + '#ack-and-nack-semantics-summary'
	},
	NACK: {
		meaning: 'Envoy rejected at least one resource in an update and reported error_detail.',
		csds: 'ERROR',
		next: 'Read the stream error, then check error_state and active resources in the Envoy config_dump.',
		href: protocol + '#ack-and-nack-semantics-summary'
	},
	'Awaiting ACK': {
		meaning: 'The controller sent a response and is waiting for Envoy to ACK or NACK it.',
		csds: 'STALE',
		next: 'If the wait persists, check connectivity and whether Envoy is processing updates.',
		href: csds
	},
	Subscribed: {
		meaning: 'Envoy requested this resource type. The controller has not sent a response for it on this stream.',
		next: 'After a Delta reconnect, Envoy may hold current versions. Check active resources in config_dump.',
		href: protocol + '#incremental-xds'
	},
	Unknown: {
		meaning: 'The controller has too little information to determine the current status.',
		csds: 'UNKNOWN',
		next: 'Open the stream details to see which resource types and responses the controller observed.',
		href: csds
	},
	'No streams': {
		meaning: 'No Envoy for this node has an open xDS connection to this controller.',
		next: 'Check the node ID, the xDS endpoint and which controller replica Envoy reached.',
		href: protocol
	},
	'No configuration on this controller': {
		meaning: 'An Envoy connected with this node ID, but this controller has no snapshot for it.',
		next: 'Check the node ID in the Envoy bootstrap and the nodes targeted by the virtual services.',
		href: protocol
	},
	Paused: {
		meaning: 'As with kubectl rollout pause, the controller accepts changes and holds them in Pending.',
		next: 'Compare Served with Pending, then resume updates to send the reviewed snapshot to Envoy.',
		href: 'https://kubernetes.io/docs/reference/kubectl/generated/kubectl_rollout/kubectl_rollout_pause/'
	}
} satisfies Record<string, Guidance>

export type HelpStatus = keyof typeof guidance

export default function StatusHelp({
	status,
	to,
	label,
	children
}: {
	status: HelpStatus
	to?: string
	label?: string
	children: ReactNode
}) {
	const help: Guidance = guidance[status]
	const id = useId()
	const container = useRef<HTMLSpanElement>(null)
	const trigger = useRef<HTMLElement | null>(null)
	const closeTimer = useRef<ReturnType<typeof setTimeout>>()
	const [open, setOpen] = useState(false)

	function show() {
		clearTimeout(closeTimer.current)
		setOpen(true)
	}

	function leave() {
		closeTimer.current = setTimeout(() => {
			if (!container.current?.contains(document.activeElement)) setOpen(false)
		}, 150)
	}

	useEffect(() => () => clearTimeout(closeTimer.current), [])
	useEffect(() => {
		if (!open) return
		function escape(event: KeyboardEvent) {
			if (event.key !== 'Escape') return
			event.stopPropagation()
			if (container.current?.contains(document.activeElement)) trigger.current?.focus()
			setOpen(false)
		}
		document.addEventListener('keydown', escape, true)
		return () => document.removeEventListener('keydown', escape, true)
	}, [open])

	const triggerProps = {
		ref: (el: HTMLElement | null) => {
			trigger.current = el
		},
		'aria-label': label,
		'aria-describedby': open ? id : undefined,
		onFocus: show
	}
	return (
		<Box
			component='span'
			ref={container}
			sx={{ display: 'inline-flex', maxWidth: '100%' }}
			onMouseEnter={show}
			onMouseLeave={leave}
			onBlur={e => {
				if (!e.currentTarget.contains(e.relatedTarget)) setOpen(false)
			}}
		>
			{to ? (
				<Link component={RouterLink} to={to} underline='hover' {...triggerProps}>
					{children}
				</Link>
			) : (
				<Box
					component='span'
					tabIndex={0}
					{...triggerProps}
					sx={{
						cursor: 'help',
						maxWidth: '100%',
						borderRadius: 1,
						'&:focus-visible': { outline: '2px solid', outlineColor: 'primary.main' }
					}}
				>
					{children}
				</Box>
			)}
			<Popper
				open={open}
				anchorEl={trigger.current}
				disablePortal
				placement='bottom-start'
				popperOptions={{ strategy: 'fixed' }}
				sx={{ zIndex: theme => theme.zIndex.tooltip }}
			>
				<Paper
					role='dialog'
					aria-label={status + ' status help'}
					elevation={4}
					sx={{
						mt: 0.5,
						p: 1.5,
						width: 310,
						maxWidth: 'calc(100vw - 20px)',
						whiteSpace: 'normal',
						overflowWrap: 'anywhere'
					}}
				>
					<Box id={id}>
						<Typography variant='body2'>{help.meaning}</Typography>
						{help.csds && (
							<Typography variant='caption' component='p' sx={{ my: 0.5 }}>
								CSDS ConfigStatus: <code>{help.csds}</code>
							</Typography>
						)}
						{help.next && (
							<Typography variant='body2' sx={{ my: 0.5 }}>
								{help.next}
							</Typography>
						)}
					</Box>
					<Link href={help.href} target='_blank' rel='noreferrer' variant='body2'>
						Learn more
					</Link>
				</Paper>
			</Popper>
		</Box>
	)
}
