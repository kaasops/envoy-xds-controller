package tests

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

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

func curl(t *testing.T, method, path string) string {
	url := envoyHTTP_url()
	if method == HTTPS_Method {
		url = envoyHTTPS_url()
	}

	if path != "/" && path != "" {
		url = fmt.Sprintf("%s/%s", url, path)
	}

	req, err := http.Get(url)
	require.NoError(t, err)

	b, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	return string(b)
}
