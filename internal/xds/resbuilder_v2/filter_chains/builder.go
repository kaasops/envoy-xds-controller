package filter_chains

import (
	"fmt"
	"slices"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Builder implements filter chain building functionality
type Builder struct {
	store *store.Store
}

// NewBuilder creates a new filter chain builder
func NewBuilder(store *store.Store) *Builder {
	return &Builder{
		store: store,
	}
}

// BuildFilterChains builds multiple filter chains from the provided parameters
func (b *Builder) BuildFilterChains(params *Params) ([]*listenerv3.FilterChain, error) {
	var filterChains []*listenerv3.FilterChain

	if len(params.SecretNameToDomains) > 0 {
		for secretName, domains := range params.SecretNameToDomains {
			// Create a copy of params to avoid modifying the original
			paramsCopy := *params
			paramsCopy.Domains = domains
			paramsCopy.DownstreamTLSContext = &tlsv3.DownstreamTlsContext{
				CommonTlsContext: &tlsv3.CommonTlsContext{
					TlsCertificateSdsSecretConfigs: []*tlsv3.SdsSecretConfig{{
						Name: secretName.String(),
						SdsConfig: &corev3.ConfigSource{
							ConfigSourceSpecifier: &corev3.ConfigSource_Ads{
								Ads: &corev3.AggregatedConfigSource{},
							},
							ResourceApiVersion: corev3.ApiVersion_V3,
						},
					}},
					AlpnProtocols: []string{"h2", "http/1.1"},
				},
			}
			fc, err := b.buildFilterChain(&paramsCopy)
			if err != nil {
				return nil, fmt.Errorf("failed to build filter chain for domain %v: %w", domains, err)
			}
			filterChains = append(filterChains, fc)
		}
		return filterChains, nil
	}

	fc, err := b.buildFilterChain(params)
	if err != nil {
		return nil, fmt.Errorf("failed to build filter chain: %w", err)
	}
	filterChains = append(filterChains, fc)
	return filterChains, nil
}

// buildFilterChain builds a single filter chain from the provided parameters
func (b *Builder) buildFilterChain(params *Params) (*listenerv3.FilterChain, error) {
	httpConnectionManager := &hcmv3.HttpConnectionManager{
		CodecType:  hcmv3.HttpConnectionManager_AUTO,
		StatPrefix: params.StatPrefix,
		RouteSpecifier: &hcmv3.HttpConnectionManager_Rds{
			Rds: &hcmv3.Rds{
				ConfigSource: &corev3.ConfigSource{
					ResourceApiVersion:    corev3.ApiVersion_V3,
					ConfigSourceSpecifier: &corev3.ConfigSource_Ads{},
				},
				RouteConfigName: params.RouteConfigName,
			},
		},
		UseRemoteAddress: &wrapperspb.BoolValue{Value: params.UseRemoteAddress},
		UpgradeConfigs:   params.UpgradeConfigs,
		HttpFilters:      params.HTTPFilters,
	}

	// Add tracing configuration if provided
	if params.Tracing != nil {
		httpConnectionManager.Tracing = params.Tracing
	}

	// Add XFF trusted hops if provided
	if params.XFFNumTrustedHops != nil {
		httpConnectionManager.XffNumTrustedHops = *params.XFFNumTrustedHops
	}

	// Add access logs if provided
	if len(params.AccessLogs) > 0 {
		httpConnectionManager.AccessLog = append(httpConnectionManager.AccessLog, params.AccessLogs...)
	}

	// Validate the HTTP connection manager
	if err := httpConnectionManager.ValidateAll(); err != nil {
		return nil, fmt.Errorf("failed to validate HTTP connection manager: %w", err)
	}

	// Marshal the HTTP connection manager to Any
	pbst, err := anypb.New(httpConnectionManager)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal HTTP connection manager to Any: %w", err)
	}

	// Create the filter chain
	fc := &listenerv3.FilterChain{}
	fc.Filters = []*listenerv3.Filter{{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &listenerv3.Filter_TypedConfig{
			TypedConfig: pbst,
		},
	}}

	// Add server names to filter chain match if needed
	if params.IsTLS && len(params.Domains) > 0 && !slices.Contains(params.Domains, "*") {
		fc.FilterChainMatch = &listenerv3.FilterChainMatch{
			ServerNames: params.Domains,
		}
	}

	// Add TLS context if provided
	if params.DownstreamTLSContext != nil {
		scfg, err := anypb.New(params.DownstreamTLSContext)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal downstream TLS context to Any: %w", err)
		}
		fc.TransportSocket = &corev3.TransportSocket{
			Name: "envoy.transport_sockets.tls",
			ConfigType: &corev3.TransportSocket_TypedConfig{
				TypedConfig: scfg,
			},
		}
	}

	// Set filter chain name
	fc.Name = params.VSName

	// Validate the filter chain
	if err := fc.ValidateAll(); err != nil {
		return nil, fmt.Errorf("failed to validate filter chain: %w", err)
	}

	return fc, nil
}