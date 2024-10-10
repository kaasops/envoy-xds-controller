package tests

import (
	"context"
	"fmt"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func init() {
	ConformanceTests = append(
		ConformanceTests,
		Policy_CannotBeEmptySpec,
		Policy_HasInvalidSpec,
		Policy_CannotDeleteUsed,
	)
}

var Policy_CannotBeEmptySpec = utils.TestCase{
	ShortName:          "Policy_CannotBeEmptySpec",
	Description:        "Test that the Policy can't be empty",
	Manifests:          []string{"../testdata/conformance/policy-spec-empty.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.PolicyCannotBeEmptyMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var Policy_HasInvalidSpec = utils.TestCase{
	ShortName:          "Policy_HasInvalidSpec",
	Description:        "Test that the Policy cannot be applied with invalid spec",
	Manifests:          []string{"../testdata/conformance/policy-spec-invalid.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.UnmarshalMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var Policy_CannotDeleteUsed = utils.TestCase{
	ShortName:          "Policy_CannotDeleteUsed",
	Description:        "Test that the Policy cannot be deleted when used by a Virtual Service",
	Manifests:          []string{"../testdata/conformance/vsvc-rbac-used-policy.yaml"},
	ApplyErrorContains: "",
	Test: func(t *testing.T, suite *utils.TestSuite) {
		err := suite.Client.Delete(context.TODO(), &v1alpha1.Policy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-policy",
				Namespace: suite.Namespace,
			},
		})
		require.ErrorContains(t, err, fmt.Sprintf("%v. It used in Virtual Service %v/%v", errors.DeleteInKubernetesMessage, "envoy-xds-controller-conformance", "vsvc-rbac-used-policy"))
	},
}
