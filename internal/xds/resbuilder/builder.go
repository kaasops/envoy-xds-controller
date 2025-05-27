package resbuilder

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	oauth2v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/oauth2/v3"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	rbacFilter "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcpProxyv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	SecretRefType     = "secretRef"
	AutoDiscoveryType = "autoDiscoveryType"
)

type FilterChainsParams struct {
	VSName               string
	UseRemoteAddress     bool
	XFFNumTrustedHops    *uint32
	RouteConfigName      string
	StatPrefix           string
	HTTPFilters          []*hcmv3.HttpFilter
	UpgradeConfigs       []*hcmv3.HttpConnectionManager_UpgradeConfig
	AccessLog            *accesslogv3.AccessLog
	Domains              []string
	DownstreamTLSContext *tlsv3.DownstreamTlsContext
	SecretNameToDomains  map[helpers.NamespacedName][]string
	IsTLS                bool
}

type Resources struct {
	Listener    helpers.NamespacedName
	FilterChain []*listenerv3.FilterChain
	RouteConfig *routev3.RouteConfiguration
	Clusters    []*cluster.Cluster
	Secrets     []*tlsv3.Secret
	UsedSecrets []helpers.NamespacedName
	Domains     []string
}

func BuildResources(vs *v1alpha1.VirtualService, store *store.Store) (*Resources, error) {
	var err error
	nn := helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}

	// Apply template if specified
	vs, err = applyVirtualServiceTemplate(vs, store)
	if err != nil {
		return nil, err
	}

	// Build listener
	listenerNN, err := vs.GetListenerNamespacedName()
	if err != nil {
		return nil, err
	}

	xdsListener, err := buildListener(listenerNN, store)
	if err != nil {
		return nil, err
	}

	// If listener already has filter chains, use them
	if len(xdsListener.FilterChains) > 0 {
		return buildResourcesFromExistingFilterChains(vs, xdsListener, listenerNN, store)
	}

	// Otherwise, build resources from virtual service configuration
	return buildResourcesFromVirtualService(vs, xdsListener, listenerNN, nn, store)
}

// applyVirtualServiceTemplate applies a template to the virtual service if specified
func applyVirtualServiceTemplate(vs *v1alpha1.VirtualService, store *store.Store) (*v1alpha1.VirtualService, error) {
	if vs.Spec.Template == nil {
		return vs, nil
	}

	templateNamespace := helpers.GetNamespace(vs.Spec.Template.Namespace, vs.Namespace)
	templateName := vs.Spec.Template.Name
	templateNN := helpers.NamespacedName{Namespace: templateNamespace, Name: templateName}

	vst := store.GetVirtualServiceTemplate(templateNN)
	if vst == nil {
		return nil, fmt.Errorf("virtual service template %s/%s not found", templateNamespace, templateName)
	}

	vsCopy := vs.DeepCopy()
	if err := vsCopy.FillFromTemplate(vst, vs.Spec.TemplateOptions...); err != nil {
		return nil, err
	}

	return vsCopy, nil
}

// buildResourcesFromExistingFilterChains builds resources using existing filter chains from the listener
func buildResourcesFromExistingFilterChains(vs *v1alpha1.VirtualService, xdsListener *listenerv3.Listener, listenerNN helpers.NamespacedName, store *store.Store) (*Resources, error) {
	// Check for conflicts with virtual service configuration
	if err := checkFilterChainsConflicts(vs); err != nil {
		return nil, err
	}

	if len(xdsListener.FilterChains) > 1 {
		return nil, fmt.Errorf("multiple filter chains found")
	}

	// Extract clusters from filter chains
	clusters, err := extractClustersFromFilterChains(xdsListener.FilterChains, store)
	if err != nil {
		return nil, err
	}

	return &Resources{
		Listener:    listenerNN,
		FilterChain: xdsListener.FilterChains,
		Clusters:    clusters,
	}, nil
}

// checkFilterChainsConflicts checks for conflicts between existing filter chains and virtual service configuration
func checkFilterChainsConflicts(vs *v1alpha1.VirtualService) error {
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
	}

	for _, conflict := range conflicts {
		if conflict.condition {
			return fmt.Errorf("conflict: %s", conflict.message)
		}
	}

	return nil
}

// extractClustersFromFilterChains extracts clusters from filter chains
func extractClustersFromFilterChains(filterChains []*listenerv3.FilterChain, store *store.Store) ([]*cluster.Cluster, error) {
	clusters := make([]*cluster.Cluster, 0)

	for _, fc := range filterChains {
		for _, filter := range fc.Filters {
			if tc := filter.GetTypedConfig(); tc != nil {
				if tc.TypeUrl != "type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy" {
					return nil, fmt.Errorf("unexpected filter type: %s", tc.TypeUrl)
				}

				var tcpProxy tcpProxyv3.TcpProxy
				if err := tc.UnmarshalTo(&tcpProxy); err != nil {
					return nil, err
				}

				clusterName := tcpProxy.GetCluster()
				cl := store.GetSpecCluster(clusterName)
				if cl == nil {
					return nil, fmt.Errorf("cluster %s not found", clusterName)
				}

				xdsCluster, err := cl.UnmarshalV3AndValidate()
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal cluster %s: %w", clusterName, err)
				}

				clusters = append(clusters, xdsCluster)
			}
		}
	}

	return clusters, nil
}

// buildResourcesFromVirtualService builds resources from virtual service configuration
func buildResourcesFromVirtualService(vs *v1alpha1.VirtualService, xdsListener *listenerv3.Listener, listenerNN helpers.NamespacedName, nn helpers.NamespacedName, store *store.Store) (*Resources, error) {
	listenerIsTLS := isTLSListener(xdsListener)

	// Build virtual host and route configuration
	virtualHost, routeConfiguration, err := buildRouteConfiguration(vs, xdsListener, nn, store)
	if err != nil {
		return nil, err
	}

	// Build HTTP filters
	httpFilters, err := buildHTTPFilters(vs, store)
	if err != nil {
		return nil, err
	}

	// Build filter chain parameters
	filterChainParams, err := buildFilterChainParams(vs, nn, httpFilters, listenerIsTLS, virtualHost, store)
	if err != nil {
		return nil, err
	}

	// Build filter chains
	fcs, err := buildFilterChains(filterChainParams)
	if err != nil {
		return nil, err
	}

	// Update listener with filter chains
	xdsListener.FilterChains = fcs
	if err := xdsListener.ValidateAll(); err != nil {
		return nil, err
	}

	// Build clusters
	clusters, err := buildClusters(virtualHost, httpFilters, store)
	if err != nil {
		return nil, err
	}

	// Build secrets
	secrets, usedSecrets, err := buildSecrets(httpFilters, filterChainParams.SecretNameToDomains, store)
	if err != nil {
		return nil, fmt.Errorf("failed to build secrets: %w", err)
	}

	return &Resources{
		Listener:    listenerNN,
		FilterChain: fcs,
		RouteConfig: routeConfiguration,
		Clusters:    clusters,
		Secrets:     secrets,
		UsedSecrets: usedSecrets,
		Domains:     virtualHost.Domains,
	}, nil
}

// buildRouteConfiguration builds the route configuration from the virtual service
func buildRouteConfiguration(
	vs *v1alpha1.VirtualService,
	xdsListener *listenerv3.Listener,
	nn helpers.NamespacedName,
	store *store.Store,
) (*routev3.VirtualHost, *routev3.RouteConfiguration, error) {
	virtualHost, err := buildVirtualHost(vs, nn, store)
	if err != nil {
		return nil, nil, err
	}

	routeConfiguration := &routev3.RouteConfiguration{
		Name:         nn.String(),
		VirtualHosts: []*routev3.VirtualHost{virtualHost},
	}

	// Add fallback route for TLS listeners
	// https://github.com/envoyproxy/envoy/issues/37810
	listenerIsTLS := isTLSListener(xdsListener)
	if listenerIsTLS && !(len(virtualHost.Domains) == 1 && virtualHost.Domains[0] == "*") {
		routeConfiguration.VirtualHosts = append(routeConfiguration.VirtualHosts, &routev3.VirtualHost{
			Name:    "421vh",
			Domains: []string{"*"},
			Routes: []*routev3.Route{
				{
					Match: &routev3.RouteMatch{PathSpecifier: &routev3.RouteMatch_Prefix{Prefix: "/"}},
					Action: &routev3.Route_DirectResponse{
						DirectResponse: &routev3.DirectResponseAction{
							Status: 421,
						},
					},
				},
			},
		})
	}

	if err = routeConfiguration.ValidateAll(); err != nil {
		return nil, nil, err
	}

	return virtualHost, routeConfiguration, nil
}

// buildFilterChainParams builds the filter chain parameters
func buildFilterChainParams(vs *v1alpha1.VirtualService, nn helpers.NamespacedName, httpFilters []*hcmv3.HttpFilter, listenerIsTLS bool, virtualHost *routev3.VirtualHost, store *store.Store) (*FilterChainsParams, error) {
	filterChainParams := &FilterChainsParams{
		VSName:            nn.String(),
		UseRemoteAddress:  helpers.BoolFromPtr(vs.Spec.UseRemoteAddress),
		RouteConfigName:   nn.String(),
		StatPrefix:        strings.ReplaceAll(nn.String(), ".", "-"),
		HTTPFilters:       httpFilters,
		IsTLS:             listenerIsTLS,
		XFFNumTrustedHops: vs.Spec.XFFNumTrustedHops,
	}

	// Build upgrade configs
	upgradeConfigs, err := buildUpgradeConfigs(vs.Spec.UpgradeConfigs)
	if err != nil {
		return nil, err
	}
	filterChainParams.UpgradeConfigs = upgradeConfigs

	// Build access log config
	accessLog, err := buildAccessLogConfig(vs, store)
	if err != nil {
		return nil, err
	}
	filterChainParams.AccessLog = accessLog

	// Check TLS configuration
	if listenerIsTLS && vs.Spec.TlsConfig == nil {
		return nil, fmt.Errorf("tls listener not configured, virtual service has not tls config")
	}
	if !listenerIsTLS && vs.Spec.TlsConfig != nil {
		return nil, fmt.Errorf("listener is not tls, virtual service has tls config")
	}

	// Process TLS configuration
	if vs.Spec.TlsConfig != nil {
		tlsType, err := getTLSType(vs.Spec.TlsConfig)
		if err != nil {
			return nil, err
		}

		switch tlsType {
		case SecretRefType:
			filterChainParams.SecretNameToDomains = getSecretNameToDomainsViaSecretRef(vs.Spec.TlsConfig.SecretRef, vs.Namespace, virtualHost.Domains)
		case AutoDiscoveryType:
			filterChainParams.SecretNameToDomains, err = getSecretNameToDomainsViaAutoDiscovery(virtualHost.Domains, store.MapDomainSecrets())
			if err != nil {
				return nil, err
			}
		}
	}

	return filterChainParams, nil
}

func buildListener(listenerNN helpers.NamespacedName, store *store.Store) (*listenerv3.Listener, error) {
	listener := store.GetListener(listenerNN)
	if listener == nil {
		return nil, fmt.Errorf("listener %s not found", listenerNN.String())
	}
	xdsListener, err := listener.UnmarshalV3()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal listener %s: %w", listenerNN.String(), err)
	}
	xdsListener.Name = listenerNN.String()
	return xdsListener, nil
}

func buildVirtualHost(vs *v1alpha1.VirtualService, nn helpers.NamespacedName, store *store.Store) (*routev3.VirtualHost, error) {
	if vs.Spec.VirtualHost == nil {
		return nil, fmt.Errorf("virtual host is empty")
	}

	virtualHost := &routev3.VirtualHost{}
	if err := protoutil.Unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, virtualHost); err != nil {
		return nil, fmt.Errorf("failed to unmarshal virtual host: %w", err)
	}
	virtualHost.Name = nn.String()

	for _, routeRef := range vs.Spec.AdditionalRoutes {
		routeRefNs := helpers.GetNamespace(routeRef.Namespace, vs.Namespace)
		route := store.GetRoute(helpers.NamespacedName{Namespace: routeRefNs, Name: routeRef.Name})
		if route == nil {
			return nil, fmt.Errorf("route %s/%s not found", routeRefNs, routeRef.Name)
		}
		for idx, rt := range route.Spec {
			var r routev3.Route
			if err := protoutil.Unmarshaler.Unmarshal(rt.Raw, &r); err != nil {
				return nil, fmt.Errorf("failed to unmarshal route %s/%s (%d): %w", routeRefNs, routeRef.Name, idx, err)
			}
			if rr := r.GetRoute(); rr != nil {
				if clName := rr.GetCluster(); clName != "" {
					cl := store.GetSpecCluster(clName)
					if cl == nil {
						return nil, fmt.Errorf("cluster %s not found", clName)
					}
				}
			}
			virtualHost.Routes = append(virtualHost.Routes, &r)
		}
	}

	rootMatchIndexes := make([]int, 0, 1)
	// reorder routes, root must be in the end
	for index, route := range virtualHost.Routes {
		if route.Match != nil && route.Match.GetPrefix() == "/" {
			rootMatchIndexes = append(rootMatchIndexes, index)
		}
	}

	switch {
	case len(rootMatchIndexes) > 1:
		return nil, fmt.Errorf("multiple root routes found")
	case len(rootMatchIndexes) == 1 && rootMatchIndexes[0] != len(virtualHost.Routes)-1:
		index := rootMatchIndexes[0]
		route := virtualHost.Routes[index]
		virtualHost.Routes = append(virtualHost.Routes[:index], virtualHost.Routes[index+1:]...)
		virtualHost.Routes = append(virtualHost.Routes, route)
	}

	if err := virtualHost.ValidateAll(); err != nil {
		return nil, fmt.Errorf("failed to validate virtual host: %w", err)
	}

	if err := checkAllDomainsUnique(virtualHost.Domains); err != nil {
		return nil, err
	}

	return virtualHost, nil
}

func buildHTTPFilters(vs *v1alpha1.VirtualService, store *store.Store) ([]*hcmv3.HttpFilter, error) {
	httpFilters := make([]*hcmv3.HttpFilter, 0, len(vs.Spec.HTTPFilters)+len(vs.Spec.AdditionalHttpFilters))

	rbacF, err := buildRBACFilter(vs, store)
	if err != nil {
		return nil, err
	}
	if rbacF != nil {
		configType := &hcmv3.HttpFilter_TypedConfig{
			TypedConfig: &anypb.Any{},
		}
		if err := configType.TypedConfig.MarshalFrom(rbacF); err != nil {
			return nil, err
		}
		httpFilters = append(httpFilters, &hcmv3.HttpFilter{
			Name:       "exc.filters.http.rbac",
			ConfigType: configType,
		})
	}

	for _, httpFilter := range vs.Spec.HTTPFilters {
		hf := &hcmv3.HttpFilter{}
		if err := protoutil.Unmarshaler.Unmarshal(httpFilter.Raw, hf); err != nil {
			return nil, fmt.Errorf("failed to unmarshal http filter: %w", err)
		}
		if err := hf.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate http filter: %w", err)
		}
		httpFilters = append(httpFilters, hf)
	}

	if len(vs.Spec.AdditionalHttpFilters) > 0 {
		for _, httpFilterRef := range vs.Spec.AdditionalHttpFilters {
			httpFilterRefNs := helpers.GetNamespace(httpFilterRef.Namespace, vs.Namespace)
			hf := store.GetHTTPFilter(helpers.NamespacedName{Namespace: httpFilterRefNs, Name: httpFilterRef.Name})
			if hf == nil {
				return nil, fmt.Errorf("http filter %s/%s not found", httpFilterRefNs, httpFilterRef.Name)
			}
			for _, filter := range hf.Spec {
				xdsHttpFilter := &hcmv3.HttpFilter{}
				if err := protoutil.Unmarshaler.Unmarshal(filter.Raw, xdsHttpFilter); err != nil {
					return nil, err
				}
				if err := xdsHttpFilter.ValidateAll(); err != nil {
					return nil, err
				}
				httpFilters = append(httpFilters, xdsHttpFilter)
			}
		}
	}

	// filter with type type.googleapis.com/envoy.extensions.filters.http.router.v3.Router must be in the end
	var routerIdxs []int
	for i, f := range httpFilters {
		if tc := f.GetTypedConfig(); tc != nil {
			if tc.TypeUrl == "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router" {
				routerIdxs = append(routerIdxs, i)
			}
		}
	}

	switch {
	case len(routerIdxs) > 1:
		return nil, fmt.Errorf("multiple root router http filters")
	case len(routerIdxs) == 1 && routerIdxs[0] != len(httpFilters)-1:
		index := routerIdxs[0]
		route := httpFilters[index]
		httpFilters = append(httpFilters[:index], httpFilters[index+1:]...)
		httpFilters = append(httpFilters, route)
	}

	return httpFilters, nil
}

func buildClusters(virtualHost *routev3.VirtualHost, httpFilters []*hcmv3.HttpFilter, store *store.Store) ([]*cluster.Cluster, error) {
	var clusters []*cluster.Cluster

	for _, route := range virtualHost.Routes {
		jsonData, err := json.Marshal(route)
		if err != nil {
			return nil, err
		}

		var data any
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return nil, err
		}

		clusterNames := findClusterNames(data, "Cluster")

		for _, clusterName := range clusterNames {
			cl := store.GetSpecCluster(clusterName)
			if cl == nil {
				return nil, fmt.Errorf("cluster %s not found", clusterName)
			}
			xdsCluster, err := cl.UnmarshalV3AndValidate()
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal cluster %s: %w", clusterName, err)
			}
			clusters = append(clusters, xdsCluster)
		}
	}

	for _, httpFilter := range httpFilters {
		if tc := httpFilter.GetTypedConfig(); tc != nil {
			if tc.TypeUrl == "type.googleapis.com/envoy.extensions.filters.http.oauth2.v3.OAuth2" {
				var oauthCfg oauth2v3.OAuth2
				if err := tc.UnmarshalTo(&oauthCfg); err != nil {
					return nil, fmt.Errorf("failed to unmarshal oauth2 config: %w", err)
				}
				jsonData, err := json.Marshal(oauthCfg.Config)
				if err != nil {
					return nil, err
				}

				var data any
				if err := json.Unmarshal(jsonData, &data); err != nil {
					return nil, err
				}

				clusterNames := findClusterNames(data, "Cluster")

				for _, clusterName := range clusterNames {
					cl := store.GetSpecCluster(clusterName)
					if cl == nil {
						return nil, fmt.Errorf("cluster %s not found", clusterName)
					}
					xdsCluster, err := cl.UnmarshalV3AndValidate()
					if err != nil {
						return nil, fmt.Errorf("failed to unmarshal cluster %s: %w", clusterName, err)
					}
					clusters = append(clusters, xdsCluster)
				}
			}
		}
	}

	return clusters, nil
}

func buildRBACFilter(vs *v1alpha1.VirtualService, store *store.Store) (*rbacFilter.RBAC, error) {
	if vs.Spec.RBAC == nil {
		return nil, nil
	}

	if vs.Spec.RBAC.Action == "" {
		return nil, fmt.Errorf("rbac action is empty")
	}

	action, ok := rbacv3.RBAC_Action_value[vs.Spec.RBAC.Action]
	if !ok {
		return nil, fmt.Errorf("invalid rbac action %s", vs.Spec.RBAC.Action)
	}

	if len(vs.Spec.RBAC.Policies) == 0 && len(vs.Spec.RBAC.AdditionalPolicies) == 0 {
		return nil, fmt.Errorf("rbac policies is empty")
	}

	rules := &rbacv3.RBAC{Action: rbacv3.RBAC_Action(action), Policies: make(map[string]*rbacv3.Policy, len(vs.Spec.RBAC.Policies))}
	for policyName, rawPolicy := range vs.Spec.RBAC.Policies {
		policy := &rbacv3.Policy{}
		if err := protoutil.Unmarshaler.Unmarshal(rawPolicy.Raw, policy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rbac policy %s: %w", policyName, err)
		}
		if err := policy.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate rbac policy %s: %w", policyName, err)
		}
		rules.Policies[policyName] = policy
	}

	for _, policyRef := range vs.Spec.RBAC.AdditionalPolicies {
		ns := helpers.GetNamespace(policyRef.Namespace, vs.Namespace)
		policy := store.GetPolicy(helpers.NamespacedName{Namespace: ns, Name: policyRef.Name})
		if policy == nil {
			return nil, fmt.Errorf("rbac policy %s/%s not found", ns, policyRef.Name)
		}
		if _, ok := rules.Policies[policy.Name]; ok {
			return nil, fmt.Errorf("policy '%s' already exist in RBAC", policy.Name)
		}
		rbacPolicy := &rbacv3.Policy{}
		if err := protoutil.Unmarshaler.Unmarshal(policy.Spec.Raw, rbacPolicy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rbac policy %s/%s: %w", ns, policyRef.Name, err)
		}
		if err := rbacPolicy.ValidateAll(); err != nil {
			return nil, fmt.Errorf("failed to validate rbac policy %s/%s: %w", ns, policyRef.Name, err)
		}
		rules.Policies[policy.Name] = rbacPolicy
	}

	return &rbacFilter.RBAC{Rules: rules}, nil
}

func buildFilterChains(params *FilterChainsParams) ([]*listenerv3.FilterChain, error) {
	var filterChains []*listenerv3.FilterChain

	if len(params.SecretNameToDomains) > 0 {
		for secretName, domains := range params.SecretNameToDomains {
			params.Domains = domains
			params.DownstreamTLSContext = &tlsv3.DownstreamTlsContext{
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
			fc, err := buildFilterChain(params)
			if err != nil {
				return nil, err
			}
			filterChains = append(filterChains, fc)
		}
		return filterChains, nil
	}

	fc, err := buildFilterChain(params)
	if err != nil {
		return nil, err
	}
	filterChains = append(filterChains, fc)
	return filterChains, nil
}

func buildFilterChain(params *FilterChainsParams) (*listenerv3.FilterChain, error) {
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
	if params.XFFNumTrustedHops != nil {
		httpConnectionManager.XffNumTrustedHops = *params.XFFNumTrustedHops
	}
	if params.AccessLog != nil {
		httpConnectionManager.AccessLog = append(httpConnectionManager.AccessLog, params.AccessLog)
	}

	if err := httpConnectionManager.ValidateAll(); err != nil {
		return nil, err
	}

	pbst, err := anypb.New(httpConnectionManager)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal httpConnectionManager to anypb: %w", err)
	}

	fc := &listenerv3.FilterChain{}
	fc.Filters = []*listenerv3.Filter{{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &listenerv3.Filter_TypedConfig{
			TypedConfig: pbst,
		},
	}}
	if params.IsTLS && len(params.Domains) > 0 && !slices.Contains(params.Domains, "*") {
		fc.FilterChainMatch = &listenerv3.FilterChainMatch{
			ServerNames: params.Domains,
		}
	}
	if params.DownstreamTLSContext != nil {
		scfg, err := anypb.New(params.DownstreamTLSContext)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal downstreamTlsContext to anypb, %w", err)
		}
		fc.TransportSocket = &corev3.TransportSocket{
			Name: "envoy.transport_sockets.tls",
			ConfigType: &corev3.TransportSocket_TypedConfig{
				TypedConfig: scfg,
			},
		}
	}
	fc.Name = params.VSName

	if err := fc.ValidateAll(); err != nil {
		return nil, err
	}

	return fc, nil
}

func buildUpgradeConfigs(rawUpgradeConfigs []*runtime.RawExtension) ([]*hcmv3.HttpConnectionManager_UpgradeConfig, error) {
	upgradeConfigs := make([]*hcmv3.HttpConnectionManager_UpgradeConfig, 0, len(rawUpgradeConfigs))
	for _, upgradeConfig := range rawUpgradeConfigs {
		uc := &hcmv3.HttpConnectionManager_UpgradeConfig{}
		if err := protoutil.Unmarshaler.Unmarshal(upgradeConfig.Raw, uc); err != nil {
			return upgradeConfigs, err
		}
		if err := uc.ValidateAll(); err != nil {
			return upgradeConfigs, err
		}
		upgradeConfigs = append(upgradeConfigs, uc)
	}

	return upgradeConfigs, nil
}

func buildAccessLogConfig(vs *v1alpha1.VirtualService, store *store.Store) (*accesslogv3.AccessLog, error) {
	if vs.Spec.AccessLog == nil && vs.Spec.AccessLogConfig == nil {
		return nil, nil
	}
	if vs.Spec.AccessLog != nil && vs.Spec.AccessLogConfig != nil {
		return nil, fmt.Errorf("can't use accessLog and accessLogConfig at the same time")
	}
	if vs.Spec.AccessLog != nil {
		var accessLog accesslogv3.AccessLog
		if err := protoutil.Unmarshaler.Unmarshal(vs.Spec.AccessLog.Raw, &accessLog); err != nil {
			return nil, fmt.Errorf("failed to unmarshal accessLog: %w", err)
		}
		if err := accessLog.ValidateAll(); err != nil {
			return nil, err
		}
		return &accessLog, nil
	}

	accessLogNs := helpers.GetNamespace(vs.Spec.AccessLogConfig.Namespace, vs.Namespace)
	accessLogConfig := store.GetAccessLog(helpers.NamespacedName{Namespace: accessLogNs, Name: vs.Spec.AccessLogConfig.Name})
	if accessLogConfig == nil {
		return nil, fmt.Errorf("can't find accessLogConfig %s/%s", accessLogNs, vs.Spec.AccessLogConfig.Name)
	}

	accessLog, err := accessLogConfig.UnmarshalAndValidateV3(v1alpha1.WithAccessLogFileName(vs.Name))
	if err != nil {
		return nil, err
	}

	return accessLog, nil
}

func getTLSType(vsTLSConfig *v1alpha1.TlsConfig) (string, error) {
	if vsTLSConfig == nil {
		return "", fmt.Errorf("tls config is empty")
	}
	if vsTLSConfig.SecretRef != nil {
		if vsTLSConfig.AutoDiscovery != nil {
			return "", fmt.Errorf("can't use secretRef and autoDiscovery at the same time")
		}
		return SecretRefType, nil
	}
	if vsTLSConfig.AutoDiscovery != nil {
		return AutoDiscoveryType, nil
	}
	return "", fmt.Errorf("tls config is empty")
}

func getSecretNameToDomainsViaSecretRef(secretRef *v1alpha1.ResourceRef, vsNamespace string, domains []string) map[helpers.NamespacedName][]string {
	m := make(map[helpers.NamespacedName][]string)

	var secretNamespace string

	if secretRef.Namespace != nil {
		secretNamespace = *secretRef.Namespace
	} else {
		secretNamespace = vsNamespace
	}

	m[helpers.NamespacedName{Namespace: secretNamespace, Name: secretRef.Name}] = domains
	return m
}

func getSecretNameToDomainsViaAutoDiscovery(domains []string, domainToSecretMap map[string]v1.Secret) (map[helpers.NamespacedName][]string, error) {
	m := make(map[helpers.NamespacedName][]string)

	for _, domain := range domains {
		var secret v1.Secret
		secret, ok := domainToSecretMap[domain]
		if !ok {
			secret, ok = domainToSecretMap[getWildcardDomain(domain)]
			if !ok {
				return nil, fmt.Errorf("can't find secret for domain %s", domain)
			}
		}

		domainsFromMap, ok := m[helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}]
		if ok {
			m[helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}] = append(domainsFromMap, domain)
		} else {
			m[helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}] = []string{domain}
		}
	}

	return m, nil
}

func getWildcardDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return ""
	}
	parts[0] = "*"
	return strings.Join(parts, ".")
}

func findClusterNames(data interface{}, fieldName string) []string {
	var results []string

	switch value := data.(type) {
	case map[string]interface{}:
		for k, v := range value {
			if k == fieldName {
				results = append(results, fmt.Sprintf("%v", v))
			}
			results = append(results, findClusterNames(v, fieldName)...)
		}
	case []interface{}:
		for _, item := range value {
			results = append(results, findClusterNames(item, fieldName)...)
		}
	}

	return results
}

func buildSecrets(httpFilters []*hcmv3.HttpFilter, secretNameToDomains map[helpers.NamespacedName][]string, store *store.Store) ([]*tlsv3.Secret, []helpers.NamespacedName, error) {
	var secrets []*tlsv3.Secret
	var usedSecrets []helpers.NamespacedName // for validation

	getEnvoySecret := func(namespace, name string) ([]*tlsv3.Secret, error) {
		kubeSecret := store.GetSecret(helpers.NamespacedName{Namespace: namespace, Name: name})
		if kubeSecret == nil {
			return nil, fmt.Errorf("can't find secret %s/%s", namespace, name)
		}
		usedSecrets = append(usedSecrets, helpers.NamespacedName{Namespace: namespace, Name: name})
		return makeEnvoySecretFromKubernetesSecret(kubeSecret)
	}

	// Get Secrets from certificatesWithDomains
	for secret := range secretNameToDomains {
		v3Secret, err := getEnvoySecret(secret.Namespace, secret.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("can't find envoy secret %s/%s", secret.Namespace, secret.Name)
		}
		secrets = append(secrets, v3Secret...)
	}

	for _, filter := range httpFilters {
		jsonData, err := json.MarshalIndent(filter, "", "  ")
		if err != nil {
			return nil, nil, err
		}

		var data interface{}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return nil, nil, err
		}

		fieldName := "sds_config"
		secretNames := findSDSNames(data, fieldName)

		for _, secretName := range secretNames {
			namespace, name, err := helpers.SplitNamespacedName(secretName)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to split secret name: %v", err)
			}

			v3Secret, err := getEnvoySecret(namespace, name)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get envoy secret: %v", err)
			}

			secrets = append(secrets, v3Secret...)
		}
	}

	return secrets, usedSecrets, nil
}

func findSDSNames(data interface{}, fieldName string) []string {
	var results []string

	switch value := data.(type) {
	case map[string]interface{}:
		for k, v := range value {
			if k == fieldName {
				results = append(results, fmt.Sprintf("%v", value["name"]))
			}
			results = append(results, findSDSNames(v, fieldName)...)
		}
	case []interface{}:
		for _, item := range value {
			results = append(results, findSDSNames(item, fieldName)...)
		}
	}

	return results
}

func makeEnvoySecretFromKubernetesSecret(kubeSecret *v1.Secret) ([]*tlsv3.Secret, error) {
	switch kubeSecret.Type {
	case v1.SecretTypeTLS:
		return makeEnvoyTLSSecret(kubeSecret)
	case v1.SecretTypeOpaque:
		return makeEnvoyOpaqueSecret(kubeSecret)
	default:
		return nil, fmt.Errorf("unsupported secret type %s", kubeSecret.Type)
	}
}

func makeEnvoyTLSSecret(kubeSecret *v1.Secret) ([]*tlsv3.Secret, error) {
	secrets := make([]*tlsv3.Secret, 0)

	envoySecret := &tlsv3.Secret{
		Name: fmt.Sprintf("%s/%s", kubeSecret.Namespace, kubeSecret.Name),
		Type: &tlsv3.Secret_TlsCertificate{
			TlsCertificate: &tlsv3.TlsCertificate{
				CertificateChain: &corev3.DataSource{
					Specifier: &corev3.DataSource_InlineBytes{
						InlineBytes: kubeSecret.Data[v1.TLSCertKey],
					},
				},
				PrivateKey: &corev3.DataSource{
					Specifier: &corev3.DataSource_InlineBytes{
						InlineBytes: kubeSecret.Data[v1.TLSPrivateKeyKey],
					},
				},
			},
		},
	}
	if err := envoySecret.ValidateAll(); err != nil {
		return nil, fmt.Errorf("failed to validate tls secret: %w", err)
	}

	secrets = append(secrets, envoySecret)

	return secrets, nil
}

func makeEnvoyOpaqueSecret(kubeSecret *v1.Secret) ([]*tlsv3.Secret, error) {
	secrets := make([]*tlsv3.Secret, 0)

	for k, v := range kubeSecret.Data {
		envoySecret := &tlsv3.Secret{
			Name: fmt.Sprintf("%s/%s/%s", kubeSecret.Namespace, kubeSecret.Name, k),
			Type: &tlsv3.Secret_GenericSecret{
				GenericSecret: &tlsv3.GenericSecret{
					Secret: &corev3.DataSource{
						Specifier: &corev3.DataSource_InlineBytes{
							InlineBytes: v,
						},
					},
				},
			},
		}

		if err := envoySecret.ValidateAll(); err != nil {
			return nil, fmt.Errorf("cannot validate Envoy Secret: %w", err)
		}

		secrets = append(secrets, envoySecret)
	}

	return secrets, nil
}

func isTLSListener(xdsListener *listenerv3.Listener) bool {
	if xdsListener == nil {
		return false
	}
	if len(xdsListener.ListenerFilters) == 0 {
		return false
	}
	for _, lFilter := range xdsListener.ListenerFilters {
		if tc := lFilter.GetTypedConfig(); tc != nil {
			if tc.TypeUrl == "type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector" {
				return true
			}
		}
	}
	return false
}

func checkAllDomainsUnique(domains []string) error {
	seen := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		if _, exists := seen[domain]; exists {
			return fmt.Errorf("duplicate domain found: %s", domain)
		}
		seen[domain] = struct{}{}
	}
	return nil
}
