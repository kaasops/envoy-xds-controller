package tests

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"

	dto "github.com/prometheus/client_model/go"
)

var (
	key_LDS_Version       = "envoy_listener_manager_lds_version"
	key_RQ_DirectResponse = "envoy_http_rq_direct_response"

	label_HTTPConMan = "envoy_http_conn_manager_prefix"

	HTTP_Method  = "HTTP"
	HTTPS_Method = "HTTPS"
)

func envoyIsReady(t *testing.T) bool {
	url := fmt.Sprintf("%s/%s", envoyAdminPannel(), "ready")

	req, err := http.Get(url)
	require.NoError(t, err)

	return req.StatusCode == http.StatusOK
}

func getEnvoyStats(t *testing.T) map[string]*dto.MetricFamily {
	url := fmt.Sprintf("%s/%s", envoyAdminPannel(), "stats/prometheus")

	req, err := http.Get(url)
	require.NoError(t, err)

	parser := expfmt.TextParser{}
	out, errParser := parser.TextToMetricFamilies(req.Body)
	require.NoError(t, errParser)

	return out
}

func envoyWaitConnectToXDS(t *testing.T) {
	err := retry.Do(
		func() error {
			stats := getEnvoyStats(t)
			ldsVersionMetric := stats[key_LDS_Version].GetMetric()
			ldsVersionValue := *ldsVersionMetric[0].Gauge.Value

			if ldsVersionValue > 0 {
				return nil
			}

			return errors.New("envoy doesn't connect to xDS")
		},
		retry.Attempts(10),
	)

	require.NoError(t, err)
}

func routeExistInxDS(t *testing.T, routeName string) bool {

	stats := getEnvoyStats(t)
	routesMetrics := stats[key_RQ_DirectResponse].GetMetric()

	for _, rm := range routesMetrics {
		labels := rm.GetLabel()
		for _, l := range labels {
			if *l.Name == label_HTTPConMan {
				if *l.Value == routeName {
					return true
				}
			}
		}
	}

	return false
}

func curl(
	t *testing.T,
	method string,
	domain *string,
	path string,
) string {
	url := envoyHTTP_url(domain)
	if method == HTTPS_Method {
		url = envoyHTTPS_url(domain)
	}
	if path != "/" && path != "" {
		url = fmt.Sprintf("%s/%s", url, path)
	}

	client := &http.Client{}

	if method == HTTPS_Method {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Close = true

	if domain != nil {
		client.Transport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}

			if addr == fmt.Sprintf("%s:443", *domain) {
				addr = "127.0.0.1:443"
			}
			return dialer.DialContext(ctx, network, addr)
		}
	}

	res, err := client.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	return string(b)
}
