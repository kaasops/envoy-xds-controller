package tests

import (
	"context"
	"testing"
	"time"

	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func init() {
	E2ETests = append(
		E2ETests,
		HTTPS_StaticRoute,
		HTTPS_SecretAndVirualServiceInDifferentNamespaces,
		HTTPS_VirtualServiceWith2DomainsWithDiffCerts,
	)
}

var HTTPS_StaticRoute = utils.TestCase{
	ShortName:   "HTTPS_StaticRoute",
	Description: "Test that the Envoy get configuration with static route from xDS for https",
	Manifests:   nil,
	Test: func(t *testing.T, suite *utils.TestSuite) {
		request_TEST(
			t,
			suite,
			suite.Namespace, "../testdata/certificates/exc-kaasops-io.yaml", // Secret data
			"virtual-service-used-route-https", "../testdata/e2e/virtualservice-static-route-https.yaml",
			ptr.To("exc.kaasops.io"),
			"{\"answer\":\"true\"}",
		)
	},
}

var HTTPS_SecretAndVirualServiceInDifferentNamespaces = utils.TestCase{
	ShortName:   "HTTPS_SecretAndVirualServiceInDifferentNamespaces",
	Description: "Test that the Envoy get configuration with static route from xDS for https, when secret with certificate exist in different namespace",
	Manifests:   nil,
	Test: func(t *testing.T, suite *utils.TestSuite) {
		request_TEST(
			t,
			suite,
			"envoy-xds-controller-test-secrets", "../testdata/certificates/exc-kaasops-io.yaml", // Secret data
			"virtual-service-used-route-https", "../testdata/e2e/virtualservice-static-route-https.yaml", // Virtual Service data
			ptr.To("exc.kaasops.io"),
			"{\"answer\":\"true\"}",
		)
	},
}

var HTTPS_VirtualServiceWith2DomainsWithDiffCerts = utils.TestCase{
	ShortName:   "HTTPS_VirtualServiceWith2DomainsWithDiffCerts",
	Description: "Test that the Envoy get configuration with static route from xDS for https, when Virtual Service has 2 domains with different certificates. TLS config type - Autodiscovery",
	Manifests:   nil,
	Test: func(t *testing.T, suite *utils.TestSuite) {
		firstSecretPath := "../testdata/certificates/exc-kaasops-io.yaml"
		secondSecretPath := "../testdata/certificates/exc2-kaasops-io.yaml"
		firstDomain := "exc.kaasops.io"
		secondDomain := "exc2.kaasops.io"
		vsPath := "../testdata/e2e/virtualservice-static-route-https-2-domain-diff-certs.yaml"
		vsName := "virtual-service-used-route-https-2-domain-diff-certs"
		validAnswer := "{\"answer\":\"true\"}"

		// Create first Secret with certificate
		err := utils.CreateSecretInNamespace(
			suite,
			firstSecretPath, suite.Namespace,
		)
		require.NoError(t, err)
		defer func() {
			// Cleanup secret with certificate
			err := utils.CleanupManifest(suite.Client, firstSecretPath, suite.Namespace)
			require.NoError(t, err)
		}()

		// Create second Secret with certificate
		err = utils.CreateSecretInNamespace(
			suite,
			secondSecretPath, suite.Namespace,
		)
		require.NoError(t, err)
		defer func() {
			// Cleanup secret with certificate
			err := utils.CleanupManifest(suite.Client, secondSecretPath, suite.Namespace)
			require.NoError(t, err)
		}()

		// Apply Virtual Service
		err = utils.ApplyManifest(
			suite.Client,
			vsPath,
			suite.Namespace,
		)
		defer func() {
			err := utils.CleanupManifest(
				suite.Client,
				vsPath,
				suite.Namespace,
			)
			require.NoError(t, err)
		}()
		require.NoError(t, err)

		// TODO: change wait to check status.valid!
		time.Sleep(2 * time.Second)

		envoyWaitConnectToXDS(t)

		// Check route in xDS
		require.True(t, routeExistInxDS(t, k8s.ResourceName(suite.Namespace, vsName)))

		// Get http Request for first domain
		answer1 := curl(t, HTTPS_Method, &firstDomain, "/")
		require.Equal(t, answer1, validAnswer)

		// Get http Request for second domain
		answer2 := curl(t, HTTPS_Method, &secondDomain, "/")
		require.Equal(t, answer2, validAnswer)

	},
}

/**
	Special test cases
**/

func request_TEST(
	t *testing.T,
	suite *utils.TestSuite,
	secretNamespaceName, secretPath string,
	vsName, vsPath string,
	domain *string,
	validAnswer string,
) {
	err := utils.CreateSecretInNamespace(
		suite,
		secretPath, secretNamespaceName,
	)
	require.NoError(t, err)
	defer func() {
		// Cleanup secret with certificate
		err := utils.CleanupManifest(suite.Client, secretPath, secretNamespaceName)
		require.NoError(t, err)
	}()

	time.Sleep(3 * time.Second)

	// If used special Namespace - delete it
	if secretNamespaceName != suite.Namespace {
		defer func() {
			err := utils.CleanupNamespace(context.TODO(), suite.Client, secretNamespaceName)
			require.NoError(t, err)
		}()
	}

	// Apply Virtual Service
	err = utils.ApplyManifest(
		suite.Client,
		vsPath,
		suite.Namespace,
	)
	defer func() {
		err := utils.CleanupManifest(
			suite.Client,
			vsPath,
			suite.Namespace,
		)
		require.NoError(t, err)
	}()
	require.NoError(t, err)

	// TODO: change wait to check status.valid!
	time.Sleep(2 * time.Second)

	envoyWaitConnectToXDS(t)

	// Check route in xDS
	require.True(t, routeExistInxDS(t, k8s.ResourceName(suite.Namespace, vsName)))

	// Get http Request
	answer := curl(t, HTTPS_Method, domain, "/")
	require.Equal(t, answer, validAnswer)
}
