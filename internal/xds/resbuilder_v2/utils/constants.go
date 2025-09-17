package utils

// TLS Configuration Types
const (
	SecretRefType     = "secretRef"
	AutoDiscoveryType = "autoDiscoveryType"
)

// Well-known filter names and type URLs
const (
	// Listener filter type URLs
	TLSInspectorTypeURL = "type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector"
)

// Common network ports
const (
	HTTPSPort = 443
)
