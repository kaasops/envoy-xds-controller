package utils

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

var (
	// validDomainRegex matches valid domain names including wildcards
	validDomainRegex = regexp.MustCompile(`^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	
	// validClusterNameRegex matches valid cluster names
	validClusterNameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-_]{0,61}[a-zA-Z0-9])?$`)
)

// ValidateDomains validates a list of domain names
func ValidateDomains(domains []string) error {
	if len(domains) == 0 {
		return fmt.Errorf("no domains specified")
	}

	seen := make(map[string]bool, len(domains))
	for i, domain := range domains {
		if domain == "" {
			return fmt.Errorf("domain[%d] is empty", i)
		}

		// Check for duplicates
		if seen[domain] {
			return fmt.Errorf("duplicate domain '%s' found", domain)
		}
		seen[domain] = true

		// Validate domain format (allow wildcards)
		if domain != "*" && !validDomainRegex.MatchString(domain) {
			return fmt.Errorf("invalid domain format: '%s'", domain)
		}
	}

	return nil
}

// ValidateClusterName validates a cluster name format
func ValidateClusterName(clusterName string) error {
	if clusterName == "" {
		return fmt.Errorf("cluster name is empty")
	}

	if !validClusterNameRegex.MatchString(clusterName) {
		return fmt.Errorf("invalid cluster name format: '%s'", clusterName)
	}

	return nil
}

// ValidateClusterNames validates a list of cluster names
func ValidateClusterNames(clusterNames []string) error {
	if len(clusterNames) == 0 {
		return fmt.Errorf("no cluster names specified")
	}

	for i, name := range clusterNames {
		if err := ValidateClusterName(name); err != nil {
			return fmt.Errorf("cluster[%d]: %w", i, err)
		}
	}

	return nil
}

// ValidateRouteConfiguration validates a route configuration
func ValidateRouteConfiguration(routeConfig *routev3.RouteConfiguration) error {
	if routeConfig == nil {
		return fmt.Errorf("route configuration is nil")
	}

	if routeConfig.Name == "" {
		return fmt.Errorf("route configuration name is empty")
	}

	if len(routeConfig.VirtualHosts) == 0 {
		return fmt.Errorf("route configuration has no virtual hosts")
	}

	// Validate each virtual host
	for i, vh := range routeConfig.VirtualHosts {
		if err := ValidateVirtualHost(vh); err != nil {
			return fmt.Errorf("virtual host[%d]: %w", i, err)
		}
	}

	return nil
}

// ValidateVirtualHost validates a virtual host configuration
func ValidateVirtualHost(vh *routev3.VirtualHost) error {
	if vh == nil {
		return fmt.Errorf("virtual host is nil")
	}

	if vh.Name == "" {
		return fmt.Errorf("virtual host name is empty")
	}

	if len(vh.Domains) == 0 {
		return fmt.Errorf("virtual host has no domains")
	}

	if err := ValidateDomains(vh.Domains); err != nil {
		return fmt.Errorf("invalid domains: %w", err)
	}

	if len(vh.Routes) == 0 {
		return fmt.Errorf("virtual host has no routes")
	}

	// Validate each route
	for i, route := range vh.Routes {
		if err := ValidateRoute(route); err != nil {
			return fmt.Errorf("route[%d]: %w", i, err)
		}
	}

	return nil
}

// ValidateRoute validates a route configuration
func ValidateRoute(route *routev3.Route) error {
	if route == nil {
		return fmt.Errorf("route is nil")
	}

	if route.Match == nil {
		return fmt.Errorf("route match is nil")
	}

	if route.Action == nil {
		return fmt.Errorf("route action is nil")
	}

	// Validate route match
	if err := ValidateRouteMatch(route.Match); err != nil {
		return fmt.Errorf("invalid route match: %w", err)
	}

	// Validate route action
	if err := ValidateRouteAction(route.Action); err != nil {
		return fmt.Errorf("invalid route action: %w", err)
	}

	return nil
}

// ValidateRouteMatch validates a route match configuration
func ValidateRouteMatch(match *routev3.RouteMatch) error {
	if match == nil {
		return fmt.Errorf("route match is nil")
	}

	if match.PathSpecifier == nil {
		return fmt.Errorf("route match path specifier is nil")
	}

	// Validate path specifier
	switch pathSpec := match.PathSpecifier.(type) {
	case *routev3.RouteMatch_Prefix:
		if pathSpec.Prefix == "" {
			return fmt.Errorf("route match prefix is empty")
		}
	case *routev3.RouteMatch_Path:
		if pathSpec.Path == "" {
			return fmt.Errorf("route match path is empty")
		}
	case *routev3.RouteMatch_SafeRegex:
		if pathSpec.SafeRegex == nil || pathSpec.SafeRegex.Regex == "" {
			return fmt.Errorf("route match regex is empty")
		}
	default:
		return fmt.Errorf("unknown route match path specifier type")
	}

	return nil
}

// ValidateRouteAction validates a route action configuration
func ValidateRouteAction(action interface{}) error {
	if action == nil {
		return fmt.Errorf("route action is nil")
	}

	switch act := action.(type) {
	case *routev3.Route_Route:
		return ValidateRouteActionRoute(act.Route)
	case *routev3.Route_DirectResponse:
		return ValidateDirectResponse(act.DirectResponse)
	case *routev3.Route_Redirect:
		return ValidateRedirectAction(act.Redirect)
	default:
		return fmt.Errorf("unknown route action type")
	}
}

// ValidateRouteActionRoute validates a route action of type route
func ValidateRouteActionRoute(routeAction *routev3.RouteAction) error {
	if routeAction == nil {
		return fmt.Errorf("route action is nil")
	}

	if routeAction.ClusterSpecifier == nil {
		return fmt.Errorf("cluster specifier is nil")
	}

	// Validate cluster specifier
	switch clusterSpec := routeAction.ClusterSpecifier.(type) {
	case *routev3.RouteAction_Cluster:
		if clusterSpec.Cluster == "" {
			return fmt.Errorf("cluster name is empty")
		}
		return ValidateClusterName(clusterSpec.Cluster)
	case *routev3.RouteAction_WeightedClusters:
		return ValidateWeightedClusters(clusterSpec.WeightedClusters)
	default:
		return fmt.Errorf("unknown cluster specifier type")
	}
}

// ValidateWeightedClusters validates weighted clusters configuration
func ValidateWeightedClusters(weightedClusters *routev3.WeightedCluster) error {
	if weightedClusters == nil {
		return fmt.Errorf("weighted clusters is nil")
	}

	if len(weightedClusters.Clusters) == 0 {
		return fmt.Errorf("no weighted clusters specified")
	}

	totalWeight := uint32(0)
	for i, cluster := range weightedClusters.Clusters {
		if cluster.Name == "" {
			return fmt.Errorf("weighted cluster[%d] name is empty", i)
		}

		if err := ValidateClusterName(cluster.Name); err != nil {
			return fmt.Errorf("weighted cluster[%d]: %w", i, err)
		}

		if cluster.Weight != nil {
			totalWeight += cluster.Weight.Value
		}
	}

	// Optional: validate total weight makes sense (typically should be 100 or similar)
	if totalWeight == 0 {
		return fmt.Errorf("total weight of weighted clusters is zero")
	}

	return nil
}

// ValidateDirectResponse validates a direct response action
func ValidateDirectResponse(directResponse *routev3.DirectResponseAction) error {
	if directResponse == nil {
		return fmt.Errorf("direct response is nil")
	}

	if directResponse.Status == 0 {
		return fmt.Errorf("direct response status is not set")
	}

	// Validate HTTP status code range
	if directResponse.Status < 100 || directResponse.Status > 599 {
		return fmt.Errorf("invalid HTTP status code: %d", directResponse.Status)
	}

	return nil
}

// ValidateRedirectAction validates a redirect action
func ValidateRedirectAction(redirect *routev3.RedirectAction) error {
	if redirect == nil {
		return fmt.Errorf("redirect action is nil")
	}

	// At least one redirect target must be specified
	hasTarget := redirect.HostRedirect != "" ||
		redirect.PathRewriteSpecifier != nil ||
		redirect.SchemeRewriteSpecifier != nil

	if !hasTarget {
		return fmt.Errorf("redirect action has no target specified")
	}

	return nil
}

// ValidateListener validates a listener configuration
func ValidateListener(listener *listenerv3.Listener) error {
	if listener == nil {
		return fmt.Errorf("listener is nil")
	}

	if listener.Name == "" {
		return fmt.Errorf("listener name is empty")
	}

	if listener.Address == nil {
		return fmt.Errorf("listener address is nil")
	}

	if err := ValidateListenerAddress(listener.Address); err != nil {
		return fmt.Errorf("invalid listener address: %w", err)
	}

	return nil
}

// ValidateListenerAddress validates a listener address
func ValidateListenerAddress(address *corev3.Address) error {
	if address == nil {
		return fmt.Errorf("address is nil")
	}

	switch addr := address.Address.(type) {
	case *corev3.Address_SocketAddress:
		return ValidateSocketAddress(addr.SocketAddress)
	case *corev3.Address_Pipe:
		return ValidatePipeAddress(addr.Pipe)
	default:
		return fmt.Errorf("unknown address type")
	}
}

// ValidateSocketAddress validates a socket address
func ValidateSocketAddress(socketAddr *corev3.SocketAddress) error {
	if socketAddr == nil {
		return fmt.Errorf("socket address is nil")
	}

	if socketAddr.Address == "" {
		return fmt.Errorf("socket address is empty")
	}

	// Validate IP address format
	if ip := net.ParseIP(socketAddr.Address); ip == nil {
		// If not a valid IP, check if it's a valid hostname
		if !IsValidHostname(socketAddr.Address) {
			return fmt.Errorf("invalid IP address or hostname: %s", socketAddr.Address)
		}
	}

	// Validate port
	switch portSpec := socketAddr.PortSpecifier.(type) {
	case *corev3.SocketAddress_PortValue:
		if portSpec.PortValue == 0 || portSpec.PortValue > 65535 {
			return fmt.Errorf("invalid port value: %d", portSpec.PortValue)
		}
	case *corev3.SocketAddress_NamedPort:
		if portSpec.NamedPort == "" {
			return fmt.Errorf("named port is empty")
		}
	}

	return nil
}

// ValidatePipeAddress validates a pipe address
func ValidatePipeAddress(pipe *corev3.Pipe) error {
	if pipe == nil {
		return fmt.Errorf("pipe address is nil")
	}

	if pipe.Path == "" {
		return fmt.Errorf("pipe path is empty")
	}

	return nil
}

// ValidateSecretData validates Kubernetes secret data for TLS usage
func ValidateSecretData(secret *v1.Secret) error {
	if secret == nil {
		return fmt.Errorf("secret is nil")
	}

	if secret.Type != v1.SecretTypeTLS && secret.Type != v1.SecretTypeOpaque {
		return fmt.Errorf("unsupported secret type: %s", secret.Type)
	}

	if secret.Data == nil {
		return fmt.Errorf("secret data is nil")
	}

	// Check for required certificate data
	if _, exists := secret.Data[v1.TLSCertKey]; !exists {
		return fmt.Errorf("missing certificate data (key: %s)", v1.TLSCertKey)
	}

	// Check for required private key data
	if _, exists := secret.Data[v1.TLSPrivateKeyKey]; !exists {
		return fmt.Errorf("missing private key data (key: %s)", v1.TLSPrivateKeyKey)
	}

	// Validate data is not empty
	if len(secret.Data[v1.TLSCertKey]) == 0 {
		return fmt.Errorf("certificate data is empty")
	}

	if len(secret.Data[v1.TLSPrivateKeyKey]) == 0 {
		return fmt.Errorf("private key data is empty")
	}

	return nil
}

// ValidateVirtualServiceSpec validates VirtualService specification
func ValidateVirtualServiceSpec(vs *v1alpha1.VirtualService) error {
	if vs == nil {
		return fmt.Errorf("virtual service is nil")
	}

	if vs.Name == "" {
		return fmt.Errorf("virtual service name is empty")
	}

	if vs.Namespace == "" {
		return fmt.Errorf("virtual service namespace is empty")
	}

	if vs.Spec.Listener == nil {
		return fmt.Errorf("virtual service listener reference is nil")
	}

	if vs.Spec.Listener.Name == "" {
		return fmt.Errorf("virtual service listener name is empty")
	}

	return nil
}

// IsValidHostname checks if a string is a valid hostname
func IsValidHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}

	// Hostname cannot start or end with a hyphen
	if strings.HasPrefix(hostname, "-") || strings.HasSuffix(hostname, "-") {
		return false
	}

	// Split by dots and validate each label
	labels := strings.Split(hostname, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		
		// Each label must start and end with alphanumeric character
		if !((label[0] >= 'a' && label[0] <= 'z') || 
		     (label[0] >= 'A' && label[0] <= 'Z') || 
		     (label[0] >= '0' && label[0] <= '9')) {
			return false
		}
		
		if !((label[len(label)-1] >= 'a' && label[len(label)-1] <= 'z') || 
		     (label[len(label)-1] >= 'A' && label[len(label)-1] <= 'Z') || 
		     (label[len(label)-1] >= '0' && label[len(label)-1] <= '9')) {
			return false
		}
		
		// Check for valid characters in the middle
		for _, char := range label {
			if !((char >= 'a' && char <= 'z') || 
			     (char >= 'A' && char <= 'Z') || 
			     (char >= '0' && char <= '9') || 
			     char == '-') {
				return false
			}
		}
	}

	return true
}

// ValidateResourceRef validates a resource reference
func ValidateResourceRef(ref *v1alpha1.ResourceRef, refType string) error {
	if ref == nil {
		return fmt.Errorf("%s reference is nil", refType)
	}

	if ref.Name == "" {
		return fmt.Errorf("%s reference name is empty", refType)
	}

	return nil
}