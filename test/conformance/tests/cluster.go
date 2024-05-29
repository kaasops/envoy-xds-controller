package tests

import (
	"fmt"
	"testing"

	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/test/utils"
)

func init() {
	ConformanceTests = append(ConformanceTests, Cluster_CannotBeEmptyTest, Cluster_HasInvalidSpec)
}

var Cluster_CannotBeEmptyTest = utils.TestCase{
	ShortName:          "ClusterAlreadExistsTest",
	Description:        "Test that the Cluster can't be empty",
	Manifests:          []string{"../testdata/conformance/cluster-empty-spec.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.ClusterCannotBeEmptyMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var Cluster_HasInvalidSpec = utils.TestCase{
	ShortName:          "ClusterHasInvalidSpec",
	Description:        "Test that the Cluster cannot be applied with invalid spec",
	Manifests:          []string{"../testdata/conformance/cluster-invalid-spec.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.UnmarshalMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

// TODO: Add check cluster not used, when deleted
// var ClusterDeleteUsed = utils.TestCase{
// 	ShortName:          "CLusterDeleteUsed",
// 	Description:        "Test that the CLuster cannot be delted when used by a Virtual Service",
// 	Manifests:          []string{"../testdata/conformance/??????.yaml"},
// 	ApplyErrorContains: "",
// 	Test: func(t *testing.T, suite *utils.TestSuite) {
// 		// Try deleting the Cluster
// 		err := suite.Client.Delete(context.TODO(), &v1alpha1.Cluster{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      "static",
// 				Namespace: suite.Namespace,
// 			},
// 		})

// 		require.ErrorContains(t, err, fmt.Sprintf("%v%v%v", ValidationErrorMessage, errors.???, []string{"virtual-service-used-stdout-alc"}))
// 	},
// }
