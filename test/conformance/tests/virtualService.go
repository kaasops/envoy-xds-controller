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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	ConformanceTests = append(
		ConformanceTests,
		VirtualService_SaveSecretWithCertificate_AutoDiscovery,
		VirtualService_VirtualHostCannotBeEmptyTest,
		VirtualService_InvalidVirtualHost,
		VirtualService_SaveSecretWithCertificate_SecretRef,
		VirtualService_SaveSecretWithCertificate_SecretRef_DiferentNamespaces,
		VirtualService_SaveSecretWithCertificate_AutoDiscovery_DiferentNamespaces,
		VirtualService_EmptyDomains,
		VirtualService_RBAC_Empty,
		VirtualService_RBAC_EmptyAction,
		VirtualService_RBAC_EmptyPermissions,
		VirtualService_RBAC_EmptyPolicies,
		VirtualService_RBAC_EmptyPolicy,
		VirtualService_RBAC_EmptyPrincipals,
		VirtualService_RBAC_InvalidAction,
		VirtualService_RBAC_UnknownAdditionalPolicy,
		VirtualService_RBAC_CollisionPoliciesNames,
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

var VirtualService_EmptyDomains = utils.TestCase{
	ShortName:          "VirtualService_EmptyDomains",
	Description:        "Test that the VirtualService cannot be applied with empty domains in VirtualHost spec",
	Manifests:          []string{"../testdata/conformance/virtualservice-empty-domains.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, errors.CannotValidateCacheResourceMessage),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var VirtualService_SaveSecretWithCertificate_SecretRef = utils.TestCase{
	ShortName:          "VirtualService_SaveSecretWithSertificate_SecretRef",
	Description:        "Test that the secret with sertificate cannot be deleted if used in VirtualService with tlsConfig.SecretRef",
	Manifests:          nil,
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
	Manifests:          nil,
	ApplyErrorContains: "",
	Test: func(t *testing.T, suite *utils.TestSuite) {
		deleteSecretWithCertificate_TEST(
			t,
			suite,
			"exc-kaasops-io", suite.Namespace, "../testdata/certificates/exc-kaasops-io.yaml", // Secret data
			"exc-kaasops-io-autodiscovery", "../testdata/conformance/virtualservice-secret-control-autoDiscovery.yaml", // Virtual Service data
		)
	},
}

var VirtualService_SaveSecretWithCertificate_SecretRef_DiferentNamespaces = utils.TestCase{
	ShortName:          "VirtualService_SaveSecretWithCertificate_SecretRef_DiferensNamespaces",
	Description:        "Test that the secret with sertificate cannot be deleted if used in VirtualService with tlsConfig.SecretRef, if Virtual Service and Secret exists in different namespaces",
	Manifests:          nil,
	ApplyErrorContains: "",
	Test: func(t *testing.T, suite *utils.TestSuite) {
		deleteSecretWithCertificate_TEST(
			t,
			suite,
			"exc-kaasops-io", "envoy-xds-controller-secretref-test", "../testdata/certificates/exc-kaasops-io.yaml", // Secret data
			"exc-kaasops-io-secretref", "../testdata/conformance/virtualservice-secret-control-secretRef-different-namespace.yaml", // Virtual Service data
		)
	},
}

var VirtualService_SaveSecretWithCertificate_AutoDiscovery_DiferentNamespaces = utils.TestCase{
	ShortName:          "VirtualService_SaveSecretWithCertificate_AutoDiscovery_DiferensNamespaces",
	Description:        "Test that the secret with sertificate cannot be deleted if used in VirtualService with tlsConfig.AutoDiscovery, if Virtual Service and Secret exists in different namespaces",
	Manifests:          nil,
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

var VirtualService_RBAC_Empty = utils.TestCase{
	ShortName:          "VirtualService_RBAC_Empty",
	Description:        "Test that the VirtualService has empty RBAC",
	Manifests:          []string{"../testdata/conformance/vsvc-rbac-empty.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, "RBAC action is empty"),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var VirtualService_RBAC_EmptyAction = utils.TestCase{
	ShortName:          "VirtualService_RBAC_EmptyAction",
	Description:        "Test that the VirtualService has empty action in RBAC",
	Manifests:          []string{"../testdata/conformance/vsvc-rbac-empty-action.yaml"},
	ApplyErrorContains: fmt.Sprintf("%v%v", ValidationErrorMessage, "RBAC action is empty"),
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var VirtualService_RBAC_EmptyPermissions = utils.TestCase{
	ShortName:          "VirtualService_RBAC_EmptyPermissions",
	Description:        "Test that the VirtualService has empty permissions in RBAC policy",
	Manifests:          []string{"../testdata/conformance/vsvc-rbac-empty-permissions.yaml"},
	ApplyErrorContains: "invalid Policy.Permissions: value must contain at least 1 item",
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var VirtualService_RBAC_EmptyPolicies = utils.TestCase{
	ShortName:          "VirtualService_RBAC_EmptyPolicies",
	Description:        "Test that the VirtualService has empty policies in RBAC",
	Manifests:          []string{"../testdata/conformance/vsvc-rbac-empty-policies.yaml"},
	ApplyErrorContains: "RBAC policies and additional policies is empty",
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var VirtualService_RBAC_EmptyPolicy = utils.TestCase{
	ShortName:          "VirtualService_RBAC_EmptyPolicy",
	Description:        "Test that the VirtualService has empty policy in RBAC policies",
	Manifests:          []string{"../testdata/conformance/vsvc-rbac-empty-policy.yaml"},
	ApplyErrorContains: "invalid Policy.Permissions: value must contain at least 1 item(s); invalid Policy.Principals: value must contain at least 1 item(s)",
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var VirtualService_RBAC_EmptyPrincipals = utils.TestCase{
	ShortName:          "VirtualService_RBAC_EmptyPrincipals",
	Description:        "Test that the VirtualService has empty principals in RBAC policy",
	Manifests:          []string{"../testdata/conformance/vsvc-rbac-empty-principals.yaml"},
	ApplyErrorContains: "invalid Policy.Principals: value must contain at least 1 item",
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var VirtualService_RBAC_InvalidAction = utils.TestCase{
	ShortName:          "VirtualService_RBAC_InvalidAction",
	Description:        "Test that the VirtualService has invalid action in RBAC",
	Manifests:          []string{"../testdata/conformance/vsvc-rbac-invalid-action.yaml"},
	ApplyErrorContains: "invalid RBAC action",
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var VirtualService_RBAC_UnknownAdditionalPolicy = utils.TestCase{
	ShortName:          "VirtualService_RBAC_UnknownAdditionalPolicy",
	Description:        "Test that the VirtualService has unknown additional policy in RBAC",
	Manifests:          []string{"../testdata/conformance/vsvc-rbac-unknown-additional-policy.yaml"},
	ApplyErrorContains: `Policy.envoy.kaasops.io "test" not found`,
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
}

var VirtualService_RBAC_CollisionPoliciesNames = utils.TestCase{
	ShortName:          "VirtualService_RBAC_CollisionPoliciesNames",
	Description:        "Test that the VirtualService contains policies with the same name",
	Manifests:          []string{"../testdata/conformance/vsvc-rbac-collision-policies-names.yaml"},
	ApplyErrorContains: `Policy.envoy.kaasops.io "demo-policy" not found`,
	Test:               func(t *testing.T, suite *utils.TestSuite) {},
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

	// Try to delete certificate
	err = suite.Client.Delete(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespaceName,
		},
	})
	require.ErrorContains(t, err, fmt.Sprintf("%v%v. It used in Virtual Service %v/%v", ValidationErrorMessage, errors.DeleteInKubernetesMessage, suite.Namespace, vsName))
}
