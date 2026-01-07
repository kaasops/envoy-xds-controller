package filter_chains

import (
	"fmt"
	"slices"
	"strings"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/secrets"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/runtime"
)

// Builder implements filter chain building functionality
type Builder struct {
	store store.Store
}

// NewBuilder creates a new filter chain builder
func NewBuilder(store store.Store) *Builder {
	return &Builder{
		store: store,
	}
}

// BuildFilterChains builds multiple filter chains from the provided parameters
func (b *Builder) BuildFilterChains(params *FilterChainsParams) ([]*listenerv3.FilterChain, error) {
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
func (b *Builder) buildFilterChain(params *FilterChainsParams) (*listenerv3.FilterChain, error) {
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

// FilterChainBuilder interface implementation
// These methods implement the interfaces.FilterChainBuilder interface

// BuildFilterChainParams builds filter chain parameters from a VirtualService
func (b *Builder) BuildFilterChainParams(
	vs *v1alpha1.VirtualService,
	nn helpers.NamespacedName,
	httpFilters []*hcmv3.HttpFilter,
	listenerIsTLS bool,
	virtualHost *routev3.VirtualHost,
) (*FilterChainsParams, error) {
	params := &FilterChainsParams{
		VSName:            nn.String(),
		UseRemoteAddress:  helpers.BoolFromPtr(vs.Spec.UseRemoteAddress),
		RouteConfigName:   nn.String(),
		StatPrefix:        strings.ReplaceAll(nn.String(), ".", "-"),
		HTTPFilters:       httpFilters,
		IsTLS:             listenerIsTLS,
		XFFNumTrustedHops: vs.Spec.XFFNumTrustedHops,
	}

	// Handle tracing configuration
	tracing, err := b.configureTracing(vs)
	if err != nil {
		return nil, err
	}
	params.Tracing = tracing

	// Handle upgrade configs
	upgradeConfigs, err := b.configureUpgradeConfigs(vs)
	if err != nil {
		return nil, err
	}
	params.UpgradeConfigs = upgradeConfigs

	// Handle access logs
	accessLogs, err := b.configureAccessLogs(vs)
	if err != nil {
		return nil, err
	}
	params.AccessLogs = accessLogs

	// Handle TLS configuration
	if err := b.configureTLS(vs, listenerIsTLS, virtualHost, params); err != nil {
		return nil, err
	}

	return params, nil
}

// configureTracing handles tracing configuration for the filter chain
func (b *Builder) configureTracing(vs *v1alpha1.VirtualService) (*hcmv3.HttpConnectionManager_Tracing, error) {
	if vs.Spec.Tracing != nil && vs.Spec.TracingRef != nil {
		return nil, fmt.Errorf("only one of spec.tracing or spec.tracingRef may be set")
	}

	if vs.Spec.Tracing != nil {
		return b.unmarshalInlineTracing(vs.Spec.Tracing)
	}

	if vs.Spec.TracingRef != nil {
		return b.resolveTracingRef(vs)
	}

	return nil, nil
}

// unmarshalInlineTracing unmarshals inline tracing configuration
func (b *Builder) unmarshalInlineTracing(
	tracingSpec *runtime.RawExtension,
) (*hcmv3.HttpConnectionManager_Tracing, error) {
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
func (b *Builder) resolveTracingRef(vs *v1alpha1.VirtualService) (*hcmv3.HttpConnectionManager_Tracing, error) {
	tracingRefNs := helpers.GetNamespace(vs.Spec.TracingRef.Namespace, vs.Namespace)
	tr := b.store.GetTracing(helpers.NamespacedName{Namespace: tracingRefNs, Name: vs.Spec.TracingRef.Name})
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
func (b *Builder) configureUpgradeConfigs(
	vs *v1alpha1.VirtualService,
) ([]*hcmv3.HttpConnectionManager_UpgradeConfig, error) {
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
func (b *Builder) configureAccessLogs(vs *v1alpha1.VirtualService) ([]*accesslogv3.AccessLog, error) {
	if err := b.validateAccessLogConfiguration(vs); err != nil {
		return nil, err
	}

	if !b.hasAccessLogConfiguration(vs) {
		return nil, nil
	}

	var accessLogs []*accesslogv3.AccessLog

	if vs.Spec.AccessLog != nil {
		logs, err := b.processDeprecatedAccessLog(vs)
		if err != nil {
			return nil, err
		}
		accessLogs = append(accessLogs, logs...)
	}

	if vs.Spec.AccessLogConfig != nil {
		logs, err := b.processDeprecatedAccessLogConfig(vs)
		if err != nil {
			return nil, err
		}
		accessLogs = append(accessLogs, logs...)
	}

	if len(vs.Spec.AccessLogs) > 0 {
		logs, err := b.processInlineAccessLogs(vs.Spec.AccessLogs)
		if err != nil {
			return nil, err
		}
		accessLogs = append(accessLogs, logs...)
	}

	if len(vs.Spec.AccessLogConfigs) > 0 {
		logs, err := b.processAccessLogConfigRefs(vs)
		if err != nil {
			return nil, err
		}
		accessLogs = append(accessLogs, logs...)
	}

	return accessLogs, nil
}

func (b *Builder) validateAccessLogConfiguration(vs *v1alpha1.VirtualService) error {
	var count int
	if vs.Spec.AccessLog != nil {
		count++
	}
	if vs.Spec.AccessLogConfig != nil {
		count++
	}
	if len(vs.Spec.AccessLogs) > 0 {
		count++
	}
	if len(vs.Spec.AccessLogConfigs) > 0 {
		count++
	}
	if count > 1 {
		return fmt.Errorf("can't use accessLog, accessLogConfig, accessLogs and accessLogConfigs at the same time")
	}
	return nil
}

func (b *Builder) hasAccessLogConfiguration(vs *v1alpha1.VirtualService) bool {
	return vs.Spec.AccessLog != nil ||
		vs.Spec.AccessLogConfig != nil ||
		len(vs.Spec.AccessLogs) > 0 ||
		len(vs.Spec.AccessLogConfigs) > 0
}

func (b *Builder) processDeprecatedAccessLog(vs *v1alpha1.VirtualService) ([]*accesslogv3.AccessLog, error) {
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

func (b *Builder) processDeprecatedAccessLogConfig(vs *v1alpha1.VirtualService) ([]*accesslogv3.AccessLog, error) {
	vs.UpdateStatus(false, "accessLogConfig is deprecated, use accessLogConfigs instead")
	accessLogNs := helpers.GetNamespace(vs.Spec.AccessLogConfig.Namespace, vs.Namespace)
	nn := helpers.NamespacedName{Namespace: accessLogNs, Name: vs.Spec.AccessLogConfig.Name}
	accessLogConfig := b.store.GetAccessLog(nn)
	if accessLogConfig == nil {
		return nil, fmt.Errorf("can't find accessLogConfig %s/%s", accessLogNs, vs.Spec.AccessLogConfig.Name)
	}
	accessLog, err := accessLogConfig.UnmarshalAndValidateV3(v1alpha1.WithAccessLogFileName(vs.Name))
	if err != nil {
		return nil, err
	}
	return []*accesslogv3.AccessLog{accessLog}, nil
}

func (b *Builder) processInlineAccessLogs(accessLogSpecs []*runtime.RawExtension) ([]*accesslogv3.AccessLog, error) {
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

func (b *Builder) processAccessLogConfigRefs(vs *v1alpha1.VirtualService) ([]*accesslogv3.AccessLog, error) {
	accessLogs := make([]*accesslogv3.AccessLog, 0, len(vs.Spec.AccessLogConfigs))
	for _, accessLogConfig := range vs.Spec.AccessLogConfigs {
		accessLogNs := helpers.GetNamespace(accessLogConfig.Namespace, vs.Namespace)
		accessLog := b.store.GetAccessLog(helpers.NamespacedName{Namespace: accessLogNs, Name: accessLogConfig.Name})
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
func (b *Builder) configureTLS(
	vs *v1alpha1.VirtualService,
	listenerIsTLS bool,
	virtualHost *routev3.VirtualHost,
	params *FilterChainsParams,
) error {
	if err := b.validateTLSConfiguration(vs, listenerIsTLS); err != nil {
		return err
	}

	if vs.Spec.TlsConfig == nil {
		return nil
	}

	tlsBuilder := secrets.NewBuilder(b.store)
	if _, err := tlsBuilder.GetTLSType(vs.Spec.TlsConfig); err != nil {
		return err
	}

	var err error
	params.SecretNameToDomains, err = tlsBuilder.GetSecretNameToDomains(vs, virtualHost.Domains)
	return err
}

func (b *Builder) validateTLSConfiguration(vs *v1alpha1.VirtualService, listenerIsTLS bool) error {
	if listenerIsTLS && vs.Spec.TlsConfig == nil {
		return fmt.Errorf("tls listener not configured, virtual service has not tls config")
	}
	if !listenerIsTLS && vs.Spec.TlsConfig != nil {
		return fmt.Errorf("listener is not tls, virtual service has tls config")
	}
	return nil
}

// CheckFilterChainsConflicts checks for conflicts between existing filter chains and virtual service configuration
func (b *Builder) CheckFilterChainsConflicts(vs *v1alpha1.VirtualService) error {
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
