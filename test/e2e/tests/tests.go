package tests

import (
	"fmt"

	"github.com/kaasops/envoy-xds-controller/test/utils"
)

var (
	E2ETests               []utils.TestCase
	ValidationErrorMessage = "admission webhook \"validate.envoy.kaasops.io\" denied the request: "

	envoyAddress     = "127.0.0.1"
	envoyAdminScheme = "http"
	envoyAdminPort   = "19000"
	envoyHTTPPort    = "80"
	envoyHTTPSPort   = "443"
)

func envoyAdminPannel() string {
	return fmt.Sprintf("%s://%s:%s", envoyAdminScheme, envoyAddress, envoyAdminPort)
}

func envoyHTTP_url() string {
	return fmt.Sprintf("%s://%s:%s", "http", envoyAddress, envoyHTTPPort)
}

func envoyHTTPS_url() string {
	return fmt.Sprintf("%s://%s:%s", "https", envoyAddress, envoyHTTPSPort)
}
