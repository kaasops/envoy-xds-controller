package config

import (
	"strings"

	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	CertManager struct {
		ClusterIssuer string `default:"" envconfig:"DEFAULT_CLUSTER_ISSUER"`
	}
	WatchNamespaces      string `default:""                     envconfig:"WATCH_NAMESPACES"`
	InstalationNamespace string `default:"envoy-xds-controller" envconfig:"INSTALATION_NAMESPACE"`
	XDS                  struct {
		Port int `default:"8888" envconfig:"XDS_PORT"`
	}
	Webhook struct {
		TLSSecretName  string `default:"envoy-xds-controller-tls"                    envconfig:"WEBHOOK_TLS_SECRET_NAME"`
		WebhookCfgName string `default:"envoy-xds-controller-validating-webhook-cfg" envconfig:"WEBHOOK_CFG_NAME"`
		ServiceName    string `default:"envoy-xds-controller-webhook-service"        envconfig:"SERVICE_NAME"`
		Path           string `default:"/validate"                                   envconfig:"WEBHOOK_PATH"`
		Port           int    `default:"9443"                                        envconfig:"WEBHOOK_PORT"`
	}
}

func New() (*Config, error) {
	var cfg Config

	err := envconfig.Process("APP", &cfg)
	return &cfg, errors.Wrap(err, "Cannot get configs from ENVs")
}

func (c *Config) GetWatchNamespaces() []string {
	if c.WatchNamespaces != "" {
		return strings.Split(c.WatchNamespaces, ",")
	}

	return nil
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
	return c.Webhook.WebhookCfgName
}

func (c *Config) GetValidationWebhookServiceName() string {
	return c.Webhook.ServiceName
}

func (c *Config) GetWebhookPath() string {
	return c.Webhook.Path
}

func (c *Config) GetWebhookPort() int {
	return c.Webhook.Port
}
