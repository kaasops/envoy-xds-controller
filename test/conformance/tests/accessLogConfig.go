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
	ConformanceTests = append(ConformanceTests, AccessLogConfig_CannotBeEmptyTest, AccessLogConfig_HasInvalidSpec, AccessLogConfig_DeleteUsed)
}

var AccessLogConfig_CannotBeEmptyTest = utils.TestCase{
	ShortName:          "AccessLogConfig_AlreadExistsTest",
	Description:        "Test that the AccessLogConfig can't be empty",
	Manifests:          []string{"../testdata/conformance/accesslogconfig-empty-spec.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.AccessLogConfigCannotBeEmptyMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var AccessLogConfig_HasInvalidSpec = utils.TestCase{
	ShortName:          "AccessLogConfig_HasInvalidSpec",
	Description:        "Test that the AccessLogConfig cannot be applied with invalid spec",
	Manifests:          []string{"../testdata/conformance/accesslogconfig-invalid-spec.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.UnmarshalMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var AccessLogConfig_DeleteUsed = utils.TestCase{
	ShortName:          "AccessLogConfig_DeleteUsed",
	Description:        "Test that the AccessLogConfig cannot be delted when used by a Virtual Service",
	Manifests:          []string{"../testdata/conformance/accesslogconfig-used-in-virtualservice.yaml"},
	ApplyErrorContains: "",
	Test: func(t *testing.T, suite *utils.TestSuite) {
		// Try deleting the AccessLogConfig
		err := suite.Client.Delete(context.TODO(), &v1alpha1.AccessLogConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "stdout",
				Namespace: suite.Namespace,
			},
		})

		require.ErrorContains(t, err, fmt.Sprintf("%v%v%v", ValidationErrorMessage, errors.AccessLogConfigDeleteUsedMessage, []string{"virtual-service-used-stdout-alc"}))
	},
}
