package cache

// Cluster type
type Cluster struct {
	Name        string     `json:"name"`
	ClusterType string     `json:"type"`
	LbPolicy    string     `json:"lb_policy"`
	Endpoints   []Endpoint `json:"endpoints"`
}

type Endpoint struct {
	Address string `json:"address"`
	Port    uint32 `json:"port"`
}

// Route type
type Route struct {
	Name         string        `json:"name"`
	VirtualHosts []VirtualHost `json:"virtual_hosts"`
}

type VirtualHost struct {
	Domains               []string               `json:"domains"`
	Routes                []string               `json:"routes"`
	RequestsHeadersToAdds []RequestsHeadersToAdd `json:"requests_headers_to_adds"`
}

type RequestsHeadersToAdd struct {
	Action string `json:"action"`
	Header Heades `json:"header"`
}

type Heades struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Listener type
type Listener struct {
	Name         string        `json:"name"`
	Address      Address       `json:"address"`
	FilterChains []FilterChain `json:"filter_chains"`
}

type Address struct {
	Bind string `json:"bind"`
	Port uint32 `json:"port"`
}

type FilterChain struct {
	Name             string            `json:"name"`
	FilterChainMatch *FilterChainMatch `json:"filter_chain_match"`
	TransportSocket  *TransportSocket  `json:"transport_socket"`
	Filters          []Filter          `json:"filters"`
}

type FilterChainMatch struct {
	Domains []string `json:"domains"`
}

type TransportSocket struct {
	Name string `json:"name"`
}

type Filter struct {
	FType       string   `json:"type"`
	StatPrefix  string   `json:"stat_prefix"`
	HttpFilters []string `json:"http_filters"`
	RDS         *string  `json:"rds"`
	Route       *Route   `json:"route"`
	Cluster     *string  `json:"cluster"`
}

// Secret type
type Secret struct {
	Name string `json:"name"`
}
