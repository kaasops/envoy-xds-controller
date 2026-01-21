export interface NodeOverviewResponse {
	nodeId: string
	summary: OverviewSummary
	resourceVersions: ResourceVersions
	endpoints: EndpointInfo[]
	certificates: CertificateInfo[]
}

export interface ResourceVersions {
	listeners: string
	clusters: string
	routes: string
	secrets: string
}

export interface OverviewSummary {
	totalDomains: number
	totalEndpoints: number
	totalCertificates: number
	certificatesWarning: number
	certificatesCritical: number
	certificatesExpired: number
}

export interface EndpointInfo {
	domain: string
	port: number
	protocol: 'HTTP' | 'HTTPS' | 'TCP'
	listenerName: string
	routeConfigName?: string
	certificate?: CertificateBrief
}

export interface CertificateBrief {
	name: string
	expiresAt: string
	daysUntilExpiry: number
	status: CertificateStatus
}

export interface CertificateInfo {
	name: string
	namespace: string
	subject: string
	issuer: string
	notBefore: string
	notAfter: string
	daysUntilExpiry: number
	status: CertificateStatus
	dnsNames: string[]
	usedByDomains: string[]
}

export type CertificateStatus = 'ok' | 'warning' | 'critical' | 'expired'
