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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"k8s.io/klog/v2"
)

// ExtractClustersFromFilterChains extracts all cluster references from listener filter chains
// This function provides optimized cluster extraction from existing listener configurations
func ExtractClustersFromFilterChains(filterChains []*listenerv3.FilterChain, store store.Store) ([]*cluster.Cluster, error) {
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

// extractClustersFromFilter extracts cluster names from a specific listener filter.
// Note: The returned slice is safe to use directly - no copy needed here since
// the final copy is made in ExtractClustersFromFilterChains before returning to caller.
func extractClustersFromFilter(filter *listenerv3.Filter) ([]string, error) {
	switch filter.Name {
	case wellknown.HTTPConnectionManager:
		return extractClustersFromHCMFilter(filter)

	case wellknown.TCPProxy:
		return extractClustersFromTCPProxyFilter(filter)

	default:
		// For other filter types, try generic JSON-based extraction as fallback
		return extractClustersFromGenericFilter(filter)
	}
}

// extractClustersFromHCMFilter extracts cluster names from HTTP Connection Manager filter.
// Note: Returns slice directly without copying - caller handles final copy.
func extractClustersFromHCMFilter(filter *listenerv3.Filter) ([]string, error) {
	typedConfig := filter.GetTypedConfig()
	if typedConfig == nil {
		return nil, nil
	}

	var hcm hcmv3.HttpConnectionManager
	if err := typedConfig.UnmarshalTo(&hcm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal HCM config: %w", err)
	}

	// Pre-allocate with estimated capacity
	names := make([]string, 0, len(hcm.HttpFilters)+len(hcm.AccessLog)+2)

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

// extractClustersFromTCPProxyFilter extracts cluster names from TCP Proxy filter.
// Note: Returns slice directly without copying - caller handles final copy.
func extractClustersFromTCPProxyFilter(filter *listenerv3.Filter) ([]string, error) {
	typedConfig := filter.GetTypedConfig()
	if typedConfig == nil {
		return nil, nil
	}

	var tcpProxy tcpProxyv3.TcpProxy
	if err := typedConfig.UnmarshalTo(&tcpProxy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TCP proxy config: %w", err)
	}

	// Extract cluster from cluster specifier
	switch clusterSpec := tcpProxy.ClusterSpecifier.(type) {
	case *tcpProxyv3.TcpProxy_Cluster:
		if clusterSpec.Cluster != "" {
			return []string{clusterSpec.Cluster}, nil
		}
	case *tcpProxyv3.TcpProxy_WeightedClusters:
		if clusterSpec.WeightedClusters != nil {
			names := make([]string, 0, len(clusterSpec.WeightedClusters.Clusters))
			for _, wc := range clusterSpec.WeightedClusters.Clusters {
				if wc.Name != "" {
					names = append(names, wc.Name)
				}
			}
			return names, nil
		}
	}

	return nil, nil
}

// extractClustersFromGenericFilter provides fallback JSON-based extraction for unknown filter types.
// Note: Returns slice directly without copying - caller handles final copy.
func extractClustersFromGenericFilter(filter *listenerv3.Filter) ([]string, error) {
	typedConfig := filter.GetTypedConfig()
	if typedConfig == nil {
		return nil, nil
	}

	// Convert protobuf Any to JSON using protojson
	// This properly handles protobuf-encoded data in the Any message
	msg, err := anypb.UnmarshalNew(typedConfig, proto.UnmarshalOptions{})
	if err != nil {
		// If we can't unmarshal the message (e.g., unknown type), skip cluster extraction.
		// This is expected for filters not registered in go-control-plane.
		klog.V(4).InfoS("cannot unmarshal filter config, skipping cluster extraction",
			"filter", filter.Name,
			"typeUrl", typedConfig.TypeUrl,
			"error", err.Error())
		return nil, nil
	}

	// Marshal the protobuf message to JSON using proto field names (snake_case)
	// to ensure consistent field name matching in FindClusterNames
	jsonBytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filter config to JSON: %w", err)
	}

	// Convert to generic data structure for searching
	var data interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Search for common cluster field names
	commonClusterFields := []string{"cluster", "cluster_name", "collector_cluster"}
	allNames := make([]string, 0, 4)

	for _, fieldName := range commonClusterFields {
		fieldNames := utils.FindClusterNames(data, fieldName)
		allNames = append(allNames, fieldNames...)
	}

	// Remove duplicates - this creates a new slice
	return removeDuplicateStrings(allNames), nil
}

// extractClustersFromHTTPFilter extracts cluster names from HTTP filters (OAuth2, etc.).
// Note: Returns slice directly without copying - caller handles final copy.
func extractClustersFromHTTPFilter(httpFilter *hcmv3.HttpFilter) []string {
	typedConfig := httpFilter.GetTypedConfig()
	if typedConfig == nil {
		return nil
	}

	// typedConfig is *anypb.Any containing serialized protobuf, not JSON
	// We need to unmarshal it properly using protojson
	msg, err := anypb.UnmarshalNew(typedConfig, proto.UnmarshalOptions{})
	if err != nil {
		// Unknown filter type - skip cluster extraction
		klog.V(4).InfoS("cannot unmarshal HTTP filter config, skipping cluster extraction",
			"filter", httpFilter.Name,
			"typeUrl", typedConfig.TypeUrl,
			"error", err.Error())
		return nil
	}

	// Use proto field names (snake_case) to ensure consistent field name matching
	jsonBytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		klog.V(4).InfoS("failed to marshal HTTP filter config to JSON, skipping cluster extraction",
			"filter", httpFilter.Name,
			"error", err.Error())
		return nil
	}

	var data interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		klog.V(4).InfoS("failed to unmarshal HTTP filter JSON, skipping cluster extraction",
			"filter", httpFilter.Name,
			"error", err.Error())
		return nil
	}

	// Search for cluster references in various formats (using snake_case proto names)
	clusterFields := []string{"cluster", "cluster_name", "token_cluster", "authorization_cluster"}
	names := make([]string, 0, 4)
	for _, field := range clusterFields {
		fieldNames := utils.FindClusterNames(data, field)
		names = append(names, fieldNames...)
	}

	return names
}

// extractClustersFromAccessLog extracts cluster names from access log configurations.
// Note: Returns slice directly without copying - caller handles final copy.
func extractClustersFromAccessLog(accessLog *accesslogv3.AccessLog) ([]string, error) {
	typedConfig := accessLog.GetTypedConfig()
	if typedConfig == nil {
		return nil, nil
	}

	// typedConfig is *anypb.Any containing serialized protobuf, not JSON
	msg, err := anypb.UnmarshalNew(typedConfig, proto.UnmarshalOptions{})
	if err != nil {
		// Unknown access log type - skip cluster extraction
		klog.V(4).InfoS("cannot unmarshal access log config, skipping cluster extraction",
			"accessLog", accessLog.Name,
			"typeUrl", typedConfig.TypeUrl,
			"error", err.Error())
		return nil, nil
	}

	// Use proto field names (snake_case) to ensure consistent field name matching
	jsonBytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal access log config to JSON: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal access log config: %w", err)
	}

	// Search for cluster references (e.g., in HTTP grpc access log service, using snake_case proto names)
	clusterFields := []string{"cluster_name", "cluster"}
	names := make([]string, 0, 2)
	for _, field := range clusterFields {
		fieldNames := utils.FindClusterNames(data, field)
		names = append(names, fieldNames...)
	}

	return names, nil
}

// extractClustersFromTracing extracts cluster names from tracing configuration.
// Note: Returns slice directly without copying - caller handles final copy.
func extractClustersFromTracing(tracing *hcmv3.HttpConnectionManager_Tracing) ([]string, error) {
	// Use protojson with proto field names (snake_case) to ensure consistent field name matching
	jsonData, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(tracing)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tracing config: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tracing config: %w", err)
	}

	// Search for cluster references in tracing providers (using snake_case proto names)
	clusterFields := []string{"cluster_name", "collector_cluster"}
	names := make([]string, 0, 2)
	for _, field := range clusterFields {
		fieldNames := utils.FindClusterNames(data, field)
		names = append(names, fieldNames...)
	}

	return names, nil
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
