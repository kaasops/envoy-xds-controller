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
		HttpFilter_CannotBeEmptyTest,
		HttpFilter_HasInvalidSpec,
		HttpFilter_DeleteUsed,
		HttpFilter_OAuth2InvalidParamsCombination,
	)
}

var HttpFilter_CannotBeEmptyTest = utils.TestCase{
	ShortName:          "HttpFilterAlreadExistsTest",
	Description:        "Test that the HttpFilter can't be empty",
	Manifests:          []string{"../testdata/conformance/httpfilter-empty-spec.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.HTTPFilterCannotBeEmptyMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var HttpFilter_HasInvalidSpec = utils.TestCase{
	ShortName:          "HttpFilterHasInvalidSpec",
	Description:        "Test that the HttpFilter cannot be applied with invalid spec",
	Manifests:          []string{"../testdata/conformance/httpfilter-invalid-spec.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.UnmarshalMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var HttpFilter_DeleteUsed = utils.TestCase{
	ShortName:          "HttpFilterDeleteUsed",
	Description:        "Test that the HttpFilter cannot be delted when used by a Virtual Service",
	Manifests:          []string{"../testdata/conformance/httpfilter-used-in-virtualservice.yaml"},
	ApplyErrorContains: "",
	Test: func(t *testing.T, suite *utils.TestSuite) {
		// Try deleting the HttpFilter
		err := suite.Client.Delete(context.TODO(), &v1alpha1.HttpFilter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "router",
				Namespace: suite.Namespace,
			},
		})

		require.ErrorContains(t, err, fmt.Sprintf("%v%v%v", ValidationErrorMessage, errors.HTTPFilterDeleteUsed, []string{"virtual-service-used-hf"}))
	},
}

var HttpFilter_OAuth2InvalidParamsCombination = utils.TestCase{
	ShortName:          "HttpFilterOauth2InvalidParamsCombination",
	Description:        "Test that the HttpFilter contains an invalid combination of parameters",
	Manifests:          []string{"../testdata/conformance/httpfilter-oauth2-invalid-combination.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.InvalidParamsCombination),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}
