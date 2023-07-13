package config

import "github.com/kelseyhightower/envconfig"

type Config interface {
	GetWatchNamespace() string
	GetDefaultIssuer() string
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
