package tests

import (
	"fmt"
	"testing"

	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/test/utils"
)

func init() {
	ConformanceTests = append(
		ConformanceTests,
		VirtualService_VirtualHostCannotBeEmptyTest,
		VirtualService_InvalidVirtualHost,
	)
}

var VirtualService_VirtualHostCannotBeEmptyTest = utils.TestCase{
	ShortName:          "VirtualService_VirtualHostCannotBeEmptyTest",
	Description:        "Test that the VirtualHost in VirtualService can't be empty",
	Manifests:          []string{"../testdata/conformance/virtualservice-empty-virtualhost.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.VirtualHostCantBeEmptyMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var VirtualService_InvalidVirtualHost = utils.TestCase{
	ShortName:          "VirtualService_InvalidVirtualHost",
	Description:        "Test that the VirtualService cannot be applied with invalid VirtualHost spec",
	Manifests:          []string{"../testdata/conformance/virtualservice-invalid-virtualhost.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.UnmarshalMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}
