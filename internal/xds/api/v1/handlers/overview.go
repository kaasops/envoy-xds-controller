package handlers

import (
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"google.golang.org/protobuf/proto"
)

// getOverview returns a human-readable overview of node configuration.
// @Summary Get human-readable overview of node configuration
// @Description Returns endpoints, certificates, and summary statistics for a node
// @Tags overview
// @Accept json
// @Produce json
// @Param node_id query string true "Node ID" format(string) example("node-id-1")
// @Success 200 {object} NodeOverviewResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/overview [get]
func (h *handler) getOverview(ctx *gin.Context) {
	nodeID, err := h.getRequiredOnlyOneParam(ctx.Request.URL.Query(), nodeIDParamName)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Check node_id exists in cache
	nodeIDs := h.getAvailableNodeIDs(ctx)
	if !slices.Contains(nodeIDs, nodeID) {
		ctx.JSON(400, gin.H{"error": "node_id not found in cache", "node_id": nodeID})
		return
	}

	// Check if we have a cached response
	if cached := h.overviewCache.Get(nodeID); cached != nil {
		ctx.JSON(200, cached)
		return
	}

	// Get all resources
	listeners, err := h.cache.GetListeners(nodeID)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	secrets, err := h.cache.GetSecrets(nodeID)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Build certificate info map for quick lookup
	certInfoMap := h.buildCertificateInfoMap(secrets)

	// Extract endpoints from listeners
	endpoints := h.extractEndpoints(listeners, certInfoMap)

	// Build certificates list with usage info
	certificates := h.buildCertificatesList(certInfoMap)

	// Calculate summary
	summary := h.calculateSummary(endpoints, certificates)

	response := &NodeOverviewResponse{
		NodeID:       nodeID,
		Summary:      summary,
		Endpoints:    endpoints,
		Certificates: certificates,
	}

	// Cache the response
	h.overviewCache.Set(nodeID, response)

	ctx.JSON(200, response)
}

// buildCertificateInfoMap parses all secrets and builds a map of certificate info.
func (h *handler) buildCertificateInfoMap(secrets []*tlsv3.Secret) map[string]*CertificateInfo {
	certMap := make(map[string]*CertificateInfo)

	for _, secret := range secrets {
		tlsCert, ok := secret.Type.(*tlsv3.Secret_TlsCertificate)
		if !ok {
			continue
		}

		certChain := tlsCert.TlsCertificate.GetCertificateChain()
		if certChain == nil {
			continue
		}

		certData := certChain.GetInlineBytes()
		if len(certData) == 0 {
			continue
		}

		certs, err := parsePEM(certData)
		if err != nil || len(certs) == 0 {
			continue
		}

		// Use the first certificate (leaf certificate)
		cert := certs[0]

		// Parse namespace and name from secret name (format: namespace/name)
		namespace, name := parseSecretName(secret.Name)

		daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
		status := calculateCertStatus(daysUntilExpiry)

		certInfo := &CertificateInfo{
			Name:            name,
			Namespace:       namespace,
			Subject:         cert.Subject.String(),
			Issuer:          cert.Issuer.String(),
			NotBefore:       cert.NotBefore.Format(time.RFC3339),
			NotAfter:        cert.NotAfter.Format(time.RFC3339),
			DaysUntilExpiry: daysUntilExpiry,
			Status:          status,
			DNSNames:        cert.DNSNames,
			// UsedByDomains is nil by default, will be initialized on first use
		}

		certMap[secret.Name] = certInfo
	}

	return certMap
}

// extractEndpoints extracts endpoint information from listeners.
func (h *handler) extractEndpoints(listeners []*listenerv3.Listener, certInfoMap map[string]*CertificateInfo) []EndpointInfo {
	var endpoints []EndpointInfo

	for _, listener := range listeners {
		port := extractPort(listener)

		for _, filterChain := range listener.FilterChains {
			// Get domains from server names
			domains := getDomainsFromFilterChain(filterChain)
			if len(domains) == 0 {
				// If no server names, this might be a default filter chain
				// or a non-SNI listener
				domains = []string{"*"}
			}

			// Determine if TLS is enabled
			hasTLS := filterChain.GetTransportSocket() != nil

			// Get secret name if TLS is enabled
			secretName := ""
			if hasTLS {
				secretName = extractSecretName(filterChain)
			}

			// Determine protocol
			protocol := determineProtocol(filterChain, hasTLS)

			// Get route config name
			routeConfigName := h.getRouteConfigNameFromFilterChain(filterChain)

			for _, domain := range domains {
				endpoint := EndpointInfo{
					Domain:          domain,
					Port:            port,
					Protocol:        protocol,
					ListenerName:    listener.Name,
					RouteConfigName: routeConfigName,
				}

				// Add certificate info if available
				if secretName != "" {
					if certInfo, ok := certInfoMap[secretName]; ok {
						endpoint.Certificate = &CertificateBrief{
							Name:            certInfo.Name,
							ExpiresAt:       certInfo.NotAfter,
							DaysUntilExpiry: certInfo.DaysUntilExpiry,
							Status:          certInfo.Status,
						}
						// Track which domains use this certificate
						if certInfo.UsedByDomains == nil {
							certInfo.UsedByDomains = make([]string, 0, 4)
						}
						certInfo.UsedByDomains = append(certInfo.UsedByDomains, domain)
					}
				}

				endpoints = append(endpoints, endpoint)
			}
		}
	}

	return endpoints
}

// buildCertificatesList converts the certificate map to a list.
func (h *handler) buildCertificatesList(certInfoMap map[string]*CertificateInfo) []CertificateInfo {
	certificates := make([]CertificateInfo, 0, len(certInfoMap))

	for _, certInfo := range certInfoMap {
		// Create a copy to avoid pointer issues
		cert := *certInfo
		certificates = append(certificates, cert)
	}

	return certificates
}

// calculateSummary calculates summary statistics.
func (h *handler) calculateSummary(endpoints []EndpointInfo, certificates []CertificateInfo) OverviewSummary {
	// Count unique domains (excluding wildcard "*")
	uniqueDomains := make(map[string]struct{})
	for _, ep := range endpoints {
		if ep.Domain != "*" {
			uniqueDomains[ep.Domain] = struct{}{}
		}
	}

	summary := OverviewSummary{
		TotalDomains:      len(uniqueDomains),
		TotalEndpoints:    len(endpoints),
		TotalCertificates: len(certificates),
	}

	for _, cert := range certificates {
		switch cert.Status {
		case CertStatusWarning:
			summary.CertificatesWarning++
		case CertStatusCritical:
			summary.CertificatesCritical++
		case CertStatusExpired:
			summary.CertificatesExpired++
		}
	}

	return summary
}

// extractPort extracts the port number from a listener's address.
func extractPort(listener *listenerv3.Listener) uint32 {
	if listener.Address == nil {
		return 0
	}

	socketAddr := listener.Address.GetSocketAddress()
	if socketAddr == nil {
		return 0
	}

	return socketAddr.GetPortValue()
}

// getDomainsFromFilterChain extracts server names from a filter chain.
func getDomainsFromFilterChain(filterChain *listenerv3.FilterChain) []string {
	if filterChain.FilterChainMatch == nil {
		return nil
	}

	return filterChain.FilterChainMatch.GetServerNames()
}

// extractSecretName extracts the secret name from a filter chain's TLS configuration.
func extractSecretName(filterChain *listenerv3.FilterChain) string {
	transportSocket := filterChain.GetTransportSocket()
	if transportSocket == nil {
		return ""
	}

	// Check if it's a TLS transport socket
	if transportSocket.Name != "envoy.transport_sockets.tls" {
		return ""
	}

	typedConfig := transportSocket.GetTypedConfig()
	if typedConfig == nil {
		return ""
	}

	// Try to unmarshal as DownstreamTlsContext
	downstreamTLS := &tlsv3.DownstreamTlsContext{}
	if err := proto.Unmarshal(typedConfig.Value, downstreamTLS); err != nil {
		return ""
	}

	commonTLS := downstreamTLS.GetCommonTlsContext()
	if commonTLS == nil {
		return ""
	}

	// Get secret name from SDS config
	sdsConfigs := commonTLS.GetTlsCertificateSdsSecretConfigs()
	if len(sdsConfigs) > 0 {
		return sdsConfigs[0].GetName()
	}

	return ""
}

// determineProtocol determines the protocol based on filter chain configuration.
func determineProtocol(filterChain *listenerv3.FilterChain, hasTLS bool) string {
	// Check if any filter is HTTP Connection Manager
	for _, filter := range filterChain.Filters {
		hcmConfig := resourcev3.GetHTTPConnectionManager(filter)
		if hcmConfig != nil {
			if hasTLS {
				return "HTTPS"
			}
			return "HTTP"
		}
	}

	// If not HTTP, check for TCP proxy
	for _, filter := range filterChain.Filters {
		if filter.Name == "envoy.filters.network.tcp_proxy" {
			return "TCP"
		}
	}

	// Default to TCP
	return "TCP"
}

// getRouteConfigNameFromFilterChain extracts the route configuration name from a filter chain.
func (h *handler) getRouteConfigNameFromFilterChain(filterChain *listenerv3.FilterChain) string {
	for _, filter := range filterChain.Filters {
		rdsName := h.getRDSNameForFilter(filter)
		if rdsName != "" {
			return rdsName
		}
	}
	return ""
}

// parseSecretName parses namespace and name from a secret name (format: namespace/name).
func parseSecretName(secretName string) (namespace, name string) {
	parts := strings.Split(secretName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", secretName
}

// calculateCertStatus calculates the certificate status based on days until expiry.
func calculateCertStatus(daysUntilExpiry int) string {
	if daysUntilExpiry <= 0 {
		return CertStatusExpired
	}
	if daysUntilExpiry <= CertCriticalThresholdDays {
		return CertStatusCritical
	}
	if daysUntilExpiry <= CertWarningThresholdDays {
		return CertStatusWarning
	}
	return CertStatusOK
}
