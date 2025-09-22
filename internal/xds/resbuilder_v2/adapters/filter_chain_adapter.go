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
	"k8s.io/apimachinery/pkg/runtime"
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

	// Handle tracing configuration
	tracing, err := a.configureTracing(vs)
	if err != nil {
		return nil, err
	}
	filterChainParams.Tracing = tracing

	// Handle upgrade configs
	upgradeConfigs, err := a.configureUpgradeConfigs(vs)
	if err != nil {
		return nil, err
	}
	filterChainParams.UpgradeConfigs = upgradeConfigs

	// Handle access logs
	accessLogs, err := a.configureAccessLogs(vs)
	if err != nil {
		return nil, err
	}
	filterChainParams.AccessLogs = accessLogs

	// Handle TLS configuration
	if err := a.configureTLS(vs, listenerIsTLS, virtualHost, filterChainParams); err != nil {
		return nil, err
	}

	return filterChainParams, nil
}

// configureTracing handles tracing configuration for the filter chain
func (a *FilterChainAdapter) configureTracing(vs *v1alpha1.VirtualService) (*hcmv3.HttpConnectionManager_Tracing, error) {
	if vs.Spec.Tracing != nil && vs.Spec.TracingRef != nil {
		return nil, fmt.Errorf("only one of spec.tracing or spec.tracingRef may be set")
	}

	if vs.Spec.Tracing != nil {
		return a.unmarshalInlineTracing(vs.Spec.Tracing)
	}

	if vs.Spec.TracingRef != nil {
		return a.resolveTracingRef(vs)
	}

	return nil, nil
}

// unmarshalInlineTracing unmarshals inline tracing configuration
func (a *FilterChainAdapter) unmarshalInlineTracing(tracingSpec *runtime.RawExtension) (*hcmv3.HttpConnectionManager_Tracing, error) {
	tracing := &hcmv3.HttpConnectionManager_Tracing{}
	if err := protoutil.Unmarshaler.Unmarshal(tracingSpec.Raw, tracing); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tracing: %w", err)
	}
	if err := tracing.ValidateAll(); err != nil {
		return nil, fmt.Errorf("failed to validate tracing: %w", err)
	}
	return tracing, nil
}

// resolveTracingRef resolves tracing reference from store
func (a *FilterChainAdapter) resolveTracingRef(vs *v1alpha1.VirtualService) (*hcmv3.HttpConnectionManager_Tracing, error) {
	tracingRefNs := helpers.GetNamespace(vs.Spec.TracingRef.Namespace, vs.Namespace)
	tr := a.store.GetTracing(helpers.NamespacedName{Namespace: tracingRefNs, Name: vs.Spec.TracingRef.Name})
	if tr == nil {
		return nil, fmt.Errorf("tracing %s/%s not found", tracingRefNs, vs.Spec.TracingRef.Name)
	}
	trv3, err := tr.UnmarshalV3AndValidate()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tracing: %w", err)
	}
	return trv3, nil
}

// configureUpgradeConfigs handles upgrade configurations
func (a *FilterChainAdapter) configureUpgradeConfigs(vs *v1alpha1.VirtualService) ([]*hcmv3.HttpConnectionManager_UpgradeConfig, error) {
	if vs.Spec.UpgradeConfigs == nil {
		return nil, nil
	}

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
	return upgradeConfigs, nil
}

// configureAccessLogs handles access log configuration
func (a *FilterChainAdapter) configureAccessLogs(vs *v1alpha1.VirtualService) ([]*accesslogv3.AccessLog, error) {
	// Validate that only one type of access log configuration is used
	if err := a.validateAccessLogConfiguration(vs); err != nil {
		return nil, err
	}

	if !a.hasAccessLogConfiguration(vs) {
		return nil, nil
	}

	var accessLogs []*accesslogv3.AccessLog

	// Process deprecated single access log
	if vs.Spec.AccessLog != nil {
		logs, err := a.processDeprecatedAccessLog(vs)
		if err != nil {
			return nil, err
		}
		accessLogs = append(accessLogs, logs...)
	}

	// Process deprecated single access log config reference
	if vs.Spec.AccessLogConfig != nil {
		logs, err := a.processDeprecatedAccessLogConfig(vs)
		if err != nil {
			return nil, err
		}
		accessLogs = append(accessLogs, logs...)
	}

	// Process multiple inline access logs
	if len(vs.Spec.AccessLogs) > 0 {
		logs, err := a.processInlineAccessLogs(vs.Spec.AccessLogs)
		if err != nil {
			return nil, err
		}
		accessLogs = append(accessLogs, logs...)
	}

	// Process multiple access log config references
	if len(vs.Spec.AccessLogConfigs) > 0 {
		logs, err := a.processAccessLogConfigRefs(vs)
		if err != nil {
			return nil, err
		}
		accessLogs = append(accessLogs, logs...)
	}

	return accessLogs, nil
}

// validateAccessLogConfiguration ensures only one type of access log config is used
func (a *FilterChainAdapter) validateAccessLogConfiguration(vs *v1alpha1.VirtualService) error {
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
		return fmt.Errorf("can't use accessLog, accessLogConfig, accessLogs and accessLogConfigs at the same time")
	}
	return nil
}

// hasAccessLogConfiguration checks if any access log configuration exists
func (a *FilterChainAdapter) hasAccessLogConfiguration(vs *v1alpha1.VirtualService) bool {
	return vs.Spec.AccessLog != nil ||
		vs.Spec.AccessLogConfig != nil ||
		len(vs.Spec.AccessLogs) > 0 ||
		len(vs.Spec.AccessLogConfigs) > 0
}

// processDeprecatedAccessLog processes deprecated single inline access log
func (a *FilterChainAdapter) processDeprecatedAccessLog(vs *v1alpha1.VirtualService) ([]*accesslogv3.AccessLog, error) {
	vs.UpdateStatus(false, "accessLog is deprecated, use accessLogs instead")
	var accessLog accesslogv3.AccessLog
	if err := protoutil.Unmarshaler.Unmarshal(vs.Spec.AccessLog.Raw, &accessLog); err != nil {
		return nil, fmt.Errorf("failed to unmarshal accessLog: %w", err)
	}
	if err := accessLog.ValidateAll(); err != nil {
		return nil, err
	}
	return []*accesslogv3.AccessLog{&accessLog}, nil
}

// processDeprecatedAccessLogConfig processes deprecated single access log config reference
func (a *FilterChainAdapter) processDeprecatedAccessLogConfig(vs *v1alpha1.VirtualService) ([]*accesslogv3.AccessLog, error) {
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
	return []*accesslogv3.AccessLog{accessLog}, nil
}

// processInlineAccessLogs processes multiple inline access logs
func (a *FilterChainAdapter) processInlineAccessLogs(accessLogSpecs []*runtime.RawExtension) ([]*accesslogv3.AccessLog, error) {
	accessLogs := make([]*accesslogv3.AccessLog, 0, len(accessLogSpecs))
	for _, accessLogSpec := range accessLogSpecs {
		var accessLogV3 accesslogv3.AccessLog
		if err := protoutil.Unmarshaler.Unmarshal(accessLogSpec.Raw, &accessLogV3); err != nil {
			return nil, fmt.Errorf("failed to unmarshal accessLog: %w", err)
		}
		if err := accessLogV3.ValidateAll(); err != nil {
			return nil, err
		}
		accessLogs = append(accessLogs, &accessLogV3)
	}
	return accessLogs, nil
}

// processAccessLogConfigRefs processes multiple access log config references
func (a *FilterChainAdapter) processAccessLogConfigRefs(vs *v1alpha1.VirtualService) ([]*accesslogv3.AccessLog, error) {
	accessLogs := make([]*accesslogv3.AccessLog, 0, len(vs.Spec.AccessLogConfigs))
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
	return accessLogs, nil
}

// configureTLS handles TLS configuration for the filter chain
func (a *FilterChainAdapter) configureTLS(vs *v1alpha1.VirtualService, listenerIsTLS bool, virtualHost *routev3.VirtualHost, params *interfaces.FilterChainsParams) error {
	// Validate TLS configuration consistency
	if err := a.validateTLSConfiguration(vs, listenerIsTLS); err != nil {
		return err
	}

	if vs.Spec.TlsConfig == nil {
		return nil
	}

	tlsAdapter := NewTLSAdapter(a.store)
	// Validate TLS configuration type
	if _, err := tlsAdapter.GetTLSType(vs.Spec.TlsConfig); err != nil {
		return err
	}

	// Get SecretNameToDomains mapping based on TLS configuration
	var err error
	params.SecretNameToDomains, err = tlsAdapter.GetSecretNameToDomains(vs, virtualHost.Domains)
	return err
}

// validateTLSConfiguration validates TLS configuration consistency
func (a *FilterChainAdapter) validateTLSConfiguration(vs *v1alpha1.VirtualService, listenerIsTLS bool) error {
	if listenerIsTLS && vs.Spec.TlsConfig == nil {
		return fmt.Errorf("tls listener not configured, virtual service has not tls config")
	}
	if !listenerIsTLS && vs.Spec.TlsConfig != nil {
		return fmt.Errorf("listener is not tls, virtual service has tls config")
	}
	return nil
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
