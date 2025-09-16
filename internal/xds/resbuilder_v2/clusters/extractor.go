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
)

// ExtractClustersFromFilterChains extracts all cluster references from listener filter chains
// This function provides optimized cluster extraction from existing listener configurations
func ExtractClustersFromFilterChains(filterChains []*listenerv3.FilterChain, store *store.Store) ([]*cluster.Cluster, error) {
	var clusterNames []string
	
	for _, filterChain := range filterChains {
		for _, filter := range filterChain.Filters {
			names, err := extractClustersFromFilter(filter)
			if err != nil {
				return nil, fmt.Errorf("failed to extract clusters from filter %s: %w", filter.Name, err)
			}
			clusterNames = append(clusterNames, names...)
		}
	}
	
	if len(clusterNames) == 0 {
		return nil, nil
	}
	
	// Remove duplicates
	uniqueNames := removeDuplicateStrings(clusterNames)
	
	// Resolve clusters from store
	clusters := make([]*cluster.Cluster, 0, len(uniqueNames))
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
	
	return clusters, nil
}

// extractClustersFromFilter extracts cluster names from a specific listener filter
func extractClustersFromFilter(filter *listenerv3.Filter) ([]string, error) {
	var names []string
	
	switch filter.Name {
	case wellknown.HTTPConnectionManager:
		hcmNames, err := extractClustersFromHCMFilter(filter)
		if err != nil {
			return nil, err
		}
		names = append(names, hcmNames...)
		
	case wellknown.TCPProxy:
		tcpNames, err := extractClustersFromTCPProxyFilter(filter)
		if err != nil {
			return nil, err
		}
		names = append(names, tcpNames...)
		
	default:
		// For other filter types, try generic JSON-based extraction as fallback
		genericNames, err := extractClustersFromGenericFilter(filter)
		if err != nil {
			return nil, err
		}
		names = append(names, genericNames...)
	}
	
	return names, nil
}

// extractClustersFromHCMFilter extracts cluster names from HTTP Connection Manager filter
func extractClustersFromHCMFilter(filter *listenerv3.Filter) ([]string, error) {
	var names []string
	
	typedConfig := filter.GetTypedConfig()
	if typedConfig == nil {
		return names, nil
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
	
	return names, nil
}

// extractClustersFromTCPProxyFilter extracts cluster names from TCP Proxy filter
func extractClustersFromTCPProxyFilter(filter *listenerv3.Filter) ([]string, error) {
	var names []string
	
	typedConfig := filter.GetTypedConfig()
	if typedConfig == nil {
		return names, nil
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
	
	return names, nil
}

// extractClustersFromGenericFilter provides fallback JSON-based extraction for unknown filter types
func extractClustersFromGenericFilter(filter *listenerv3.Filter) ([]string, error) {
	typedConfig := filter.GetTypedConfig()
	if typedConfig == nil {
		return nil, nil
	}
	
	// Convert to generic data structure for searching
	var data interface{}
	if err := json.Unmarshal(typedConfig.Value, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal filter config: %w", err)
	}
	
	// Search for common cluster field names
	var allNames []string
	commonClusterFields := []string{"cluster", "cluster_name", "collector_cluster"}
	
	for _, fieldName := range commonClusterFields {
		names := findClusterNames(data, fieldName)
		allNames = append(allNames, names...)
	}
	
	return removeDuplicateStrings(allNames), nil
}

// extractClustersFromHTTPFilter extracts cluster names from HTTP filters (OAuth2, etc.)
func extractClustersFromHTTPFilter(httpFilter *hcmv3.HttpFilter) []string {
	var names []string
	
	typedConfig := httpFilter.GetTypedConfig()
	if typedConfig == nil {
		return names
	}
	
	// Convert to generic data structure for searching
	var data interface{}
	if err := json.Unmarshal(typedConfig.Value, &data); err != nil {
		return names
	}
	
	// Search for cluster references in various formats
	clusterFields := []string{"cluster", "cluster_name", "token_cluster", "authorization_cluster"}
	for _, field := range clusterFields {
		fieldNames := findClusterNames(data, field)
		names = append(names, fieldNames...)
	}
	
	return names
}

// extractClustersFromAccessLog extracts cluster names from access log configurations
func extractClustersFromAccessLog(accessLog *accesslogv3.AccessLog) ([]string, error) {
	var names []string
	
	typedConfig := accessLog.GetTypedConfig()
	if typedConfig == nil {
		return names, nil
	}
	
	// Convert to generic data structure for searching
	var data interface{}
	if err := json.Unmarshal(typedConfig.Value, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal access log config: %w", err)
	}
	
	// Search for cluster references (e.g., in HTTP grpc access log service)
	clusterFields := []string{"cluster_name", "cluster"}
	for _, field := range clusterFields {
		fieldNames := findClusterNames(data, field)
		names = append(names, fieldNames...)
	}
	
	return names, nil
}

// extractClustersFromTracing extracts cluster names from tracing configuration
func extractClustersFromTracing(tracing *hcmv3.HttpConnectionManager_Tracing) ([]string, error) {
	var names []string
	
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
		fieldNames := findClusterNames(data, field)
		names = append(names, fieldNames...)
	}
	
	return names, nil
}

// removeDuplicateStrings removes duplicate entries from a string slice
func removeDuplicateStrings(strings []string) []string {
	if len(strings) == 0 {
		return strings
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