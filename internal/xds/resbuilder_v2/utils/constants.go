package utils

// TypeURL constants for Envoy filters and extensions
const (
	// HTTP Filters
	TypeURLOAuth2 = "type.googleapis.com/envoy.extensions.filters.http.oauth2.v3.OAuth2"
	TypeURLRouter = "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
	TypeURLCORS   = "type.googleapis.com/envoy.extensions.filters.http.cors.v3.Cors"

	// Network Filters
	TypeURLTCPProxy = "type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy"

	// Listener Filters
	TLSInspectorTypeURL = "type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector"
)

// Common port constants
const (
	HTTPSPort        = 443
	HTTPPort         = 80
	DefaultProxyPort = 8080
)

// Worker pool configuration
const (
	DefaultWorkerPoolSize = 4
)

// TLS configuration types
const (
	SecretRefType     = "secretRef"
	AutoDiscoveryType = "autoDiscovery"
)
