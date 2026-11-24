import { ConsoleTime } from '../../time/ConsoleTime'
import { useConsoleTime } from '../../time/useConsoleTime'
import { Stack } from '@mui/material'
import client from '../../../../api/axiosApiClient'
import type { NodeOverviewResponse } from '../../../../common/types/overviewApiTypes'
import ConfigResourceTable, { Fields, ResourceSelection } from '../ConfigResourceTable'
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
export default function Secrets() {
	const { formatTime } = useConsoleTime()
	const columns: Column<SecretRow>[] = [
		{ key: 'name', label: 'Secret', value: r => r.name },
		{ key: 'type', label: 'Type', value: r => r.type },
		{
			key: 'domains',
			label: 'DNS names',
			value: r => r.certificate?.dnsNames ?? [],
			render: r => {
				const names = r.certificate?.dnsNames ?? []
				return names.length > 2 ? (
					<ResourceSelection name={r.name}>{names.length} names</ResourceSelection>
				) : (
					names.join(', ')
				)
			}
		},
		{
			key: 'expiry',
			label: 'Not after',
			value: r => formatTime(r.certificate?.notAfter) || 'Unavailable',
			sortValue: r => Date.parse(r.certificate?.notAfter ?? '') || 0,
			render: r => <ConsoleTime value={r.certificate?.notAfter} />
		},
		{ key: 'status', label: 'Status', value: r => r.certificate?.status || 'Unavailable' }
	]
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
								['Not before', <ConsoleTime value={row.certificate.notBefore} />],
								['Not after', <ConsoleTime value={row.certificate.notAfter} />],
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
