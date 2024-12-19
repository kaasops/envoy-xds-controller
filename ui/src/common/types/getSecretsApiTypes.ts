export interface ISecretsResponse {
	secrets: Secret[]
}

export interface Secret {
	name: string
	Type: Type
}

export interface Type {
	TlsCertificate: string | null
}

export interface ICertificatesResponse {
	name: string
	namespace: string
	type: string
	certs: ICertificate[]
}

export interface ICertificate {
	serialNumber: string
	subject: string
	notBefore: string
	notAfter: string
	issuer: string
	raw: string
	dnsNames: string[] | null
}
