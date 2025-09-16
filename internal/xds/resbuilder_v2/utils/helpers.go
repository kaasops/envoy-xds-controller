package utils

import (
	"fmt"
	"strings"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
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

// IsRootRoute checks if a route matches root path patterns (prefix="/" or path="/")
func IsRootRoute(route *routev3.Route) bool {
	if route == nil || route.Match == nil {
		return false
	}

	switch pathSpec := route.Match.PathSpecifier.(type) {
	case *routev3.RouteMatch_Prefix:
		return pathSpec.Prefix == RootPrefix
	case *routev3.RouteMatch_Path:
		return pathSpec.Path == RootPath
	default:
		return false
	}
}

// FindRootRouteIndexes finds indexes of routes that match root paths in a slice
func FindRootRouteIndexes(routes []*routev3.Route) []int {
	var rootIndexes []int

	for index, route := range routes {
		if IsRootRoute(route) {
			rootIndexes = append(rootIndexes, index)
		}
	}

	return rootIndexes
}

// MoveRouteToEnd moves a route from the given index to the end of the routes slice
func MoveRouteToEnd(routes []*routev3.Route, index int) []*routev3.Route {
	if index < 0 || index >= len(routes) {
		return routes // Invalid index, return unchanged
	}

	route := routes[index]
	// Remove route from current position
	result := append(routes[:index], routes[index+1:]...)
	// Add route to the end
	result = append(result, route)
	
	return result
}

// RemoveDuplicateStrings removes duplicate entries from a string slice while preserving order
func RemoveDuplicateStrings(strings []string) []string {
	if len(strings) <= 1 {
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

// GenerateStatPrefix creates a stat prefix from a namespaced name by replacing dots with hyphens
func GenerateStatPrefix(nn helpers.NamespacedName) string {
	return strings.ReplaceAll(nn.String(), ".", DefaultStatPrefixSeparator)
}

// ShouldAddFallbackVirtualHost determines if a fallback virtual host should be added
func ShouldAddFallbackVirtualHost(domains []string, isTLSListener, hasPort443 bool) bool {
	// Add fallback route for TLS listeners
	// https://github.com/envoyproxy/envoy/issues/37810
	return isTLSListener && 
		!(len(domains) == 1 && domains[0] == FallbackVirtualHostDomain) && 
		hasPort443
}

// ExtractClusterNamesFromRoutes extracts all cluster names referenced in a slice of routes
func ExtractClusterNamesFromRoutes(routes []*routev3.Route) []string {
	var names []string
	
	for _, route := range routes {
		routeNames := ExtractClusterNamesFromRoute(route)
		names = append(names, routeNames...)
	}
	
	return RemoveDuplicateStrings(names)
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

// FindClusterNames recursively searches for cluster names in configuration data
// This is a utility function for JSON-based cluster extraction
func FindClusterNames(data interface{}, clusterField string) []string {
	var names []string
	
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if key == clusterField {
				if str, ok := value.(string); ok && str != "" {
					names = append(names, str)
				}
			} else {
				names = append(names, FindClusterNames(value, clusterField)...)
			}
		}
	case []interface{}:
		for _, item := range v {
			names = append(names, FindClusterNames(item, clusterField)...)
		}
	}
	
	return names
}

// FindSDSNames recursively searches for SDS names in configuration data
func FindSDSNames(data interface{}, fieldName string) []string {
	var results []string

	switch value := data.(type) {
	case map[string]interface{}:
		for k, v := range value {
			if k == fieldName {
				if nameValue, ok := value[SDSNameFieldName]; ok {
					results = append(results, fmt.Sprintf("%v", nameValue))
				}
			}
			results = append(results, FindSDSNames(v, fieldName)...)
		}
	case []interface{}:
		for _, item := range value {
			results = append(results, FindSDSNames(item, fieldName)...)
		}
	}

	return results
}

// HasRouterFilter checks if a slice of HTTP filters contains a router filter
func HasRouterFilter(filters []*hcmv3.HttpFilter) bool {
	for _, f := range filters {
		if tc := f.GetTypedConfig(); tc != nil {
			if tc.TypeUrl == RouterFilterTypeURL {
				return true
			}
		}
	}
	return false
}

// FindRouterFilterIndexes finds all indexes of router filters in a slice of HTTP filters
func FindRouterFilterIndexes(filters []*hcmv3.HttpFilter) []int {
	var indexes []int
	
	for i, f := range filters {
		if tc := f.GetTypedConfig(); tc != nil {
			if tc.TypeUrl == RouterFilterTypeURL {
				indexes = append(indexes, i)
			}
		}
	}
	
	return indexes
}

// CopyDomains creates a deep copy of a domains slice to avoid mutation issues
func CopyDomains(domains []string) []string {
	if len(domains) == 0 {
		return nil
	}
	
	result := make([]string, len(domains))
	copy(result, domains)
	
	return result
}

// IsWildcardDomain checks if a domain is a wildcard domain
func IsWildcardDomain(domain string) bool {
	return domain == "*" || strings.HasPrefix(domain, "*.")
}

// NormalizeNamespacedName creates a helpers.NamespacedName with proper namespace handling
func NormalizeNamespacedName(namespace, name, defaultNamespace string) helpers.NamespacedName {
	ns := namespace
	if ns == "" {
		ns = defaultNamespace
	}
	
	return helpers.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
}

// ValidateNonEmpty checks that a string is not empty and returns an appropriate error
func ValidateNonEmpty(value, fieldName string) error {
	if value == "" {
		return fmt.Errorf("%s is empty", fieldName)
	}
	return nil
}

// ValidateNonNil checks that a pointer is not nil and returns an appropriate error
func ValidateNonNil(value interface{}, fieldName string) error {
	if value == nil {
		return fmt.Errorf("%s is nil", fieldName)
	}
	return nil
}

// CountNonEmptyStrings counts the number of non-empty strings in a slice
func CountNonEmptyStrings(strings []string) int {
	count := 0
	for _, s := range strings {
		if s != "" {
			count++
		}
	}
	return count
}

// FilterNonEmptyStrings returns a new slice with only non-empty strings
func FilterNonEmptyStrings(strings []string) []string {
	result := make([]string, 0, len(strings))
	for _, s := range strings {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// ContainsString checks if a slice contains a specific string
func ContainsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// AppendUniqueStrings appends strings to a slice only if they're not already present
func AppendUniqueStrings(slice []string, items ...string) []string {
	for _, item := range items {
		if item != "" && !ContainsString(slice, item) {
			slice = append(slice, item)
		}
	}
	return slice
}