package clusters

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	oauth2v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/oauth2/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/utils"
	"k8s.io/apimachinery/pkg/runtime"
)

// Builder handles the construction of Envoy clusters from various sources
type Builder struct {
	cache *cache
	store store.Store
}

// NewBuilder creates a new cluster builder with caching enabled
func NewBuilder(store store.Store) *Builder {
	return &Builder{
		cache: newCache(),
		store: store,
	}
}

// FromVirtualHostRoutes extracts clusters referenced by routes inside the given VirtualHost
func (b *Builder) FromVirtualHostRoutes(virtualHost *routev3.VirtualHost) ([]*cluster.Cluster, error) {
	var clusterNames []string

	// Direct traversal of route structures instead of JSON marshaling
	for _, route := range virtualHost.Routes {
		names := extractClusterNamesFromRoute(route)
		clusterNames = append(clusterNames, names...)
	}

	if len(clusterNames) == 0 {
		return nil, nil
	}

	return b.getClustersByNames(clusterNames)
}

// extractClusterNamesFromRoute directly extracts cluster names from route configuration
func extractClusterNamesFromRoute(route *routev3.Route) []string {
	var names []string

	if route.Action == nil {
		return names
	}

	switch action := route.Action.(type) {
	case *routev3.Route_Route:
		if action.Route == nil {
			break
		}
		switch cluster := action.Route.ClusterSpecifier.(type) {
		case *routev3.RouteAction_Cluster:
			if cluster.Cluster != "" {
				names = append(names, cluster.Cluster)
			}
		case *routev3.RouteAction_WeightedClusters:
			if cluster.WeightedClusters != nil {
				for _, wc := range cluster.WeightedClusters.Clusters {
					if wc.Name != "" {
						names = append(names, wc.Name)
					}
				}
			}
		}
	case *routev3.Route_DirectResponse:
		// Direct responses don't reference clusters
	case *routev3.Route_Redirect:
		// Redirects don't reference clusters
	}

	return names
}

// FromOAuth2HTTPFilters extracts clusters referenced by OAuth2 HTTP filters (token/authorize/etc)
func (b *Builder) FromOAuth2HTTPFilters(httpFilters []*hcmv3.HttpFilter) ([]*cluster.Cluster, error) {
	// Check cache first
	cacheKey := b.generateOAuth2CacheKey(httpFilters)
	if cached, exists := b.cache.get(cacheKey); exists {
		return cached, nil
	}

	var clusters []*cluster.Cluster
	for _, httpFilter := range httpFilters {
		tc := httpFilter.GetTypedConfig()
		if tc == nil || tc.TypeUrl != utils.TypeURLOAuth2 {
			continue
		}
		var oauthCfg oauth2v3.OAuth2
		if err := tc.UnmarshalTo(&oauthCfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal oauth2 config: %w", err)
		}
		jsonData, err := json.Marshal(oauthCfg.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal oauth2 config: %w", err)
		}
		var data interface{}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal oauth2 config data: %w", err)
		}
		names := utils.FindClusterNames(data, "Cluster")
		cl, err := b.getClustersByNames(names)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, cl...)
	}

	// Store result in cache before returning
	b.cache.set(cacheKey, clusters)

	return clusters, nil
}

// FromTracingRaw extracts clusters referenced by inline tracing configuration
func (b *Builder) FromTracingRaw(tr *runtime.RawExtension) ([]*cluster.Cluster, error) {
	if tr == nil {
		return nil, nil
	}

	// Check cache first
	cacheKey := b.generateTracingRawCacheKey(tr)
	if cached, exists := b.cache.get(cacheKey); exists {
		return cached, nil
	}

	// Direct parsing of RawExtension data instead of JSON marshal/unmarshal roundtrip
	var data interface{}
	if tr.Raw != nil {
		if err := json.Unmarshal(tr.Raw, &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tracing raw data: %w", err)
		}
	} else if tr.Object != nil {
		// Handle structured object case if needed
		jsonData, err := json.Marshal(tr.Object)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tracing object: %w", err)
		}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tracing object data: %w", err)
		}
	} else {
		return nil, nil
	}

	var clusters []*cluster.Cluster
	var allNames []string

	// opentelemetry/zipkin use different field names
	names := utils.FindClusterNames(data, "cluster_name")
	allNames = append(allNames, names...)

	names = utils.FindClusterNames(data, "collector_cluster") // zipkin
	allNames = append(allNames, names...)

	if len(allNames) > 0 {
		cl, err := b.getClustersByNames(allNames)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, cl...)
	}

	// Store result in cache before returning
	b.cache.set(cacheKey, clusters)

	return clusters, nil
}

// FromTracingRef extracts clusters referenced by a Tracing resource referenced from VS
func (b *Builder) FromTracingRef(vs *v1alpha1.VirtualService) ([]*cluster.Cluster, error) {
	if vs.Spec.TracingRef == nil {
		return nil, nil
	}

	// Check cache first
	cacheKey := b.generateTracingRefCacheKey(vs)
	if cached, exists := b.cache.get(cacheKey); exists {
		return cached, nil
	}

	tracingRefNs := helpers.GetNamespace(vs.Spec.TracingRef.Namespace, vs.Namespace)
	tracing := b.store.GetTracing(helpers.NamespacedName{Namespace: tracingRefNs, Name: vs.Spec.TracingRef.Name})
	if tracing == nil {
		return nil, fmt.Errorf("tracing %s/%s not found", tracingRefNs, vs.Spec.TracingRef.Name)
	}

	// Direct parsing of tracing spec instead of JSON marshal/unmarshal roundtrip
	var data interface{}
	if tracing.Spec != nil {
		if tracing.Spec.Raw != nil {
			if err := json.Unmarshal(tracing.Spec.Raw, &data); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tracing spec raw: %w", err)
			}
		} else if tracing.Spec.Object != nil {
			// Handle structured object case if needed
			jsonData, err := json.Marshal(tracing.Spec.Object)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tracing spec object: %w", err)
			}
			if err := json.Unmarshal(jsonData, &data); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tracing spec object data: %w", err)
			}
		} else {
			return nil, nil
		}
	} else {
		return nil, nil
	}

	var clusters []*cluster.Cluster
	var allNames []string

	// opentelemetry/zipkin use different field names
	names := utils.FindClusterNames(data, "cluster_name")
	allNames = append(allNames, names...)

	names = utils.FindClusterNames(data, "collector_cluster") // zipkin
	allNames = append(allNames, names...)

	if len(allNames) > 0 {
		cl, err := b.getClustersByNames(allNames)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, cl...)
	}

	// Store result in cache before returning
	b.cache.set(cacheKey, clusters)

	return clusters, nil
}

// getClustersByNames resolves cluster specs by names and validates them
func (b *Builder) getClustersByNames(names []string) ([]*cluster.Cluster, error) {
	clusters := make([]*cluster.Cluster, 0, len(names))
	for _, clusterName := range names {
		cl := b.store.GetSpecCluster(clusterName)
		if cl == nil {
			return nil, fmt.Errorf("cluster %s not found", clusterName)
		}
		xdsCluster, err := cl.UnmarshalV3AndValidate()
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal cluster %s: %w", clusterName, err)
		}
		clusters = append(clusters, xdsCluster)
	}
	return clusters, nil
}

// generateOAuth2CacheKey creates a cache key for OAuth2 HTTP filters
// It includes cluster generations to invalidate cache when referenced clusters change
func (b *Builder) generateOAuth2CacheKey(httpFilters []*hcmv3.HttpFilter) string {
	hasher := sha256.New()

	var allClusterNames []string
	for _, httpFilter := range httpFilters {
		tc := httpFilter.GetTypedConfig()
		if tc == nil || tc.TypeUrl != utils.TypeURLOAuth2 {
			continue
		}
		// Use the raw TypedConfig data for consistent hashing
		hasher.Write(tc.Value)

		// Extract cluster names from OAuth2 config for generation tracking
		var oauthCfg oauth2v3.OAuth2
		if err := tc.UnmarshalTo(&oauthCfg); err == nil {
			if jsonData, err := json.Marshal(oauthCfg.Config); err == nil {
				var data interface{}
				if err := json.Unmarshal(jsonData, &data); err == nil {
					allClusterNames = append(allClusterNames, utils.FindClusterNames(data, "Cluster")...)
				}
			}
		}
	}

	// Include referenced cluster generations in cache key (BUG-004 fix)
	b.writeClusterGenerations(hasher, allClusterNames)

	return fmt.Sprintf("oauth2_%x", hasher.Sum(nil))
}

// generateTracingRawCacheKey creates a cache key for inline tracing configuration
// It includes cluster generations to invalidate cache when referenced clusters change
func (b *Builder) generateTracingRawCacheKey(tr *runtime.RawExtension) string {
	if tr == nil {
		return "tracing_raw_nil"
	}

	hasher := sha256.New()

	if tr.Raw != nil {
		hasher.Write(tr.Raw)
	} else if tr.Object != nil {
		// Handle structured object case
		if jsonData, err := json.Marshal(tr.Object); err == nil {
			hasher.Write(jsonData)
		}
	}

	// Include referenced cluster generations in cache key (BUG-004 fix)
	clusterNames := b.extractClusterNamesFromTracing(tr)
	b.writeClusterGenerations(hasher, clusterNames)

	return fmt.Sprintf("tracing_raw_%x", hasher.Sum(nil))
}

// generateTracingRefCacheKey creates a cache key for tracing reference configuration
// It includes cluster generations to invalidate cache when referenced clusters change
func (b *Builder) generateTracingRefCacheKey(vs *v1alpha1.VirtualService) string {
	if vs.Spec.TracingRef == nil {
		return "tracing_ref_nil"
	}

	tracingRefNs := helpers.GetNamespace(vs.Spec.TracingRef.Namespace, vs.Namespace)
	hasher := sha256.New()

	// Include the reference path in the key
	hasher.Write([]byte(fmt.Sprintf("%s/%s", tracingRefNs, vs.Spec.TracingRef.Name)))

	// Include the actual tracing content if available
	tracingNN := helpers.NamespacedName{Namespace: tracingRefNs, Name: vs.Spec.TracingRef.Name}
	tracing := b.store.GetTracing(tracingNN)
	if tracing != nil {
		if tracing.Spec != nil {
			if tracing.Spec.Raw != nil {
				hasher.Write(tracing.Spec.Raw)
			} else if tracing.Spec.Object != nil {
				if jsonData, err := json.Marshal(tracing.Spec.Object); err == nil {
					hasher.Write(jsonData)
				}
			}
		}

		// Include referenced cluster generations in cache key (BUG-004 fix)
		clusterNames := b.extractClusterNamesFromTracing(tracing.Spec)
		b.writeClusterGenerations(hasher, clusterNames)
	}

	return fmt.Sprintf("tracing_ref_%x", hasher.Sum(nil))
}

// extractClusterNamesFromTracing extracts cluster names from tracing configuration without resolving
func (b *Builder) extractClusterNamesFromTracing(spec *runtime.RawExtension) []string {
	if spec == nil {
		return nil
	}

	var data interface{}
	if spec.Raw != nil {
		if err := json.Unmarshal(spec.Raw, &data); err != nil {
			return nil
		}
	} else if spec.Object != nil {
		jsonData, err := json.Marshal(spec.Object)
		if err != nil {
			return nil
		}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return nil
		}
	} else {
		return nil
	}

	var names []string
	names = append(names, utils.FindClusterNames(data, "cluster_name")...)
	names = append(names, utils.FindClusterNames(data, "collector_cluster")...)
	return names
}

// writeClusterGenerations writes cluster generations to hasher for cache key
func (b *Builder) writeClusterGenerations(hasher interface{ Write([]byte) (int, error) }, clusterNames []string) {
	for _, name := range clusterNames {
		if cl := b.store.GetSpecCluster(name); cl != nil {
			_, _ = hasher.Write([]byte(fmt.Sprintf("%s:%d,", name, cl.Generation)))
		}
	}
}

// ClusterExtractor interface implementation
// These methods implement the interfaces.ClusterExtractor interface

// ExtractClustersFromFilterChains extracts clusters from filter chains
func (b *Builder) ExtractClustersFromFilterChains(filterChains []*listenerv3.FilterChain) ([]*cluster.Cluster, error) {
	return ExtractClustersFromFilterChains(filterChains, b.store)
}

// ExtractClustersFromVirtualHost extracts clusters from a virtual host
func (b *Builder) ExtractClustersFromVirtualHost(virtualHost *routev3.VirtualHost) ([]*cluster.Cluster, error) {
	return b.FromVirtualHostRoutes(virtualHost)
}

// ExtractClustersFromHTTPFilters extracts clusters from HTTP filters
func (b *Builder) ExtractClustersFromHTTPFilters(httpFilters []*hcmv3.HttpFilter) ([]*cluster.Cluster, error) {
	return b.FromOAuth2HTTPFilters(httpFilters)
}

// ExtractClustersFromTracingRaw extracts clusters from inline tracing configuration
func (b *Builder) ExtractClustersFromTracingRaw(tr *runtime.RawExtension) ([]*cluster.Cluster, error) {
	return b.FromTracingRaw(tr)
}

// ExtractClustersFromTracingRef extracts clusters from tracing reference
func (b *Builder) ExtractClustersFromTracingRef(vs *v1alpha1.VirtualService) ([]*cluster.Cluster, error) {
	return b.FromTracingRef(vs)
}
