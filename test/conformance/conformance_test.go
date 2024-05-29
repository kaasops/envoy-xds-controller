package conformance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/test/utils"

	"github.com/kaasops/envoy-xds-controller/test/conformance/tests"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	testNamespace     = "envoy-xds-controller-conformance"
	baseManifestsPath = "../testdata/base"
)

func TestConformance(t *testing.T) {
	cfg, err := config.GetConfig()
	require.NoError(t, err)

	c, err := client.New(cfg, client.Options{})
	require.NoError(t, err)

	v1alpha1.AddToScheme(c.Scheme())

	// Create test namespace
	err = c.Create(context.Background(), &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": testNamespace,
			},
		},
	})
	if !api_errors.IsAlreadyExists(err) {
		require.NoError(t, err)
	}

	// Apply base manifests
	err = utils.ApplyManifestsFromPath(c, baseManifestsPath, testNamespace)
	defer func() {
		err := utils.CleanupManifestsFromPath(c, baseManifestsPath, testNamespace)
		require.NoError(t, err)
	}()
	require.NoError(t, err)

	// Init tests
	for _, test := range tests.ConformanceTests {
		t.Run(test.ShortName, func(t *testing.T) {
			test.Run(t, &utils.TestSuite{
				Client:    c,
				Namespace: testNamespace,
			})
		})
	}
}
