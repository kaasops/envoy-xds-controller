package tests

import (
	"fmt"
	"testing"

	"github.com/kaasops/envoy-xds-controller/test/utils"
	"github.com/stretchr/testify/require"
)

func init() {
	E2ETests = append(
		E2ETests,
		E2E_AccessLogConfig_AutoGenerateFilename,
	)
}

var E2E_AccessLogConfig_AutoGenerateFilename = utils.TestCase{
	ShortName:   "E2E_AccessLogConfig_AutoGenerateFilename",
	Description: "Test that the AccessLogConfig can be applied with annotation envoy.kaasops.io/auto-generated-filename and path will be patched",
	Manifests:   []string{"../testdata/e2e/virtualservice-accesslog-auto-generate-filename.yaml"},
	Test: func(t *testing.T, suite *utils.TestSuite) {
		vsName := "accesslog-auto-generate-filename"

		envoyWaitConnectToXDS(t)

		configDump := getEnvoyConfigDump(t)
		require.Contains(t, configDump, fmt.Sprintf("/tmp/%s.log", vsName))
	},
}
