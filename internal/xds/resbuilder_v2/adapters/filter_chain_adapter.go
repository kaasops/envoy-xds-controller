package adapters

import (
	"fmt"
	"strings"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/filter_chains"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/interfaces"
)

// FilterChainAdapter adapts the filter_chains.Builder to implement the FilterChainBuilder interface
type FilterChainAdapter struct {
	builder *filter_chains.Builder
	store   *store.Store
}

// NewFilterChainAdapter creates a new adapter for the filter_chains.Builder
func NewFilterChainAdapter(builder *filter_chains.Builder, store *store.Store) interfaces.FilterChainBuilder {
	return &FilterChainAdapter{
		builder: builder,
		store:   store,
	}
}

// BuildFilterChains delegates to the wrapped builder's BuildFilterChains method
func (a *FilterChainAdapter) BuildFilterChains(params *interfaces.FilterChainsParams) ([]*listenerv3.FilterChain, error) {
	// Convert from interfaces.FilterChainsParams to filter_chains.Params
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
// This method implements the interfaces.FilterChainBuilder interface
func (a *FilterChainAdapter) BuildFilterChainParams(
	vs *v1alpha1.VirtualService,
	nn helpers.NamespacedName,
	httpFilters []*hcmv3.HttpFilter,
	listenerIsTLS bool,
	virtualHost *routev3.VirtualHost,
) (*interfaces.FilterChainsParams, error) {
	// Create basic filter chain parameters
	filterChainParams := &interfaces.FilterChainsParams{
		VSName:            nn.String(),
		UseRemoteAddress:  helpers.BoolFromPtr(vs.Spec.UseRemoteAddress),
		RouteConfigName:   nn.String(),
		StatPrefix:        strings.ReplaceAll(nn.String(), ".", "-"),
		HTTPFilters:       httpFilters,
		IsTLS:             listenerIsTLS,
		XFFNumTrustedHops: vs.Spec.XFFNumTrustedHops,
	}

	// 1. Handle tracing configuration
	if vs.Spec.Tracing != nil && vs.Spec.TracingRef != nil {
		return nil, fmt.Errorf("only one of spec.tracing or spec.tracingRef may be set")
	}

	if vs.Spec.Tracing != nil {
		tracing := &hcmv3.HttpConnectionManager_Tracing{}
		if err := protoutil.Unmarshaler.Unmarshal(vs.Spec.Tracing.Raw, tracing); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tracing: %w", err)
		}
		if err := tracing.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate tracing: %w", err)
		}
		filterChainParams.Tracing = tracing
	} else if vs.Spec.TracingRef != nil {
		tracingRefNs := helpers.GetNamespace(vs.Spec.TracingRef.Namespace, vs.Namespace)
		tr := a.store.GetTracing(helpers.NamespacedName{Namespace: tracingRefNs, Name: vs.Spec.TracingRef.Name})
		if tr == nil {
			return nil, fmt.Errorf("tracing %s/%s not found", tracingRefNs, vs.Spec.TracingRef.Name)
		}
		trv3, err := tr.UnmarshalV3AndValidate()
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal tracing: %w", err)
		}
		filterChainParams.Tracing = trv3
	}

	// 2. Handle upgrade configs
	if vs.Spec.UpgradeConfigs != nil {
		upgradeConfigs := make([]*hcmv3.HttpConnectionManager_UpgradeConfig, 0, len(vs.Spec.UpgradeConfigs))
		for _, upgradeConfig := range vs.Spec.UpgradeConfigs {
			uc := &hcmv3.HttpConnectionManager_UpgradeConfig{}
			if err := protoutil.Unmarshaler.Unmarshal(upgradeConfig.Raw, uc); err != nil {
				return nil, fmt.Errorf("failed to unmarshal upgrade config: %w", err)
			}
			if err := uc.ValidateAll(); err != nil {
				return nil, fmt.Errorf("failed to validate upgrade config: %w", err)
			}
			upgradeConfigs = append(upgradeConfigs, uc)
		}
		filterChainParams.UpgradeConfigs = upgradeConfigs
	}

	// 3. Handle access log configs
	var accessLogCount int
	if vs.Spec.AccessLog != nil {
		accessLogCount++
	}
	if vs.Spec.AccessLogConfig != nil {
		accessLogCount++
	}
	if len(vs.Spec.AccessLogs) > 0 {
		accessLogCount++
	}
	if len(vs.Spec.AccessLogConfigs) > 0 {
		accessLogCount++
	}

	if accessLogCount > 1 {
		return nil, fmt.Errorf("can't use accessLog, accessLogConfig, accessLogs and accessLogConfigs at the same time")
	}

	if accessLogCount > 0 {
		var accessLogs []*accesslogv3.AccessLog

		// Handle inline access log
		if vs.Spec.AccessLog != nil {
			vs.UpdateStatus(false, "accessLog is deprecated, use accessLogs instead")
			var accessLog accesslogv3.AccessLog
			if err := protoutil.Unmarshaler.Unmarshal(vs.Spec.AccessLog.Raw, &accessLog); err != nil {
				return nil, fmt.Errorf("failed to unmarshal accessLog: %w", err)
			}
			if err := accessLog.ValidateAll(); err != nil {
				return nil, err
			}
			accessLogs = append(accessLogs, &accessLog)
		}

		// Handle access log config reference
		if vs.Spec.AccessLogConfig != nil {
			vs.UpdateStatus(false, "accessLogConfig is deprecated, use accessLogConfigs instead")
			accessLogNs := helpers.GetNamespace(vs.Spec.AccessLogConfig.Namespace, vs.Namespace)
			accessLogConfig := a.store.GetAccessLog(helpers.NamespacedName{Namespace: accessLogNs, Name: vs.Spec.AccessLogConfig.Name})
			if accessLogConfig == nil {
				return nil, fmt.Errorf("can't find accessLogConfig %s/%s", accessLogNs, vs.Spec.AccessLogConfig.Name)
			}
			accessLog, err := accessLogConfig.UnmarshalAndValidateV3(v1alpha1.WithAccessLogFileName(vs.Name))
			if err != nil {
				return nil, err
			}
			accessLogs = append(accessLogs, accessLog)
		}

		// Handle multiple inline access logs
		if len(vs.Spec.AccessLogs) > 0 {
			for _, accessLogSpec := range vs.Spec.AccessLogs {
				var accessLogV3 accesslogv3.AccessLog
				if err := protoutil.Unmarshaler.Unmarshal(accessLogSpec.Raw, &accessLogV3); err != nil {
					return nil, fmt.Errorf("failed to unmarshal accessLog: %w", err)
				}
				if err := accessLogV3.ValidateAll(); err != nil {
					return nil, err
				}
				accessLogs = append(accessLogs, &accessLogV3)
			}
		}

		// Handle multiple access log config references
		if len(vs.Spec.AccessLogConfigs) > 0 {
			for _, accessLogConfig := range vs.Spec.AccessLogConfigs {
				accessLogNs := helpers.GetNamespace(accessLogConfig.Namespace, vs.Namespace)
				accessLog := a.store.GetAccessLog(helpers.NamespacedName{Namespace: accessLogNs, Name: accessLogConfig.Name})
				if accessLog == nil {
					return nil, fmt.Errorf("can't find accessLogConfig %s/%s", accessLogNs, accessLogConfig.Name)
				}
				accessLogV3, err := accessLog.UnmarshalAndValidateV3(v1alpha1.WithAccessLogFileName(vs.Name))
				if err != nil {
					return nil, err
				}
				accessLogs = append(accessLogs, accessLogV3)
			}
		}

		filterChainParams.AccessLogs = accessLogs
	}

	// 4. Handle TLS configuration
	if listenerIsTLS && vs.Spec.TlsConfig == nil {
		return nil, fmt.Errorf("tls listener not configured, virtual service has not tls config")
	}
	if !listenerIsTLS && vs.Spec.TlsConfig != nil {
		return nil, fmt.Errorf("listener is not tls, virtual service has tls config")
	}

	if vs.Spec.TlsConfig != nil {
		tlsAdapter := NewTLSAdapter(a.store)
		// Validate TLS configuration type
		if _, err := tlsAdapter.GetTLSType(vs.Spec.TlsConfig); err != nil {
			return nil, err
		}

		// Get SecretNameToDomains mapping based on TLS configuration
		var err error
		filterChainParams.SecretNameToDomains, err = tlsAdapter.GetSecretNameToDomains(vs, virtualHost.Domains)
		if err != nil {
			return nil, err
		}
	}

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
