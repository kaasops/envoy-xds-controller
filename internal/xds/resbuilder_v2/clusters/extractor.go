package clusters

import (
	"encoding/json"
	"fmt"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcpProxyv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/utils"
)

// ExtractClustersFromFilterChains extracts all cluster references from listener filter chains
// This function provides optimized cluster extraction from existing listener configurations
func ExtractClustersFromFilterChains(filterChains []*listenerv3.FilterChain, store *store.Store) ([]*cluster.Cluster, error) {
	// Use pooled string slice for cluster names
	namesPtr := utils.GetStringSlice()
	defer utils.PutStringSlice(namesPtr)
	names := *namesPtr

	// Extract cluster names from all filter chains
	for _, filterChain := range filterChains {
		for _, filter := range filterChain.Filters {
			filterNames, err := extractClustersFromFilter(filter)
			if err != nil {
				return nil, fmt.Errorf("failed to extract clusters from filter %s: %w", filter.Name, err)
			}
			names = append(names, filterNames...)
		}
	}

	if len(names) == 0 {
		return nil, nil
	}

	// Remove duplicates
	uniqueNames := removeDuplicateStrings(names)

	// Use pooled cluster slice for results
	clustersPtr := utils.GetClusterSlice()
	defer utils.PutClusterSlice(clustersPtr)
	clusters := *clustersPtr

	// Resolve clusters from store
	for _, clusterName := range uniqueNames {
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

	// Create a new slice to return to the caller
	// We can't return the pooled slice directly as it would be reused
	result := make([]*cluster.Cluster, len(clusters))
	copy(result, clusters)

	return result, nil
}

// extractClustersFromFilter extracts cluster names from a specific listener filter
func extractClustersFromFilter(filter *listenerv3.Filter) ([]string, error) {
	// Use pooled string slice for results
	namesPtr := utils.GetStringSlice()
	defer utils.PutStringSlice(namesPtr)
	// We're not using the pooled slice directly in this function,
	// but we still need to acquire and release it properly

	var err error
	var extractedNames []string

	switch filter.Name {
	case wellknown.HTTPConnectionManager:
		extractedNames, err = extractClustersFromHCMFilter(filter)
		if err != nil {
			return nil, err
		}

	case wellknown.TCPProxy:
		extractedNames, err = extractClustersFromTCPProxyFilter(filter)
		if err != nil {
			return nil, err
		}

	default:
		// For other filter types, try generic JSON-based extraction as fallback
		extractedNames, err = extractClustersFromGenericFilter(filter)
		if err != nil {
			return nil, err
		}
	}

	// Create a new slice to return to the caller
	result := make([]string, len(extractedNames))
	copy(result, extractedNames)

	return result, nil
}

// extractClustersFromHCMFilter extracts cluster names from HTTP Connection Manager filter
func extractClustersFromHCMFilter(filter *listenerv3.Filter) ([]string, error) {
	// Use pooled string slice for results
	namesPtr := utils.GetStringSlice()
	defer utils.PutStringSlice(namesPtr)
	names := *namesPtr

	typedConfig := filter.GetTypedConfig()
	if typedConfig == nil {
		return make([]string, 0), nil
	}

	var hcm hcmv3.HttpConnectionManager
	if err := typedConfig.UnmarshalTo(&hcm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal HCM config: %w", err)
	}

	// Extract clusters from HTTP filters (OAuth2, etc.)
	for _, httpFilter := range hcm.HttpFilters {
		filterNames := extractClustersFromHTTPFilter(httpFilter)
		names = append(names, filterNames...)
	}

	// Extract clusters from access logs
	for _, accessLog := range hcm.AccessLog {
		logNames, err := extractClustersFromAccessLog(accessLog)
		if err != nil {
			return nil, err
		}
		names = append(names, logNames...)
	}

	// Extract clusters from tracing configuration
	if hcm.Tracing != nil {
		tracingNames, err := extractClustersFromTracing(hcm.Tracing)
		if err != nil {
			return nil, err
		}
		names = append(names, tracingNames...)
	}

	// Create a new slice to return to the caller
	result := make([]string, len(names))
	copy(result, names)

	return result, nil
}

// extractClustersFromTCPProxyFilter extracts cluster names from TCP Proxy filter
func extractClustersFromTCPProxyFilter(filter *listenerv3.Filter) ([]string, error) {
	// Use pooled string slice for results
	namesPtr := utils.GetStringSlice()
	defer utils.PutStringSlice(namesPtr)
	names := *namesPtr

	typedConfig := filter.GetTypedConfig()
	if typedConfig == nil {
		return make([]string, 0), nil
	}

	var tcpProxy tcpProxyv3.TcpProxy
	if err := typedConfig.UnmarshalTo(&tcpProxy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TCP proxy config: %w", err)
	}

	// Extract cluster from cluster specifier
	switch clusterSpec := tcpProxy.ClusterSpecifier.(type) {
	case *tcpProxyv3.TcpProxy_Cluster:
		if clusterSpec.Cluster != "" {
			names = append(names, clusterSpec.Cluster)
		}
	case *tcpProxyv3.TcpProxy_WeightedClusters:
		if clusterSpec.WeightedClusters != nil {
			for _, wc := range clusterSpec.WeightedClusters.Clusters {
				if wc.Name != "" {
					names = append(names, wc.Name)
				}
			}
		}
	}

	// Create a new slice to return to the caller
	result := make([]string, len(names))
	copy(result, names)

	return result, nil
}

// extractClustersFromGenericFilter provides fallback JSON-based extraction for unknown filter types
func extractClustersFromGenericFilter(filter *listenerv3.Filter) ([]string, error) {
	typedConfig := filter.GetTypedConfig()
	if typedConfig == nil {
		return make([]string, 0), nil
	}

	// Convert to generic data structure for searching
	var data interface{}
	if err := json.Unmarshal(typedConfig.Value, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal filter config: %w", err)
	}

	// Search for common cluster field names
	// Use pooled string slice for collecting names
	allNamesPtr := utils.GetStringSlice()
	defer utils.PutStringSlice(allNamesPtr)
	allNames := *allNamesPtr

	commonClusterFields := []string{"cluster", "cluster_name", "collector_cluster"}

	for _, fieldName := range commonClusterFields {
		fieldNames := utils.FindClusterNames(data, fieldName)
		allNames = append(allNames, fieldNames...)
	}

	// Remove duplicates and create a new slice to return
	uniqueNames := removeDuplicateStrings(allNames)

	// Create a new slice to return to the caller
	result := make([]string, len(uniqueNames))
	copy(result, uniqueNames)

	return result, nil
}

// extractClustersFromHTTPFilter extracts cluster names from HTTP filters (OAuth2, etc.)
func extractClustersFromHTTPFilter(httpFilter *hcmv3.HttpFilter) []string {
	// Use pooled string slice for results
	namesPtr := utils.GetStringSlice()
	defer utils.PutStringSlice(namesPtr)
	names := *namesPtr

	typedConfig := httpFilter.GetTypedConfig()
	if typedConfig == nil {
		return make([]string, 0)
	}

	// Convert to generic data structure for searching
	var data interface{}
	if err := json.Unmarshal(typedConfig.Value, &data); err != nil {
		return make([]string, 0)
	}

	// Search for cluster references in various formats
	clusterFields := []string{"cluster", "cluster_name", "token_cluster", "authorization_cluster"}
	for _, field := range clusterFields {
		fieldNames := utils.FindClusterNames(data, field)
		names = append(names, fieldNames...)
	}

	// Create a new slice to return to the caller
	result := make([]string, len(names))
	copy(result, names)

	return result
}

// extractClustersFromAccessLog extracts cluster names from access log configurations
func extractClustersFromAccessLog(accessLog *accesslogv3.AccessLog) ([]string, error) {
	// Use pooled string slice for results
	namesPtr := utils.GetStringSlice()
	defer utils.PutStringSlice(namesPtr)
	names := *namesPtr

	typedConfig := accessLog.GetTypedConfig()
	if typedConfig == nil {
		return make([]string, 0), nil
	}

	// Convert to generic data structure for searching
	var data interface{}
	if err := json.Unmarshal(typedConfig.Value, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal access log config: %w", err)
	}

	// Search for cluster references (e.g., in HTTP grpc access log service)
	clusterFields := []string{"cluster_name", "cluster"}
	for _, field := range clusterFields {
		fieldNames := utils.FindClusterNames(data, field)
		names = append(names, fieldNames...)
	}

	// Create a new slice to return to the caller
	result := make([]string, len(names))
	copy(result, names)

	return result, nil
}

// extractClustersFromTracing extracts cluster names from tracing configuration
func extractClustersFromTracing(tracing *hcmv3.HttpConnectionManager_Tracing) ([]string, error) {
	// Use pooled string slice for results
	namesPtr := utils.GetStringSlice()
	defer utils.PutStringSlice(namesPtr)
	names := *namesPtr

	// Convert tracing config to generic data for searching
	jsonData, err := json.Marshal(tracing)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tracing config: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tracing config: %w", err)
	}

	// Search for cluster references in tracing providers
	clusterFields := []string{"cluster_name", "collector_cluster"}
	for _, field := range clusterFields {
		fieldNames := utils.FindClusterNames(data, field)
		names = append(names, fieldNames...)
	}

	// Create a new slice to return to the caller
	result := make([]string, len(names))
	copy(result, names)

	return result, nil
}

// removeDuplicateStrings removes duplicate entries from a string slice
func removeDuplicateStrings(strings []string) []string {
	if len(strings) == 0 {
		return make([]string, 0)
	}

	seen := make(map[string]struct{}, len(strings))
	result := make([]string, 0, len(strings))

	for _, s := range strings {
		if _, exists := seen[s]; !exists && s != "" {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}

	return result
}
