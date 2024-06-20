package tests

import (
	"testing"

	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	"github.com/stretchr/testify/require"
)

func init() {
	E2ETests = append(
		E2ETests,
		HTTP_StaticRoute,
	)
}

var HTTP_StaticRoute = utils.TestCase{
	ShortName:   "HTTP_StaticRoute",
	Description: "Test that the Envoy get configuration with static route from xDS",
	Manifests:   []string{"../testdata/e2e/virtualservice-static-route.yaml"},
	Test: func(t *testing.T, suite *utils.TestSuite) {
		vsName := "virtual-service-used-route"
		validAnswer := "{\"answer\":\"true\"}"

		envoyWaitConnectToXDS(t)

		require.True(t, routeExistInxDS(t, k8s.ResourceName(suite.Namespace, vsName)))

		answer := curl(t, HTTP_Method, nil, "/")
		require.Equal(t, answer, validAnswer)
	},
}
