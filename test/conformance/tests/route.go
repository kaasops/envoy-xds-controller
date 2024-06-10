package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	ConformanceTests = append(
		ConformanceTests,
		Route_CannotBeEmptyTest,
		Route_HasInvalidSpec,
		Route_DeleteUsed,
	)
}

var Route_CannotBeEmptyTest = utils.TestCase{
	ShortName:          "RouteAlreadExistsTest",
	Description:        "Test that the Route can't be empty",
	Manifests:          []string{"../testdata/conformance/route-empty-spec.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.RouteCannotBeEmptyMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var Route_HasInvalidSpec = utils.TestCase{
	ShortName:          "RouteHasInvalidSpec",
	Description:        "Test that the Route cannot be applied with invalid spec",
	Manifests:          []string{"../testdata/conformance/route-invalid-spec.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.UnmarshalMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var Route_DeleteUsed = utils.TestCase{
	ShortName:          "RouteDeleteUsed",
	Description:        "Test that the Route cannot be delted when used by a Virtual Service",
	Manifests:          []string{"../testdata/conformance/route-used-in-virtualservice.yaml"},
	ApplyErrorContains: "",
	Test: func(t *testing.T, suite *utils.TestSuite) {
		// Try deleting the Route
		err := suite.Client.Delete(context.TODO(), &v1alpha1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "static",
				Namespace: suite.Namespace,
			},
		})

		require.ErrorContains(t, err, fmt.Sprintf("%v%v%v", ValidationErrorMessage, errors.RouteDeleteUsed, []string{"virtual-service-used-route"}))
	},
}
