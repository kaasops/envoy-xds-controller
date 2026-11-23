import type { CertificateInfo } from '../../../common/types/overviewApiTypes'

export function certificateId(cert: Pick<CertificateInfo, 'namespace' | 'name'>) {
	return cert.namespace ? cert.namespace + '/' + cert.name : cert.name
}

export const resourceHashesKey = (nodeId: string) => ['beta', 'node-overview', nodeId, 'hashes'] as const
