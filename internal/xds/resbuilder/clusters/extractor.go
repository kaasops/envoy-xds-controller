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
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder/utils"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"k8s.io/klog/v2"
)

// clusterExtractionResult holds results from cluster name extraction with source information
type clusterExtractionResult struct {
	name       string
	isRequired bool // true for known filter types (TCP Proxy, HCM), false for generic extraction
}

// ExtractClustersFromFilterChains extracts all cluster references from listener filter chains.
// This function provides optimized cluster extraction from existing listener configurations.
func ExtractClustersFromFilterChains(
	filterChains []*listenerv3.FilterChain,
	store store.Store,
) ([]*cluster.Cluster, error) {
	// Use pooled slice for cluster extraction results
	results := make([]clusterExtractionResult, 0, 8)

	// Extract cluster names from all filter chains
	for _, filterChain := range filterChains {
		for _, filter := range filterChain.Filters {
			filterResults, err := extractClustersFromFilter(filter)
			if err != nil {
				return nil, fmt.Errorf("failed to extract clusters from filter %s: %w", filter.Name, err)
			}
			results = append(results, filterResults...)
		}
	}

	if len(results) == 0 {
		return nil, nil
	}

	// Remove duplicates while preserving isRequired flag (required takes precedence)
	uniqueResults := removeDuplicateResults(results)

	// Pre-allocate with capacity hint
	// Note: Removed sync.Pool usage - benchmarks showed pool overhead (57ns)
	// exceeded direct allocation (0.25ns) due to metrics recording.
	clusters := make([]*cluster.Cluster, 0, len(uniqueResults))

	// Resolve clusters from store
	for _, result := range uniqueResults {
		cl := store.GetSpecCluster(result.name)
		if cl == nil {
			if result.isRequired {
				return nil, fmt.Errorf("cluster %s not found", result.name)
			}
			// For optional clusters (from generic extraction), skip with warning
			// This handles false positives where a field named "cluster" doesn't
			// actually reference an Envoy cluster
			klog.V(2).InfoS("cluster from generic extraction not found in store, skipping",
				"cluster", result.name)
			continue
		}
		xdsCluster, err := cl.UnmarshalV3AndValidate()
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal cluster %s: %w", result.name, err)
		}
		clusters = append(clusters, xdsCluster)
	}

	return clusters, nil
}

// removeDuplicateResults removes duplicate cluster names while preserving isRequired flag
// If a cluster appears both as required and optional, required takes precedence
func removeDuplicateResults(results []clusterExtractionResult) []clusterExtractionResult {
	if len(results) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(results)) // value: isRequired
	uniqueResults := make([]clusterExtractionResult, 0, len(results))

	for _, r := range results {
		if r.name == "" {
			continue
		}
		if isRequired, exists := seen[r.name]; exists {
			// If current is required, update the existing entry
			if r.isRequired && !isRequired {
				seen[r.name] = true
				// Update in-place in uniqueResults
				for i := range uniqueResults {
					if uniqueResults[i].name == r.name {
						uniqueResults[i].isRequired = true
						break
					}
				}
			}
		} else {
			seen[r.name] = r.isRequired
			uniqueResults = append(uniqueResults, r)
		}
	}

	return uniqueResults
}

// extractClustersFromFilter extracts cluster names from a specific listener filter.
// Returns clusterExtractionResult with isRequired=true for known filter types (TCP Proxy, HCM)
// and isRequired=false for generic extraction (unknown filter types).
func extractClustersFromFilter(filter *listenerv3.Filter) ([]clusterExtractionResult, error) {
	switch filter.Name {
	case wellknown.HTTPConnectionManager:
		names, err := extractClustersFromHCMFilter(filter)
		if err != nil {
			return nil, err
		}
		return toRequiredResults(names), nil

	case wellknown.TCPProxy:
		names, err := extractClustersFromTCPProxyFilter(filter)
		if err != nil {
			return nil, err
		}
		return toRequiredResults(names), nil

	default:
		// For other filter types, try generic JSON-based extraction as fallback
		// Results are marked as optional (isRequired=false) to handle false positives
		names, err := extractClustersFromGenericFilter(filter)
		if err != nil {
			return nil, err
		}
		return toOptionalResults(names), nil
	}
}

// toRequiredResults converts string slice to clusterExtractionResult with isRequired=true
func toRequiredResults(names []string) []clusterExtractionResult {
	if len(names) == 0 {
		return nil
	}
	results := make([]clusterExtractionResult, len(names))
	for i, name := range names {
		results[i] = clusterExtractionResult{name: name, isRequired: true}
	}
	return results
}

// toOptionalResults converts string slice to clusterExtractionResult with isRequired=false
func toOptionalResults(names []string) []clusterExtractionResult {
	if len(names) == 0 {
		return nil
	}
	results := make([]clusterExtractionResult, len(names))
	for i, name := range names {
		results[i] = clusterExtractionResult{name: name, isRequired: false}
	}
	return results
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
		// Log at V(2) for visibility in production debugging.
		klog.V(2).InfoS("unknown filter type, skipping cluster extraction (filter type not registered in go-control-plane)",
			"filter", filter.Name,
			"typeUrl", typedConfig.TypeUrl)
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
		// Unknown HTTP filter type - skip cluster extraction
		// Log at V(2) for visibility in production debugging.
		klog.V(2).InfoS("unknown HTTP filter type, skipping cluster extraction",
			"filter", httpFilter.Name,
			"typeUrl", typedConfig.TypeUrl)
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
		// Log at V(2) for visibility in production debugging.
		klog.V(2).InfoS("unknown access log type, skipping cluster extraction",
			"accessLog", accessLog.Name,
			"typeUrl", typedConfig.TypeUrl)
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
