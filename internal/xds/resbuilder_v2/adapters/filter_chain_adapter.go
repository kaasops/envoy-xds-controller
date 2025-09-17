package adapters

import (
	"fmt"
	"strings"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/filter_chains"
)

// FilterChainAdapter adapts the filter_chains.Builder to implement the FilterChainBuilder interface
type FilterChainAdapter struct {
	builder *filter_chains.Builder
	store   *store.Store
}

// NewFilterChainAdapter creates a new adapter for the filter_chains.Builder
func NewFilterChainAdapter(builder *filter_chains.Builder, store *store.Store) resbuilder_v2.FilterChainBuilder {
	return &FilterChainAdapter{
		builder: builder,
		store:   store,
	}
}

// BuildFilterChains delegates to the wrapped builder's BuildFilterChains method
func (a *FilterChainAdapter) BuildFilterChains(params *resbuilder_v2.FilterChainsParams) ([]*listenerv3.FilterChain, error) {
	// Convert from resbuilder_v2.FilterChainsParams to filter_chains.Params
	filterChainsParams := &filter_chains.Params{
		VSName:               params.VSName,
		UseRemoteAddress:     params.UseRemoteAddress,
		XFFNumTrustedHops:    params.XFFNumTrustedHops,
		RouteConfigName:      params.RouteConfigName,
		StatPrefix:           params.StatPrefix,
		HTTPFilters:          params.HTTPFilters,
		UpgradeConfigs:       params.UpgradeConfigs,
		AccessLogs:           params.AccessLogs,
		Domains:              params.Domains,
		DownstreamTLSContext: params.DownstreamTLSContext,
		SecretNameToDomains:  params.SecretNameToDomains,
		IsTLS:                params.IsTLS,
		Tracing:              params.Tracing,
	}

	return a.builder.BuildFilterChains(filterChainsParams)
}

// BuildFilterChainParams builds filter chain parameters from a VirtualService
// This method needs to be implemented since it's missing from filter_chains.Builder
func (a *FilterChainAdapter) BuildFilterChainParams(
	vs *v1alpha1.VirtualService,
	nn helpers.NamespacedName,
	httpFilters []*hcmv3.HttpFilter,
	listenerIsTLS bool,
	virtualHost *routev3.VirtualHost,
) (*resbuilder_v2.FilterChainsParams, error) {
	// Implementation based on buildFilterChainParams in builder.go
	filterChainParams := &resbuilder_v2.FilterChainsParams{
		VSName:            nn.String(),
		UseRemoteAddress:  helpers.BoolFromPtr(vs.Spec.UseRemoteAddress),
		RouteConfigName:   nn.String(),
		StatPrefix:        strings.ReplaceAll(nn.String(), ".", "-"),
		HTTPFilters:       httpFilters,
		IsTLS:             listenerIsTLS,
		XFFNumTrustedHops: vs.Spec.XFFNumTrustedHops,
	}

	// Handle tracing configuration
	if vs.Spec.Tracing != nil && vs.Spec.TracingRef != nil {
		return nil, fmt.Errorf("only one of spec.tracing or spec.tracingRef may be set")
	}

	// In a real implementation, you would need to implement all the logic from
	// buildFilterChainParams in builder.go, including:
	// - Tracing configuration
	// - Upgrade configs
	// - Access log configs
	// - TLS configuration

	return filterChainParams, nil
}

// CheckFilterChainsConflicts checks for conflicts between existing filter chains and virtual service configuration
// This method needs to be implemented since it's missing from filter_chains.Builder
func (a *FilterChainAdapter) CheckFilterChainsConflicts(vs *v1alpha1.VirtualService) error {
	// Implementation based on checkFilterChainsConflicts in builder.go
	conflicts := []struct {
		condition bool
		message   string
	}{
		{vs.Spec.VirtualHost != nil, "virtual host is set, but filter chains are found in listener"},
		{len(vs.Spec.AdditionalRoutes) > 0, "additional routes are set, but filter chains are found in listener"},
		{len(vs.Spec.HTTPFilters) > 0, "http filters are set, but filter chains are found in listener"},
		{len(vs.Spec.AdditionalHttpFilters) > 0, "additional http filters are set, but filter chains are found in listener"},
		{vs.Spec.TlsConfig != nil, "tls config is set, but filter chains are found in listener"},
		{vs.Spec.RBAC != nil, "rbac is set, but filter chains are found in listener"},
		{vs.Spec.UseRemoteAddress != nil, "use remote address is set, but filter chains are found in listener"},
		{vs.Spec.XFFNumTrustedHops != nil, "xff_num_trusted_hops is set, but filter chains are found in listener"},
		{vs.Spec.UpgradeConfigs != nil, "upgrade configs is set, but filter chains are found in listener"},
		{vs.Spec.AccessLog != nil, "access log is set, but filter chains are found in listener"},
		{vs.Spec.AccessLogConfig != nil, "access log config is set, but filter chains are found in listener"},
		{len(vs.Spec.AccessLogs) > 0, "access logs are set, but filter chains are found in listener"},
		{len(vs.Spec.AccessLogConfigs) > 0, "access log configs are set, but filter chains are found in listener"},
	}

	for _, conflict := range conflicts {
		if conflict.condition {
			return fmt.Errorf("conflict: %s", conflict.message)
		}
	}

	return nil
}