package filter_chains

import (
	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

// Params holds the parameters for building filter chains
type Params struct {
	// VSName is the name of the virtual service
	VSName string

	// UseRemoteAddress determines if the original client address should be used
	UseRemoteAddress bool

	// XFFNumTrustedHops is the number of trusted hops in the XFF header
	XFFNumTrustedHops *uint32

	// RouteConfigName is the name of the route configuration
	RouteConfigName string

	// StatPrefix is the prefix for statistics
	StatPrefix string

	// HTTPFilters is the list of HTTP filters
	HTTPFilters []*hcmv3.HttpFilter

	// UpgradeConfigs is the list of protocol upgrade configurations
	UpgradeConfigs []*hcmv3.HttpConnectionManager_UpgradeConfig

	// AccessLogs is the list of access log configurations
	AccessLogs []*accesslogv3.AccessLog

	// Domains is the list of domain names
	Domains []string

	// DownstreamTLSContext is the TLS context for downstream connections
	DownstreamTLSContext *tlsv3.DownstreamTlsContext

	// SecretNameToDomains maps secret names to domains
	SecretNameToDomains map[helpers.NamespacedName][]string

	// IsTLS indicates if TLS is enabled
	IsTLS bool

	// Tracing is the tracing configuration
	Tracing *hcmv3.HttpConnectionManager_Tracing
}