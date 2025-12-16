package e2e

import (
	"os/exec"
	"time"

	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// secretAutodiscoveryFallbackContext contains tests for secret autodiscovery fallback
// when one secret is deleted and another exists for the same domain in a different namespace.
//
// This test reproduces a bug where:
// 1. Two secrets exist in different namespaces with the same domain annotation
// 2. The first secret is loaded into domainSecrets index
// 3. The second secret is ignored (domain already exists in index)
// 4. When the first secret is deleted, the domain is removed from index
// 5. The second secret is NOT discovered as a fallback
// 6. VirtualService with autoDiscovery fails validation
func secretAutodiscoveryFallbackContext() {
	var fixture *fixtures.EnvoyFixture

	// Test data paths
	const (
		namespace1Path            = "test/testdata/e2e/secret_autodiscovery_fallback/namespace-1.yaml"
		namespace2Path            = "test/testdata/e2e/secret_autodiscovery_fallback/namespace-2.yaml"
		secretNs1Path             = "test/testdata/e2e/secret_autodiscovery_fallback/secret-ns1.yaml"
		secretNs2Path             = "test/testdata/e2e/secret_autodiscovery_fallback/secret-ns2.yaml"
		listenerPath              = "test/testdata/e2e/secret_autodiscovery_fallback/listener.yaml"
		virtualServicePath        = "test/testdata/e2e/secret_autodiscovery_fallback/virtual-service.yaml"
		virtualServiceExplicitRef = "test/testdata/e2e/secret_autodiscovery_fallback/virtual-service-explicit-ref.yaml"
	)

	BeforeEach(func() {
		By("setting up EnvoyFixture")
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)

		By("creating test namespaces")
		// Ensure any pending deletions from previous tests are complete
		time.Sleep(2 * time.Second)
		err := utils.ApplyManifests(namespace1Path)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace-1")
		err = utils.ApplyManifests(namespace2Path)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace-2")
		// Wait for namespaces to be fully ready
		time.Sleep(1 * time.Second)

		DeferCleanup(func() {
			By("cleaning up test namespaces")
			// Use --timeout to prevent hanging on namespace deletion with finalizers
			cmd := exec.Command("kubectl", "delete", "ns", "exc-secrets-ns1", "exc-secrets-ns2",
				"--ignore-not-found=true", "--timeout=30s", "--wait=false")
			_, _ = utils.Run(cmd)
		})
	})

	It("should find fallback secret after primary secret is deleted", func() {
		// This test reproduces the exact user scenario:
		// 1. Two secrets exist with the same domain in different namespaces
		// 2. VS uses autoDiscovery, picks up ns1 secret (first indexed)
		// 3. User switches VS to explicit secretRef to ns2
		// 4. User deletes ns1 secret (now allowed, VS explicitly references ns2)
		// 5. User switches VS back to autoDiscovery
		// 6. BUG: autodiscovery fails because domain was removed from index

		By("applying listener")
		fixture.ApplyManifests(listenerPath)

		By("applying first secret (ns1) - this will be indexed first")
		err := utils.ApplyManifests(secretNs1Path)
		Expect(err).NotTo(HaveOccurred(), "Failed to apply secret in ns1")

		// Give the controller time to index the secret
		time.Sleep(2 * time.Second)

		By("applying second secret (ns2) - same domain, different namespace")
		err = utils.ApplyManifests(secretNs2Path)
		Expect(err).NotTo(HaveOccurred(), "Failed to apply secret in ns2")

		// Give the controller time to process
		time.Sleep(2 * time.Second)

		By("applying VirtualService with autoDiscovery - should work with ns1 secret")
		fixture.ApplyManifests(virtualServicePath)

		// Wait for config to be applied
		fixture.WaitEnvoyConfigChanged()

		By("verifying VirtualService is configured correctly")
		listenerJsonPath := "configs.2.dynamic_listeners." +
			"#(name==\"default/https-autodiscovery-fallback\").active_state.listener"
		expectations := map[string]string{
			listenerJsonPath + ".name": "default/https-autodiscovery-fallback",
			listenerJsonPath + ".filter_chains.0.filter_chain_match.server_names.0": "autodiscovery-fallback.kaasops.io",
		}
		fixture.VerifyEnvoyConfig(expectations)

		By("switching VirtualService to explicit secretRef to ns2")
		// This allows us to delete ns1 secret without webhook blocking it
		fixture.ApplyManifests(virtualServiceExplicitRef)
		fixture.WaitEnvoyConfigChanged()

		By("deleting the first secret (ns1) - now allowed because VS references ns2 explicitly")
		err = utils.DeleteManifests(secretNs1Path)
		Expect(err).NotTo(HaveOccurred(), "Failed to delete secret in ns1")

		// Give the controller time to process the deletion
		time.Sleep(3 * time.Second)

		By("switching VirtualService back to autoDiscovery - THIS IS WHERE THE BUG MANIFESTS")
		// With the bug: this will FAIL because domainSecrets index lost the domain
		// After fix: this should SUCCEED because ns2 secret should be discovered as fallback
		err = utils.ApplyManifests(virtualServicePath)
		Expect(err).NotTo(HaveOccurred(),
			"VirtualService should be updated successfully - autodiscovery should find the secret from ns2. "+
				"If this fails with 'secret not found for domain', the bug is reproduced.")

		By("verifying the VirtualService is still working with the fallback secret from ns2")
		// Note: we don't call WaitEnvoyConfigChanged here because the config should remain the same
		// (both explicit ns2 reference and autodiscovery now select the same ns2 secret)
		fixture.VerifyEnvoyConfig(expectations)

		By("cleaning up secrets")
		_ = utils.DeleteManifests(secretNs2Path)
	})

	It("should validate VirtualService after secret migration between namespaces", func() {
		// This test verifies that when secrets are created in reverse order (ns2 first),
		// the bug still manifests. Tests a different code path.

		By("applying listener")
		fixture.ApplyManifests(listenerPath)

		By("applying second secret (ns2) first - this will be indexed first")
		err := utils.ApplyManifests(secretNs2Path)
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(2 * time.Second)

		By("applying first secret (ns1) - same domain, different namespace")
		err = utils.ApplyManifests(secretNs1Path)
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(2 * time.Second)

		By("applying VirtualService with autoDiscovery - should work with ns2 secret (indexed first)")
		fixture.ApplyManifests(virtualServicePath)
		fixture.WaitEnvoyConfigChanged()

		By("deleting VirtualService to allow secret deletion")
		fixture.DeleteManifests(virtualServicePath)
		// Give controller time to process deletion and update config baseline
		time.Sleep(3 * time.Second)
		// Capture current config as baseline for WaitEnvoyConfigChanged
		fixture.ConfigDump = fixture.GetEnvoyConfigDump("")

		By("deleting ns2 secret (simulating duplicate removal)")
		err = utils.DeleteManifests(secretNs2Path)
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(3 * time.Second)

		By("recreating VirtualService with autoDiscovery - should find ns1 secret")
		// With the bug: this will FAIL because domainSecrets index lost the domain
		// After fix: this should SUCCEED because ns1 secret should still work
		fixture.ApplyManifests(virtualServicePath)

		By("waiting for config to be applied with ns1 secret")
		fixture.WaitEnvoyConfigChanged()

		By("verifying VirtualService is configured correctly")
		listenerJsonPath := "configs.2.dynamic_listeners." +
			"#(name==\"default/https-autodiscovery-fallback\").active_state.listener"
		expectations := map[string]string{
			listenerJsonPath + ".name": "default/https-autodiscovery-fallback",
			listenerJsonPath + ".filter_chains.0.filter_chain_match.server_names.0": "autodiscovery-fallback.kaasops.io",
		}
		fixture.VerifyEnvoyConfig(expectations)

		By("cleaning up")
		_ = utils.DeleteManifests(secretNs1Path)
	})
}
