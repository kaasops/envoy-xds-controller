package fake

type Config struct {
	CertManager struct {
		ClusterIssuer string `default:"" envconfig:"DEFAULT_CLUSTER_ISSUER"`
	}
	WatchNamespace string `default:"" envconfig:"WATCH_NAMESPACE"`
	XDS            struct {
		Port int `default:"8888" envconfig:"XDS_PORT"`
	}
}

func New(watchNamespace, defaultIssuer string, xdsPort int) *Config {
	var cfg Config

	cfg.WatchNamespace = watchNamespace
	cfg.CertManager.ClusterIssuer = defaultIssuer
	cfg.XDS.Port = xdsPort

	return &cfg
}

func (c *Config) GetWatchNamespace() string {
	return c.WatchNamespace
}

func (c *Config) GetDefaultIssuer() string {
	return c.CertManager.ClusterIssuer
}

func (c *Config) GetXDSPort() int {
	return c.XDS.Port
}
