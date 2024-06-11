package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func init() {
	ConformanceTests = append(
		ConformanceTests,
		VirtualService_VirtualHostCannotBeEmptyTest,
		VirtualService_InvalidVirtualHost,
		VirtualService_SaveSecretWithCertificate_SecretRef,
		VirtualService_SaveSecretWithCertificate_AutoDiscovery,
		VirtualService_SaveSecretWithCertificate_SecretRef_DiferentNamespaces,
		VirtualService_SaveSecretWithCertificate_AutoDiscovery_DiferentNamespaces,
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

var VirtualService_SaveSecretWithCertificate_SecretRef = utils.TestCase{
	ShortName:          "VirtualService_SaveSecretWithSertificate_SecretRef",
	Description:        "Test that the secret with sertificate cannot be deleted if used in VirtualService with tlsConfig.SecretRef",
	Manifests:          []string{"../testdata/certificates/exc-kaasops-io.yaml"},
	ApplyErrorContains: "",
	Test: func(t *testing.T, suite *utils.TestSuite) {
		deleteSecretWithCertificate_TEST(
			t,
			suite,
			"exc-kaasops-io", suite.Namespace, "../testdata/certificates/exc-kaasops-io.yaml", // Secret data
			"exc-kaasops-io-secretref", "../testdata/conformance/virtualservice-secret-control-secretRef.yaml", // Virtual Service data
		)
	},
}

var VirtualService_SaveSecretWithCertificate_AutoDiscovery = utils.TestCase{
	ShortName:          "VirtualService_SaveSecretWithCertificate_AutoDiscovery",
	Description:        "Test that the secret with sertificate cannot be deleted if used in VirtualService with tlsConfig.autoDiscovery",
	Manifests:          []string{"../testdata/certificates/exc-kaasops-io.yaml"},
	ApplyErrorContains: "",
	Test: func(t *testing.T, suite *utils.TestSuite) {
		deleteSecretWithCertificate_TEST(
			t,
			suite,
			"exc-kaasops-io", suite.Namespace, "../testdata/certificates/exc-kaasops-io.yaml", // Secret data
			"exc-kaasops-io-autodiscovery", "../testdata/conformance/virtualservice-secret-control-autodiscovery.yaml", // Virtual Service data
		)

	},
}

var VirtualService_SaveSecretWithCertificate_SecretRef_DiferentNamespaces = utils.TestCase{
	ShortName:          "VirtualService_SaveSecretWithCertificate_SecretRef_DiferensNamespaces",
	Description:        "Test that the secret with sertificate cannot be deleted if used in VirtualService with tlsConfig.SecretRef, if Virtual Service and Secret exists in different namespaces",
	Manifests:          []string{"../testdata/certificates/exc-kaasops-io.yaml"},
	ApplyErrorContains: "",
	Test: func(t *testing.T, suite *utils.TestSuite) {
		deleteSecretWithCertificate_TEST(
			t,
			suite,
			"exc-kaasops-io", "envoy-xds-controller-secretref-test", "../testdata/certificates/exc-kaasops-io.yaml", // Secret data
			"exc-kaasops-io-secretref", "../testdata/conformance/virtualservice-secret-control-secretRef.yaml", // Virtual Service data
		)
	},
}

var VirtualService_SaveSecretWithCertificate_AutoDiscovery_DiferentNamespaces = utils.TestCase{
	ShortName:          "VirtualService_SaveSecretWithCertificate_AutoDiscovery_DiferensNamespaces",
	Description:        "Test that the secret with sertificate cannot be deleted if used in VirtualService with tlsConfig.AutoDiscovery, if Virtual Service and Secret exists in different namespaces",
	Manifests:          []string{"../testdata/certificates/exc-kaasops-io.yaml"},
	ApplyErrorContains: "",
	Test: func(t *testing.T, suite *utils.TestSuite) {
		deleteSecretWithCertificate_TEST(
			t,
			suite,
			"exc-kaasops-io", "envoy-xds-controller-autodiscovery-test", "../testdata/certificates/exc-kaasops-io.yaml", // Secret data
			"exc-kaasops-io-autodiscovery", "../testdata/conformance/virtualservice-secret-control-autoDiscovery.yaml", // Virtual Service data
		)
	},
}

/**
	Special test cases
**/

func deleteSecretWithCertificate_TEST(
	t *testing.T,
	suite *utils.TestSuite,
	secretName, secretNamespaceName, secretPath string,
	vsName, vsPath string,
) {
	// If Namespace for secret not set - use suite Namespace
	if secretNamespaceName != suite.Namespace {
		// Create Namespace for secret if set special
		err := suite.Client.Create(context.TODO(), &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Namespace",
				"metadata": map[string]interface{}{
					"name": secretNamespaceName,
				},
			},
		})
		if !api_errors.IsAlreadyExists(err) {
			require.NoError(t, err)
		}
		defer func() {
			err = suite.Client.Delete(context.TODO(), &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Namespace",
					"metadata": map[string]interface{}{
						"name": secretNamespaceName,
					},
				},
			})
		}()
	}

	// Create secret with Certificate in special Namespace
	err := utils.ApplyManifest(
		suite.Client,
		secretPath,
		secretNamespaceName,
	)
	defer func() {
		err := utils.CleanupManifest(
			suite.Client,
			secretPath,
			secretNamespaceName,
		)
		require.NoError(t, err)
	}()
	require.NoError(t, err)

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

	// Try to delete certificate
	err = suite.Client.Delete(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: suite.Namespace,
		},
	})
	require.ErrorContains(t, err, fmt.Sprintf("%v%v. It used in Virtual Service %v/%v", ValidationErrorMessage, errors.DeleteInKubernetesMessage, suite.Namespace, vsName))
}
