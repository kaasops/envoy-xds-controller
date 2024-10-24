package tests

import (
	"context"
	"fmt"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	"github.com/stretchr/testify/require"
	"testing"
)

func init() {
	ConformanceTests = append(
		ConformanceTests,
		VirtualServiceTemplate_CannotDeleteLinkedResources,
	)
}

var VirtualServiceTemplate_CannotDeleteLinkedResources = utils.TestCase{
	ShortName:   "VirtualServiceTemplate_CannotDeleteLinkedResources",
	Description: "Test that you cannot delete the linked resources that is used in the template",
	Manifests: []string{
		"../testdata/conformance/templates_testdata/listener-for-template.yaml",
		"../testdata/conformance/templates_testdata/alc-for-template.yaml",
		"../testdata/conformance/templates_testdata/http-filter-for-template.yaml",
		"../testdata/conformance/templates_testdata/route-for-template.yaml",
		"../testdata/conformance/templates_testdata/virtual-service-template.yaml",
	},
	Test: func(t *testing.T, suite *utils.TestSuite) {
		listener := v1alpha1.Listener{}
		listener.Name = "listener-for-template"
		listener.Namespace = suite.Namespace
		err := suite.Client.Delete(context.TODO(), &listener)
		require.ErrorContains(t, err, "listener is used in Virtual Service Templates: [virtual-service-template]")

		alc := v1alpha1.AccessLogConfig{}
		alc.Name = "alc-for-template"
		alc.Namespace = suite.Namespace
		err = suite.Client.Delete(context.TODO(), &alc)
		require.ErrorContains(t, err, "access log config is used in Virtual Service Templates: [virtual-service-template]")

		httpFilter := v1alpha1.HttpFilter{}
		httpFilter.Name = "http-filter-for-template"
		httpFilter.Namespace = suite.Namespace
		err = suite.Client.Delete(context.TODO(), &httpFilter)
		require.ErrorContains(t, err, fmt.Sprintf("%s:%+v", errors.HTTPFilterUsedInVST, []string{"virtual-service-template"}))

		route := v1alpha1.Route{}
		route.Name = "route-for-template"
		route.Namespace = suite.Namespace
		err = suite.Client.Delete(context.TODO(), &route)
		require.ErrorContains(t, err, fmt.Sprintf("%s:%+v", errors.RouteUsedInVST, []string{"virtual-service-template"}))
	},
}
