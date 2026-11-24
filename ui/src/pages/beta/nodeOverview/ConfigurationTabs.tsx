import type { ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link as RouterLink } from 'react-router-dom'
import { Alert, Box, Chip, LinearProgress, Link, Stack, Typography } from '@mui/material'
import type { CertificateInfo, EndpointInfo, NodeOverviewResponse } from '../../../common/types/overviewApiTypes'
import { apiError, getSnapshotDiff, manualQueryOptions, nodeOverviewPath, typeLabel } from '../api'
import { Fields } from '../console/ConfigResourceTable'
import ConfigurationTable from './ConfigurationTable'
import { certificateId, resourceHashesKey } from './configurationData'

export type ConfigurationTab = 'endpoints' | 'certificates' | 'hashes'

interface Props {
	nodeId: string
	tab: ConfigurationTab
	overview?: NodeOverviewResponse
	selected: string
}

function FieldLink({
	nodeId,
	tab,
	id,
	children
}: {
	nodeId: string
	tab: ConfigurationTab
	id: string
	children: ReactNode
}) {
	return (
		<Link component={RouterLink} to={nodeOverviewPath(nodeId, tab, id)}>
			{children}
		</Link>
	)
}

function SectionHeading({ title, description }: { title: string; description: string }) {
	return (
		<Box sx={{ px: 2, py: 1.5 }}>
			<Typography component='h2' variant='subtitle1' fontWeight={600}>
				{title}
			</Typography>
			<Typography variant='body2' color='text.secondary'>
				{description}
			</Typography>
		</Box>
	)
}

function Detail({ title, children }: { title: string; children: ReactNode }) {
	return (
		<Box
			component='section'
			data-node-detail
			tabIndex={-1}
			sx={{ p: 2, bgcolor: 'action.hover', scrollMarginTop: '5rem', overflowWrap: 'anywhere' }}
		>
			<Typography component='h3' variant='subtitle1' fontWeight={600} gutterBottom>
				{title}
			</Typography>
			{children}
		</Box>
	)
}

function endpointCertificate(ep: EndpointInfo, byName: Map<string, CertificateInfo[]>) {
	if (!ep.certificate) return
	const candidates = byName.get(ep.certificate.name) ?? []
	if (candidates.length === 1) return candidates[0]
	// The endpoint carries the secret's basename, but two namespaces can contain the same name.
	const matches = candidates.filter(
		c => c.notAfter === ep.certificate?.expiresAt && c.usedByDomains?.includes(ep.domain)
	)
	return matches.length === 1 ? matches[0] : undefined
}

function EndpointTable({ nodeId, overview }: Required<Pick<Props, 'nodeId' | 'overview'>>) {
	const byName = new Map<string, CertificateInfo[]>()
	for (const cert of overview.certificates ?? []) {
		const matching = byName.get(cert.name)
		if (matching) matching.push(cert)
		else byName.set(cert.name, [cert])
	}
	const rows = (overview.endpoints ?? []).map((ep, index) => ({
		...ep,
		index,
		cert: endpointCertificate(ep, byName)
	}))
	rows.sort(
		(a, b) => a.domain.localeCompare(b.domain) || a.port - b.port || a.listenerName.localeCompare(b.listenerName)
	)
	return (
		<>
			<SectionHeading title='Listener endpoints' description='Incoming traffic by domain and port' />
			<ConfigurationTable
				label='Listener endpoints'
				rows={rows}
				rowId={ep => String(ep.index)}
				columns={[
					{ label: 'Domain', value: ep => ep.domain },
					{ label: 'Port', value: ep => (ep.port ? String(ep.port) : 'Unavailable') },
					{ label: 'Protocol', value: ep => ep.protocol },
					{
						label: 'Listener',
						value: ep => ep.listenerName,
						render: ep => (
							<FieldLink nodeId={nodeId} tab='hashes' id={'LDS/' + ep.listenerName}>
								{ep.listenerName}
							</FieldLink>
						)
					},
					{
						label: 'Route configuration',
						value: ep => ep.routeConfigName || 'None',
						render: ep =>
							ep.routeConfigName ? (
								<FieldLink nodeId={nodeId} tab='hashes' id={'RDS/' + ep.routeConfigName}>
									{ep.routeConfigName}
								</FieldLink>
							) : (
								'None'
							)
					},
					{
						label: 'Certificate',
						value: ep => (ep.cert ? certificateId(ep.cert) : ep.certificate?.name || 'Unavailable'),
						render: ep =>
							ep.cert ? (
								<FieldLink nodeId={nodeId} tab='certificates' id={certificateId(ep.cert)}>
									{certificateId(ep.cert)}
								</FieldLink>
							) : (
								ep.certificate?.name || (ep.protocol === 'HTTP' ? 'None (HTTP)' : 'Unavailable')
							)
					}
				]}
			/>
		</>
	)
}

function validity(cert: CertificateInfo): { label: string; color: 'default' | 'error' | 'success' | 'warning' } {
	const starts = Date.parse(cert.notBefore)
	const expires = Date.parse(cert.notAfter)
	if (!Number.isFinite(starts) || !Number.isFinite(expires) || starts > expires) {
		return { label: 'Unavailable', color: 'default' }
	}
	const now = Date.now()
	if (now < starts) return { label: 'Not yet valid', color: 'error' }
	if (now >= expires) return { label: 'Expired', color: 'error' }
	if (cert.status === 'ok') return { label: 'Valid', color: 'success' }
	const label =
		cert.daysUntilExpiry <= 0
			? 'Expires within 24 hours'
			: 'Expires in ' + cert.daysUntilExpiry + (cert.daysUntilExpiry === 1 ? ' day' : ' days')
	return { label, color: cert.status === 'warning' ? 'warning' : 'error' }
}

function CertificateTable({ nodeId, overview, selected }: Required<Pick<Props, 'nodeId' | 'overview' | 'selected'>>) {
	const certs = [...(overview.certificates ?? [])].sort(
		(a, b) => (a.notAfter || '').localeCompare(b.notAfter || '') || certificateId(a).localeCompare(certificateId(b))
	)
	const selectedCert = certs.find(c => certificateId(c) === selected)
	return (
		<>
			<SectionHeading title='Certificates' description='Validity and domains in the served configuration' />
			{selectedCert && (
				<Detail title={certificateId(selectedCert)}>
					<Fields
						entries={[
							['DNS names (SAN)', selectedCert.dnsNames?.join(', ') || 'None'],
							['Used by domains', [...new Set(selectedCert.usedByDomains)].join(', ') || 'None'],
							['Subject', selectedCert.subject],
							['Issuer', selectedCert.issuer],
							['Not before', selectedCert.notBefore || 'Unavailable'],
							['Not after', selectedCert.notAfter || 'Unavailable'],
							[
								'SDS resource',
								<FieldLink nodeId={nodeId} tab='hashes' id={'SDS/' + certificateId(selectedCert)}>
									{certificateId(selectedCert)}
								</FieldLink>
							]
						]}
					/>
				</Detail>
			)}
			{selected && !selectedCert && (
				<Alert severity='info'>The selected certificate is no longer in the served configuration.</Alert>
			)}
			<ConfigurationTable
				label='Certificates'
				rows={certs}
				rowId={certificateId}
				selected={selected}
				columns={[
					{
						label: 'Secret',
						value: certificateId,
						render: c => (
							<FieldLink nodeId={nodeId} tab='certificates' id={certificateId(c)}>
								{certificateId(c)}
							</FieldLink>
						)
					},
					{ label: 'Used by domains', value: c => [...new Set(c.usedByDomains)].join(', ') || 'None' },
					{ label: 'Not after (UTC)', value: c => c.notAfter || 'Unavailable' },
					{
						label: 'Validity',
						value: c => validity(c).label,
						render: c => <Chip data-status size='small' variant='outlined' {...validity(c)} />
					}
				]}
			/>
		</>
	)
}

const typeCodes: Record<string, string> = {
	'type.googleapis.com/envoy.config.listener.v3.Listener': 'LDS',
	'type.googleapis.com/envoy.config.route.v3.RouteConfiguration': 'RDS',
	'type.googleapis.com/envoy.config.cluster.v3.Cluster': 'CDS',
	'type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret': 'SDS'
}

function HashTable({ nodeId, selected }: Pick<Props, 'nodeId' | 'selected'>) {
	const query = useQuery({
		...manualQueryOptions,
		refetchOnMount: 'always',
		retryOnMount: true,
		queryKey: resourceHashesKey(nodeId),
		// Both sides read the served cache. This endpoint checks access to the requested node.
		queryFn: ({ signal }) => getSnapshotDiff(nodeId, 'served', 'served', signal)
	})
	const data = query.isFetchedAfterMount && !query.error ? query.data : undefined
	const rows = (data?.resources ?? [])
		.filter(r => r.toSha256)
		.map(r => ({
			name: r.name,
			hash: r.toSha256 === 'error' ? 'Unavailable' : r.toSha256,
			type: typeCodes[r.typeUrl] || typeLabel(r.typeUrl),
			id: (typeCodes[r.typeUrl] || r.typeUrl) + '/' + r.name
		}))
	const selectedRow = rows.find(r => r.id === selected)
	return (
		<>
			<SectionHeading title='Resource hashes' description='Per-resource SHA-256 hashes of the served snapshot' />
			{query.isFetching && <LinearProgress aria-label='Loading resource hashes' />}
			{query.error && <Alert severity='error'>Could not load resource hashes. {apiError(query.error)}</Alert>}
			{data && (
				<>
					{Object.keys(data.to.versions ?? {}).length > 0 && (
						<Stack direction='row' alignItems='center' flexWrap='wrap' gap={1} sx={{ px: 2, pb: 1.5 }}>
							<Typography variant='body2'>Served versions</Typography>
							{Object.entries(data.to.versions).map(([type, version]) => (
								<Chip
									key={type}
									label={<code>{(typeCodes[type] || typeLabel(type)) + ' ' + version}</code>}
									size='small'
									variant='outlined'
								/>
							))}
						</Stack>
					)}
					{selectedRow && (
						<Detail title={selectedRow.name}>
							<Fields
								entries={[
									['Type', selectedRow.type],
									['Full resource hash', <code>{selectedRow.hash}</code>]
								]}
							/>
						</Detail>
					)}
					{selected && !selectedRow && (
						<Alert severity='info'>The selected resource is no longer in the served snapshot.</Alert>
					)}
					<ConfigurationTable
						label='Resource hashes'
						rows={rows}
						rowId={r => r.id}
						selected={selected}
						columns={[
							{ label: 'Type', value: r => r.type },
							{
								label: 'Resource',
								value: r => r.name,
								render: r => (
									<FieldLink nodeId={nodeId} tab='hashes' id={r.id}>
										{r.name}
									</FieldLink>
								)
							},
							{
								label: 'Resource hash',
								value: r => r.hash,
								render: r => (
									<code>
										{r.hash.length > 24 ? r.hash.slice(0, 12) + '…' + r.hash.slice(-8) : r.hash}
									</code>
								)
							}
						]}
					/>
				</>
			)}
		</>
	)
}

export default function ConfigurationTabs({ nodeId, tab, overview, selected }: Props) {
	switch (tab) {
		case 'endpoints':
			return overview ? <EndpointTable nodeId={nodeId} overview={overview} /> : null
		case 'certificates':
			return overview ? <CertificateTable nodeId={nodeId} overview={overview} selected={selected} /> : null
		case 'hashes':
			return <HashTable key={nodeId} nodeId={nodeId} selected={selected} />
	}
}
