package utils

import (
	"fmt"
	"strings"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

// IsTLSListener checks if a listener has TLS configuration enabled
func IsTLSListener(xdsListener *listenerv3.Listener) bool {
	if xdsListener == nil {
		return false
	}

	if len(xdsListener.ListenerFilters) == 0 {
		return false
	}

	for _, lFilter := range xdsListener.ListenerFilters {
		if tc := lFilter.GetTypedConfig(); tc != nil {
			if tc.TypeUrl == TLSInspectorTypeURL {
				return true
			}
		}
	}

	return false
}

// ListenerHasPort443 checks if a listener is configured for port 443
func ListenerHasPort443(xdsListener *listenerv3.Listener) bool {
	if xdsListener == nil || xdsListener.Address == nil {
		return false
	}

	socketAddr := xdsListener.Address.GetSocketAddress()
	if socketAddr == nil {
		return false
	}

	return socketAddr.GetPortValue() == HTTPSPort
}

// CheckAllDomainsUnique ensures that all domains in a slice are unique
func CheckAllDomainsUnique(domains []string) error {
	if len(domains) <= 1 {
		return nil
	}

	seen := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		if domain == "" {
			continue // Skip empty domains
		}
		if _, exists := seen[domain]; exists {
			return fmt.Errorf("duplicate domain found: %s", domain)
		}
		seen[domain] = struct{}{}
	}

	return nil
}

// ExtractClusterNamesFromRoute directly extracts cluster names from a single route configuration
func ExtractClusterNamesFromRoute(route *routev3.Route) []string {
	var names []string

	if route == nil || route.Action == nil {
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

// maxRecursionDepth limits the recursion depth for FindClusterNames
// to prevent stack overflow on deeply nested or malformed configurations
const maxRecursionDepth = 50

// FindClusterNames recursively searches for cluster names in configuration data
// This is a utility function for JSON-based cluster extraction
func FindClusterNames(data interface{}, clusterField string) []string {
	return findClusterNamesWithDepth(data, clusterField, 0)
}

// findClusterNamesWithDepth is the internal recursive implementation with depth tracking
func findClusterNamesWithDepth(data interface{}, clusterField string, depth int) []string {
	// Prevent stack overflow on deeply nested structures
	if depth >= maxRecursionDepth {
		return nil
	}

	var names []string

	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if key == clusterField {
				if str, ok := value.(string); ok && str != "" {
					names = append(names, str)
				}
			} else {
				names = append(names, findClusterNamesWithDepth(value, clusterField, depth+1)...)
			}
		}
	case []interface{}:
		for _, item := range v {
			names = append(names, findClusterNamesWithDepth(item, clusterField, depth+1)...)
		}
	}

	return names
}

// GetWildcardDomain converts a domain to its wildcard equivalent
// For example: "api.example.com" -> "*.example.com"
func GetWildcardDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return ""
	}
	parts[0] = "*"
	return strings.Join(parts, ".")
}
