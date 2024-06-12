package tests

import (
	"testing"

	"github.com/kaasops/envoy-xds-controller/test/utils"
	"github.com/stretchr/testify/require"
)

func init() {
	E2ETests = append(
		E2ETests,
		Base_EnvoyWorking,
	)
}

var Base_EnvoyWorking = utils.TestCase{
	ShortName:   "Base_EnvoyWorking",
	Description: "Test that the pod with Envoy started and ready for working",
	Test: func(t *testing.T, suite *utils.TestSuite) {
		require.True(t, envoyIsReady(t))
	},
}
