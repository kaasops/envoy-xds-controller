package e2e

import (
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ValidationTestCase defines a test case for validation tests
type ValidationTestCase struct {
	Description   string
	ApplyBefore   []string
	Manifest      string
	ExpectedError string
	Cleanup       bool
}

// validationEnvoyContext contains tests for Envoy configuration validation
func validationEnvoyContext() {
	var fixture *fixtures.EnvoyFixture

	BeforeEach(func() {
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	It("should apply only valid manifests", func() {
		By("applying base manifests")
		baseManifests := []string{
			"test/testdata/base/listener-http.yaml",
			"test/testdata/base/listener-https.yaml",
			"test/testdata/base/route-static.yaml",
			"test/testdata/base/route-default.yaml",
			"test/testdata/base/httpfilter-router.yaml",
			"test/testdata/base/accesslogconfig-stdout.yaml",
		}
		fixture.ApplyManifests(baseManifests...)

		// Define validation test cases
		testCases := []ValidationTestCase{
			{
				Description:   "AccessLogConfig with invalid auto-generated filename annotation",
				Manifest:      "test/testdata/conformance/accesslogconfig-auto-generated-filename-not-bool.yaml",
				ExpectedError: v1alpha1.ErrInvalidAnnotationAutogenFilenameValue.Error(),
			},
			{
				Description:   "AccessLogConfig with auto-generated filename for stdout",
				Manifest:      "test/testdata/conformance/accesslogconfig-auto-generated-filename-stdout.yaml",
				ExpectedError: v1alpha1.ErrInvalidAccessLogConfigType.Error(),
			},
			{
				Description:   "AccessLogConfig with empty spec",
				Manifest:      "test/testdata/conformance/accesslogconfig-empty-spec.yaml",
				ExpectedError: v1alpha1.ErrSpecNil.Error(),
			},
			{
				Description:   "AccessLogConfig with invalid spec",
				Manifest:      "test/testdata/conformance/accesslogconfig-invalid-spec.yaml",
				ExpectedError: `unknown field "foo"`,
			},
			{
				Description:   "Cluster with empty spec",
				Manifest:      "test/testdata/conformance/cluster-empty-spec.yaml",
				ExpectedError: v1alpha1.ErrSpecNil.Error(),
			},
			{
				Description:   "Cluster with invalid spec",
				Manifest:      "test/testdata/conformance/cluster-invalid-spec.yaml",
				ExpectedError: `unknown field "foo"`,
			},
			{
				Description:   "HTTPFilter with empty spec",
				Manifest:      "test/testdata/conformance/httpfilter-empty-spec.yaml",
				ExpectedError: v1alpha1.ErrSpecNil.Error(),
			},
			{
				Description:   "HTTPFilter with invalid spec",
				Manifest:      "test/testdata/conformance/httpfilter-invalid-spec.yaml",
				ExpectedError: `unknown field "foo"`,
			},
			{
				Description:   "Policy with empty spec",
				Manifest:      "test/testdata/conformance/policy-spec-empty.yaml",
				ExpectedError: v1alpha1.ErrSpecNil.Error(),
			},
			{
				Description:   "Policy with invalid spec",
				Manifest:      "test/testdata/conformance/policy-spec-invalid.yaml",
				ExpectedError: `unknown field "field"`,
			},
			{
				Description:   "Route with empty spec",
				Manifest:      "test/testdata/conformance/route-empty-spec.yaml",
				ExpectedError: v1alpha1.ErrSpecNil.Error(),
			},
			{
				Description:   "Route with invalid spec",
				Manifest:      "test/testdata/conformance/route-invalid-spec.yaml",
				ExpectedError: `unknown field "foo"`,
			},
			{
				Description:   "VirtualService with empty nodeIDs",
				Manifest:      "test/testdata/conformance/virtualservice-empty-nodeids.yaml",
				ExpectedError: "nodeIDs is required",
			},
			{
				Description:   "VirtualService with empty domains",
				Manifest:      "test/testdata/conformance/virtualservice-empty-domains.yaml",
				ExpectedError: "invalid VirtualHost.Domains: value must contain at least 1 item(s)",
			},
			{
				Description:   "VirtualService with empty virtualhost",
				Manifest:      "test/testdata/conformance/virtualservice-empty-virtualhost.yaml",
				ExpectedError: "listener is nil",
			},
			{
				Description:   "VirtualService with invalid virtualhost",
				Manifest:      "test/testdata/conformance/virtualservice-invalid-virtualhost.yaml",
				ExpectedError: `unknown field "foo"`,
			},
			{
				Description:   "VirtualService with empty object virtualhost",
				Manifest:      "test/testdata/conformance/virtualservice-empty-object-virtualhost.yaml",
				ExpectedError: "invalid VirtualHost.Domains: value must contain at least 1 item(s)",
			},
			{
				Description:   "VirtualService with secret control autoDiscovery",
				ApplyBefore:   []string{"test/testdata/certificates/exc-kaasops-io.yaml"},
				Manifest:      "test/testdata/conformance/virtualservice-secret-control-autoDiscovery.yaml",
				ExpectedError: "",
				Cleanup:       true,
			},
			{
				Description:   "VirtualService with secret control secretRef",
				ApplyBefore:   []string{"test/testdata/certificates/exc-kaasops-io.yaml"},
				Manifest:      "test/testdata/conformance/virtualservice-secret-control-secretRef.yaml",
				ExpectedError: "",
				Cleanup:       true,
			},
			{
				Description:   "VirtualService with RBAC policy name collision",
				ApplyBefore:   []string{"test/testdata/conformance/misc/policy.yaml"},
				Manifest:      "test/testdata/conformance/vsvc-rbac-collision-policies-names.yaml",
				ExpectedError: `policy 'demo-policy' already exist in RBAC`,
			},
			{
				Description:   "VirtualService with empty RBAC",
				Manifest:      "test/testdata/conformance/vsvc-rbac-empty.yaml",
				ExpectedError: "rbac action is empty",
			},
			{
				Description:   "VirtualService with empty RBAC action",
				Manifest:      "test/testdata/conformance/vsvc-rbac-empty-action.yaml",
				ExpectedError: "rbac action is empty",
			},
			{
				Description:   "VirtualService with empty RBAC permissions",
				Manifest:      "test/testdata/conformance/vsvc-rbac-empty-permissions.yaml",
				ExpectedError: "invalid Policy.Permissions: value must contain at least 1 item(s)",
			},
			{
				Description:   "VirtualService with empty RBAC policies",
				Manifest:      "test/testdata/conformance/vsvc-rbac-empty-policies.yaml",
				ExpectedError: "rbac policies is empty",
			},
			{
				Description:   "VirtualService with empty RBAC policy",
				Manifest:      "test/testdata/conformance/vsvc-rbac-empty-policy.yaml",
				ExpectedError: "invalid Policy.Permissions: value must contain at least 1 item(s)",
			},
			{
				Description:   "VirtualService with empty RBAC principals",
				Manifest:      "test/testdata/conformance/vsvc-rbac-empty-principals.yaml",
				ExpectedError: "invalid Policy.Principals: value must contain at least 1 item(s)",
			},
			{
				Description:   "VirtualService with unknown additional policy",
				Manifest:      "test/testdata/conformance/vsvc-rbac-unknown-additional-policy.yaml",
				ExpectedError: "rbac policy default/test not found",
			},
			{
				Description:   "VirtualService with template not found",
				Manifest:      "test/testdata/conformance/vsvc-template-not-found.yaml",
				ExpectedError: "virtual service template default/unknown-template-name not found",
			},
			{
				Description:   "AccessLogConfig with auto-generated filename",
				Manifest:      "test/testdata/conformance/accesslogconfig-auto-generated-filename.yaml",
				ExpectedError: "",
				Cleanup:       true,
			},
			// TCP Proxy validation tests
			{
				Description:   "VirtualService with virtual host and filter chains",
				Manifest:      "test/testdata/conformance/tcp-proxy/vs-virtual-host.yaml",
				ExpectedError: "conflict: virtual host is set, but filter chains are found in listener",
				ApplyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				Description:   "VirtualService with additional routes and filter chains",
				Manifest:      "test/testdata/conformance/tcp-proxy/vs-additional-routes.yaml",
				ExpectedError: "conflict: additional routes are set, but filter chains are found in listener",
				ApplyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				Description:   "VirtualService with HTTP filters and filter chains",
				Manifest:      "test/testdata/conformance/tcp-proxy/vs-http-filters.yaml",
				ExpectedError: "conflict: http filters are set, but filter chains are found in listener",
				ApplyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				Description:   "VirtualService with additional HTTP filters and filter chains",
				Manifest:      "test/testdata/conformance/tcp-proxy/vs-additional-http-filters.yaml",
				ExpectedError: "conflict: additional http filters are set, but filter chains are found in listener",
				ApplyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				Description:   "VirtualService with TLS config and filter chains",
				Manifest:      "test/testdata/conformance/tcp-proxy/vs-tls-config.yaml",
				ExpectedError: "conflict: tls config is set, but filter chains are found in listener",
				ApplyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				Description:   "VirtualService with RBAC and filter chains",
				Manifest:      "test/testdata/conformance/tcp-proxy/vs-rbac.yaml",
				ExpectedError: "conflict: rbac is set, but filter chains are found in listener",
				ApplyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				Description:   "VirtualService with use remote address and filter chains",
				Manifest:      "test/testdata/conformance/tcp-proxy/vs-use-remote-address.yaml",
				ExpectedError: "conflict: use remote address is set, but filter chains are found in listener",
				ApplyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				Description:   "VirtualService with upgrade configs and filter chains",
				Manifest:      "test/testdata/conformance/tcp-proxy/vs-upgrade-configs.yaml",
				ExpectedError: "conflict: upgrade configs is set, but filter chains are found in listener",
				ApplyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				Description:   "VirtualService with access log and filter chains",
				Manifest:      "test/testdata/conformance/tcp-proxy/vs-access-log.yaml",
				ExpectedError: "conflict: access log is set, but filter chains are found in listener",
				ApplyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				Description:   "VirtualService with access log config and filter chains",
				Manifest:      "test/testdata/conformance/tcp-proxy/vs-access-log-config.yaml",
				ExpectedError: "conflict: access log config is set, but filter chains are found in listener",
				ApplyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				Description:   "VirtualService HTTPS without TLS",
				Manifest:      "test/testdata/conformance/virtual-service-https-without-tls.yaml",
				ExpectedError: "tls listener not configured, virtual service has not tls config",
			},
			{
				Description:   "VirtualService HTTP with TLS",
				Manifest:      "test/testdata/conformance/virtual-service-http-with-tls.yaml",
				ExpectedError: "listener is not tls, virtual service has tls config",
			},
			{
				Description:   "VirtualService template with multiple root routes",
				Manifest:      "test/testdata/e2e/template_validation/virtual-service-template-2.yaml",
				ExpectedError: "failed to apply VirtualServiceTemplate: multiple root routes found",
				ApplyBefore: []string{
					"test/testdata/e2e/template_validation/cert.yaml",
					"test/testdata/e2e/template_validation/listener.yaml",
					"test/testdata/e2e/template_validation/route.yaml",
					"test/testdata/e2e/template_validation/virtual-service-template-1.yaml",
					"test/testdata/e2e/template_validation/virtual-service.yaml",
				},
			},
		}

		// Run all test cases
		for _, tc := range testCases {
			By(tc.Description)

			// Apply prerequisite manifests if any
			if len(tc.ApplyBefore) > 0 {
				for _, manifest := range tc.ApplyBefore {
					err := utils.ApplyManifests(manifest)
					Expect(err).NotTo(HaveOccurred(), "Failed to apply prerequisite manifest: "+manifest)
				}
			}

			// Apply the test manifest and check for expected error
			if tc.ExpectedError == "" {
				// Should succeed
				err := utils.ApplyManifests(tc.Manifest)
				Expect(err).NotTo(HaveOccurred(), "Expected manifest to be applied successfully: "+tc.Manifest)
			} else {
				// Should fail with expected error
				err := utils.ApplyManifests(tc.Manifest)
				Expect(err).To(HaveOccurred(), "Expected manifest to be rejected: "+tc.Manifest)
				Expect(err.Error()).To(ContainSubstring(tc.ExpectedError),
					"Error message did not match expected: "+tc.ExpectedError)
			}

			// Clean up if needed
			if tc.Cleanup {
				err := utils.DeleteManifests(tc.Manifest)
				Expect(err).NotTo(HaveOccurred(), "Failed to clean up manifest: "+tc.Manifest)
			}

			// Clean up prerequisite manifests if any
			if len(tc.ApplyBefore) > 0 {
				for i := len(tc.ApplyBefore) - 1; i >= 0; i-- {
					err := utils.DeleteManifests(tc.ApplyBefore[i])
					Expect(err).NotTo(HaveOccurred(), "Failed to clean up prerequisite manifest: "+tc.ApplyBefore[i])
				}
			}
		}

		// Clean up base manifests
		for _, manifest := range baseManifests {
			err := utils.DeleteManifests(manifest)
			Expect(err).NotTo(HaveOccurred(), "Failed to clean up base manifest: "+manifest)
		}
	})
}
