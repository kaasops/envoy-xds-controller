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
		VirtualService_VirtualHostCannotBeEmptyTest,
		VirtualService_InvalidVirtualHost,
		VirtualService_SaveSecretWithCertificate_SecretRef,
		VirtualService_SaveSecretWithCertificate_AutoDiscovery,
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
		secretName := "exc-kaasops-io"
		vsPath := "../testdata/conformance/virtualservice-secret-control-secretRef.yaml"
		vsName := "exc-kaasops-io-secretref"

		// Apply Virtual Service
		err := utils.ApplyManifest(
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
		time.Sleep(5 * time.Second)

		// Try to delete certificate
		err = suite.Client.Delete(context.TODO(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: suite.Namespace,
			},
		})
		require.ErrorContains(t, err, fmt.Sprintf("%v%v. It used in Virtual Service %v/%v", ValidationErrorMessage, errors.DeleteInKubernetesMessage, suite.Namespace, vsName))
	},
}

var VirtualService_SaveSecretWithCertificate_AutoDiscovery = utils.TestCase{
	ShortName:          "VirtualService_SaveSecretWithCertificate_AutoDiscovery",
	Description:        "Test that the secret with sertificate cannot be deleted if used in VirtualService with tlsConfig.autoDiscovery",
	Manifests:          []string{"../testdata/certificates/exc-kaasops-io.yaml"},
	ApplyErrorContains: "",
	Test: func(t *testing.T, suite *utils.TestSuite) {
		secretName := "exc-kaasops-io"
		vsPath := "../testdata/conformance/virtualservice-secret-control-autoDiscovery.yaml"
		vsName := "exc-kaasops-io-autodiscovery"

		// Apply Virtual Service
		err := utils.ApplyManifest(
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
		time.Sleep(5 * time.Second)

		// Try to delete certificate
		err = suite.Client.Delete(context.TODO(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: suite.Namespace,
			},
		})
		require.ErrorContains(t, err, fmt.Sprintf("%v%v. It used in Virtual Service %v/%v", ValidationErrorMessage, errors.DeleteInKubernetesMessage, suite.Namespace, vsName))
	},
}
