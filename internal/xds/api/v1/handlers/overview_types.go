package handlers

// NodeOverviewResponse represents the complete overview of a node's configuration
// in a human-readable format.
type NodeOverviewResponse struct {
	NodeID           string            `json:"nodeId"`
	Summary          OverviewSummary   `json:"summary"`
	ResourceVersions ResourceVersions  `json:"resourceVersions"`
	Endpoints        []EndpointInfo    `json:"endpoints"`
	Certificates     []CertificateInfo `json:"certificates"`
}

// ResourceVersions contains xDS resource version strings for each resource type.
type ResourceVersions struct {
	Listeners string `json:"listeners"`
	Clusters  string `json:"clusters"`
	Routes    string `json:"routes"`
	Secrets   string `json:"secrets"`
}

// OverviewSummary provides aggregate statistics for the node.
type OverviewSummary struct {
	TotalDomains         int `json:"totalDomains"`
	TotalEndpoints       int `json:"totalEndpoints"`
	TotalCertificates    int `json:"totalCertificates"`
	CertificatesWarning  int `json:"certificatesWarning"`  // 7-30 days until expiry
	CertificatesCritical int `json:"certificatesCritical"` // < 7 days until expiry
	CertificatesExpired  int `json:"certificatesExpired"`  // already expired
}

// EndpointInfo represents a single endpoint (domain + port combination).
type EndpointInfo struct {
	Domain          string            `json:"domain"`
	Port            uint32            `json:"port"`
	Protocol        string            `json:"protocol"` // HTTP, HTTPS, TCP
	ListenerName    string            `json:"listenerName"`
	RouteConfigName string            `json:"routeConfigName,omitempty"`
	Certificate     *CertificateBrief `json:"certificate,omitempty"`
}

// CertificateBrief contains minimal certificate info for endpoint display.
type CertificateBrief struct {
	Name            string `json:"name"`
	ExpiresAt       string `json:"expiresAt"`
	DaysUntilExpiry int    `json:"daysUntilExpiry"`
	Status          string `json:"status"` // ok, warning, critical, expired
}

// CertificateInfo contains detailed certificate information.
type CertificateInfo struct {
	Name            string   `json:"name"`
	Namespace       string   `json:"namespace"`
	Subject         string   `json:"subject"`
	Issuer          string   `json:"issuer"`
	NotBefore       string   `json:"notBefore"`
	NotAfter        string   `json:"notAfter"`
	DaysUntilExpiry int      `json:"daysUntilExpiry"`
	Status          string   `json:"status"` // ok, warning, critical, expired
	DNSNames        []string `json:"dnsNames"`
	UsedByDomains   []string `json:"usedByDomains"`
}

// Certificate status constants.
const (
	CertStatusOK       = "ok"
	CertStatusWarning  = "warning"
	CertStatusCritical = "critical"
	CertStatusExpired  = "expired"
)

// Certificate status thresholds in days.
const (
	CertWarningThresholdDays  = 30
	CertCriticalThresholdDays = 7
)
