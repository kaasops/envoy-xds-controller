package tests

import (
	"encoding/json"
	"fmt"
	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func init() {
	E2ETests = append(
		E2ETests,
		HTTPS_TEMPLATE,
	)
}

var HTTPS_TEMPLATE = utils.TestCase{
	ShortName:   "HTTPS_TEMPLATE",
	Description: "Validation of the correct application of the template to the virtual service",
	Manifests:   nil,
	Test: func(t *testing.T, suite *utils.TestSuite) {
		firstSecretPath := "../testdata/certificates/exc-kaasops-io.yaml"
		domain := "exc.kaasops.io"
		vsPath := "../testdata/e2e/vsvc-template-domain.yaml"
		vsName := "virtual-service-from-template"
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

		answer1 := curl(t, HTTPS_Method, &domain, "/")
		require.Equal(t, validAnswer, answer1)
		cfgDump := getEnvoyConfigDump(t)
		tmp := make(map[string]any)
		err = json.Unmarshal([]byte(cfgDump), &tmp)
		require.NoError(t, err)

		fmt.Println(cfgDump)
		fmt.Println(tmp)

	},
}
