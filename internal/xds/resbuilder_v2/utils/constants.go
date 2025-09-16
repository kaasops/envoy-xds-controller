package utils

// TLS Configuration Types
const (
	SecretRefType     = "secretRef"
	AutoDiscoveryType = "autoDiscovery"
)

// Well-known filter names and type URLs
const (
	// HTTP filter type URLs
	RouterFilterTypeURL = "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
	OAuth2FilterTypeURL = "type.googleapis.com/envoy.extensions.filters.http.oauth2.v3.OAuth2"
	RBACFilterTypeURL   = "type.googleapis.com/envoy.extensions.filters.http.rbac.v3.RBAC"
	
	// Custom filter names
	RBACFilterName = "exc.filters.http.rbac"
	
	// Listener filter type URLs
	TLSInspectorTypeURL = "type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector"
)

// Field names commonly used in configuration searching
const (
	// Cluster field names
	ClusterFieldName          = "cluster"
	ClusterNameFieldName      = "cluster_name"
	CollectorClusterFieldName = "collector_cluster"
	TokenClusterFieldName     = "token_cluster"
	AuthClusterFieldName      = "authorization_cluster"
	
	// SDS field names
	SDSNameFieldName = "name"
	
	// Tracing field names
	TracingClusterFieldName = "cluster_name"
	ZipkinClusterFieldName  = "collector_cluster"
)

// Common network ports
const (
	HTTPSPort = 443
	HTTPPort  = 80
)

// Common route paths
const (
	RootPath   = "/"
	RootPrefix = "/"
)

// Fallback virtual host configuration
const (
	FallbackVirtualHostName   = "421vh"
	FallbackVirtualHostDomain = "*"
	FallbackStatusCode        = 421
)

// Default cache sizes for optimization
const (
	DefaultHTTPFiltersCacheSize = 1000
	DefaultClusterCacheSize     = 500
	DefaultSlicePoolCapacity    = 8
	DefaultClusterPoolCapacity  = 4
)

// Kubernetes secret field names
const (
	TLSCertificateKey = "tls.crt"
	TLSPrivateKeyKey  = "tls.key"
)

// Error messages
const (
	ErrMultipleRouterFilters     = "multiple router HTTP filters found"
	ErrMultipleRootRoutes        = "multiple root routes found"
	ErrVirtualHostEmpty          = "virtual host is empty"
	ErrListenerNotFound          = "listener not found"
	ErrClusterNotFound           = "cluster not found"
	ErrSecretNotFound            = "secret not found"
	ErrInvalidRBACAction         = "invalid RBAC action"
	ErrRBACPoliciesEmpty         = "RBAC policies is empty"
	ErrRBACActionEmpty           = "RBAC action is empty"
	ErrDuplicateDomain           = "duplicate domain found"
	ErrTLSConfigConflict         = "multiple TLS configuration types specified"
	ErrTLSConfigEmpty            = "no TLS configuration specified"
	ErrSecretRefNameEmpty        = "secretRef name is empty"
	ErrSecretRefNamespaceEmpty   = "secretRef namespace is required"
	ErrInvalidSecretType         = "unsupported secret type"
	ErrSecretDataNil             = "secret data is nil"
	ErrCertificateDataEmpty      = "certificate data is empty"
	ErrPrivateKeyDataEmpty       = "private key data is empty"
)

// HTTP Connection Manager configuration
const (
	HTTPConnectionManagerCodecAuto = "AUTO"
	DefaultStatPrefixSeparator     = "-"
)

// Access log configuration
const (
	AccessLogConflictError = "can't use accessLog, accessLogConfig, accessLogs and accessLogConfigs at the same time"
)

// Route configuration
const (
	MaxRootRoutes = 1
)

// Resource validation
const (
	MinDomainsForUniquenessCheck = 2
	MinRoutesForReordering       = 2
)