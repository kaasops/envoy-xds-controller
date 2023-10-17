package config

import (
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	CertManager struct {
		ClusterIssuer string `default:"" envconfig:"DEFAULT_CLUSTER_ISSUER"`
	}
	WatchNamespace       string `default:""                     envconfig:"WATCH_NAMESPACE"`
	InstalationNamespace string `default:"envoy-xds-controller" envconfig:"INSTALATION_NAMESPACE"`
	XDS                  struct {
		Port int `default:"8888" envconfig:"XDS_PORT"`
	}
	Webhook struct {
		Disable                  bool   `default:"false"                                       envconfig:"WEBHOOK_DISABLE"`
		TLSSecretName            string `default:"envoy-xds-controller-tls"                    envconfig:"TLS_SECRET_NAME"`
		ValidatingWebhookCfgName string `default:"envoy-xds-controller-validating-webhook-cfg" envconfig:"VALIDATING_WEBHOOK_CFG_NAME"`
		Port                     int    `default:"9443"                                        envconfig:"WEBHOOK_PORT"`
	}
}

func New() (*Config, error) {
	var cfg Config

	err := envconfig.Process("APP", &cfg)
	return &cfg, errors.Wrap(err, "Cannot get configs from ENVs")
}

func (c *Config) GetWatchNamespace() string {
	return c.WatchNamespace
}

func (c *Config) GetInstalationNamespace() string {
	return c.InstalationNamespace
}

func (c *Config) GetDefaultIssuer() string {
	return c.CertManager.ClusterIssuer
}

func (c *Config) GetXDSPort() int {
	return c.XDS.Port
}

func (c *Config) GetTLSSecretName() string {
	return c.Webhook.TLSSecretName
}

func (c *Config) GetValidatingWebhookCfgName() string {
	return c.Webhook.ValidatingWebhookCfgName
}

func (c *Config) GerWebhookPort() int {
	return c.Webhook.Port
}
