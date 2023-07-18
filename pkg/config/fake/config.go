package fake

type config struct {
	CertManager struct {
		ClusterIssuer string `default:"" envconfig:"DEFAULT_CLUSTER_ISSUER"`
	}
	WatchNamespace string `default:"" envconfig:"WATCH_NAMESPACE"`
	XDS            struct {
		Port int `default:"8888" envconfig:"XDS_PORT"`
	}
}

func New(watchNamespace, defaultIssuer string, xdsPort int) *config {
	var cfg config

	cfg.WatchNamespace = watchNamespace
	cfg.CertManager.ClusterIssuer = defaultIssuer
	cfg.XDS.Port = xdsPort

	return &cfg
}

func (c *config) GetWatchNamespace() string {
	return c.WatchNamespace
}

func (c *config) GetDefaultIssuer() string {
	return c.CertManager.ClusterIssuer
}

func (c *config) GetXDSPort() int {
	return c.XDS.Port
}
