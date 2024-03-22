export interface ISecretsResponse {
    secrets: Secret[];
}

export interface Secret {
    name: string;
    Type: Type;
}

export interface Type {
    TlsCertificate: string | null;
}
