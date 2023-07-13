package config

import "github.com/kelseyhightower/envconfig"

type Config interface {
	GetWatchNamespace() string
	GetDefaultIssuer() string
	GetXDSPort() int
}

const (
	IssuerType        = "Issuer"
	ClusterIssuerType = "ClusterIssuer"
)

type config struct {
	CertManager struct {
		ClusterIssuer string `default:"" envconfig:"DEFAULT_CLUSTER_ISSUER"`
	}
	WatchNamespace string `default:"" envconfig:"WATCH_NAMESPACE"`
	XDS            struct {
		Port int `default:"8888" envconfig:"XDS_PORT"`
	}
}

func New() (Config, error) {
	var cfg config

	err := envconfig.Process("APP", &cfg)
	return &cfg, err
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
