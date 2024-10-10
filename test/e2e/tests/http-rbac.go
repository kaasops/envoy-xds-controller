package tests

import (
	"testing"
	"time"

	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	"github.com/stretchr/testify/require"
)

func init() {
	E2ETests = append(
		E2ETests,
		//HTTP_RBAC,
		HTTPS_RBAC,
	)
}

//var HTTP_RBAC = utils.TestCase{
//	ShortName:   "HTTP_RBAC",
//	Description: "Test that the Envoy get configuration with rbac http filter from xDS",
//	Manifests:   []string{"../testdata/e2e/vs-rbac.yaml"},
//	Test: func(t *testing.T, suite *utils.TestSuite) {
//		vsName := "vs-rbac"
//		validAnswer := "{\"answer\":\"true\"}"
//
//		envoyWaitConnectToXDS(t)
//
//		require.True(t, routeExistInxDS(t, k8s.ResourceName(suite.Namespace, vsName)))
//
//		answer := curl(t, HTTP_Method, nil, "/")
//		require.Equal(t, validAnswer, answer)
//		cfgDump := getEnvoyConfigDump(t)
//		require.Contains(t, cfgDump, `"name": "exc.filters.http.rbac"`)
//	},
//}

var HTTPS_RBAC = utils.TestCase{
	ShortName:   "HTTPS_RBAC",
	Description: "Test that the Envoy get configuration with rbac https filter from xDS",
	Manifests:   nil,
	Test: func(t *testing.T, suite *utils.TestSuite) {
		firstSecretPath := "../testdata/certificates/exc-kaasops-io.yaml"
		firstDomain := "exc.kaasops.io"
		vsPath := "../testdata/e2e/vs-https-rbac.yaml"
		vsName := "vs-https-rbac"
		validAnswer := `{"answer":"true"}`

		err := utils.CreateSecretInNamespace(
			suite,
			firstSecretPath, suite.Namespace,
		)
		require.NoError(t, err)
		defer func() {
			err := utils.CleanupManifest(suite.Client, firstSecretPath, suite.Namespace)
			require.NoError(t, err)
		}()

		err = utils.ApplyManifest(
			suite.Client,
			vsPath,
			suite.Namespace,
		)
		defer func() {
			err := utils.CleanupManifest(
				suite.Client,
				vsPath,
				suite.Namespace,
			)
			require.NoError(t, err)
		}()
		require.NoError(t, err)

		// TODO: change wait to check status.valid!
		time.Sleep(2 * time.Second)

		envoyWaitConnectToXDS(t)

		require.True(t, routeExistInxDS(t, k8s.ResourceName(suite.Namespace, vsName)))

		answer1 := curl(t, HTTPS_Method, &firstDomain, "/")
		require.Equal(t, validAnswer, answer1)
		cfgDump := getEnvoyConfigDump(t)
		require.Contains(t, cfgDump, `"name": "exc.filters.http.rbac"`)
		answer2 := curl(t, HTTPS_Method, &firstDomain, "ping")
		require.Equal(t, "RBAC: access denied", answer2)
	},
}
