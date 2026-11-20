import { Stack } from '@mui/material'
import client from '../../../../api/axiosApiClient'
import type { NodeOverviewResponse } from '../../../../common/types/overviewApiTypes'
import ConfigResourceTable, { Fields } from '../ConfigResourceTable'
import type { Column } from '../SectionTable'

async function loadSecrets(node: string, signal: AbortSignal) {
	const options = { params: { node_id: node }, signal }
	const [inventory, overview] = await Promise.all([
		client.get<{ secrets: { name: string; Type?: Record<string, unknown> }[] | null }>('/secrets', options),
		client.get<NodeOverviewResponse>('/overview', options)
	])
	const certificates = new Map(
		(overview.data.certificates ?? []).map(cert => [
			cert.namespace ? cert.namespace + '/' + cert.name : cert.name,
			cert
		])
	)
	return (inventory.data.secrets ?? []).map(entry => ({
		name: entry.name,
		type: Object.keys(entry.Type ?? {})[0] || 'Unknown',
		certificate: certificates.get(entry.name)
	}))
}

type SecretRow = Awaited<ReturnType<typeof loadSecrets>>[number]
const columns: Column<SecretRow>[] = [
	{ key: 'name', label: 'Secret', value: r => r.name },
	{ key: 'type', label: 'Type', value: r => r.type },
	{ key: 'domains', label: 'DNS names', value: r => r.certificate?.dnsNames ?? [] },
	{ key: 'expiry', label: 'Not after', value: r => r.certificate?.notAfter || 'Unavailable' },
	{ key: 'status', label: 'Certificate status', value: r => r.certificate?.status || 'Unavailable' }
]
export default function Secrets() {
	return (
		<ConfigResourceTable
			title='Secrets'
			load={loadSecrets}
			columns={columns}
			detail={row => (
				<Stack spacing={1.5}>
					{row.certificate ? (
						<Fields
							entries={[
								['Type', row.type],
								['Subject', row.certificate.subject],
								['Issuer', row.certificate.issuer],
								['Not before', row.certificate.notBefore],
								['Not after', row.certificate.notAfter],
								['DNS names', (row.certificate.dnsNames ?? []).join(', ') || 'None'],
								['Used by domains', (row.certificate.usedByDomains ?? []).join(', ') || 'None']
							]}
						/>
					) : (
						<Fields
							entries={[
								['Type', row.type],
								['Certificate metadata', 'Unavailable']
							]}
						/>
					)}
				</Stack>
			)}
		/>
	)
}
