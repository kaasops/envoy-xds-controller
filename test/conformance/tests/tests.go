package tests

import (
	"github.com/kaasops/envoy-xds-controller/test/utils"
)

var (
	ConformanceTests       []utils.TestCase
	ValidationErrorMessage = "admission webhook \"validate.envoy.kaasops.io\" denied the request: "
)
