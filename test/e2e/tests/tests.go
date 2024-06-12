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
)

func envoyAdminPannel() string {
	return fmt.Sprintf("%s://%s:%s", envoyAdminScheme, envoyAddress, envoyAdminPort)
}
