package e2e

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

// statusPropagationContext contains tests for VirtualService status propagation
// when switching between valid, invalid, and valid configurations
func statusPropagationContext() {
	var fixture *fixtures.EnvoyFixture

	BeforeEach(func() {
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	It("should propagate status correctly when VS changes from valid to invalid to valid", func() {
		By("Step 1: Apply valid VirtualService and verify snapshot is created")
		fixture.ApplyManifests(
			"test/testdata/e2e/status_propagation/listener.yaml",
			"test/testdata/e2e/status_propagation/vs-valid.yaml",
		)

		// Wait for Envoy config to update with the valid VS
		fixture.WaitEnvoyConfigChanged()

		// Verify that the snapshot contains the valid VirtualService configuration
		// Note: Route config name is formatted as "namespace/virtualservice-name"
		// configs.4 is the dynamic_route_configs section
		expectations := map[string]string{
			// Check that route config exists with correct name
			"configs.4.dynamic_route_configs.#(route_config.name==\"envoy-xds-controller/test-status-vs\").route_config.name": "envoy-xds-controller/test-status-vs",
			// Check that virtual host has the correct domain
			"configs.4.dynamic_route_configs.#(route_config.name==\"envoy-xds-controller/test-status-vs\").route_config.virtual_hosts.0.domains.0": "example.local",
		}
		fixture.VerifyEnvoyConfig(expectations)

		// Get VirtualService status and verify it's valid
		By("Verifying VirtualService status is valid")
		Eventually(func() map[string]interface{} {
			cmd := exec.Command("kubectl", "get", "virtualservice", "test-status-vs", "-n", "envoy-xds-controller", "-o", "json")
			output, err := utils.Run(cmd)
			if err != nil {
				return nil
			}

			statusInvalid := gjson.Get(output, "status.invalid").Bool()
			statusMessage := gjson.Get(output, "status.message").String()

			return map[string]interface{}{
				"invalid": statusInvalid,
				"message": statusMessage,
			}
		}, 30*time.Second, 2*time.Second).Should(Equal(map[string]interface{}{
			"invalid": false,
			"message": "",
		}))

		// Store the current snapshot version for comparison
		initialConfigDump := fixture.ConfigDump

		By("Step 2: Apply invalid VirtualService (missing domains) and verify status changes")
		// Apply invalid VS - this should be rejected by webhook if enabled,
		// or accepted but marked invalid in status if webhook is disabled
		err := utils.ApplyManifests("test/testdata/e2e/status_propagation/vs-invalid.yaml")

		// Check if webhook rejected it (webhook enabled case)
		if err != nil {
			By("Webhook rejected invalid VS (expected if webhook is enabled)")
			Expect(err.Error()).To(ContainSubstring("invalid VirtualHost.Domains"))
			// Skip the rest of this test case since webhook prevents invalid VS from being stored
			Skip("Webhook is enabled - cannot test status propagation for invalid VS")
		}

		// Webhook is disabled - verify status is marked as invalid
		By("Verifying VirtualService status is now invalid")
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "get", "virtualservice", "test-status-vs", "-n", "envoy-xds-controller", "-o", "json")
			output, err := utils.Run(cmd)
			if err != nil {
				return false
			}

			statusInvalid := gjson.Get(output, "status.invalid").Bool()
			statusMessage := gjson.Get(output, "status.message").String()

			// Verify the error message is concise (our getRootCause improvement)
			if statusInvalid && statusMessage != "" {
				By(fmt.Sprintf("Status message: %s", statusMessage))
				// Should contain the root error without the full chain
				Expect(statusMessage).To(ContainSubstring("VirtualHost.Domains"))
				Expect(statusMessage).NotTo(ContainSubstring("MainBuilder.BuildResources failed"))
				return true
			}
			return false
		}, 30*time.Second, 2*time.Second).Should(BeTrue())

		By("Verifying snapshot has NOT changed (invalid VS should not update Envoy config)")
		// Give some time for controller to potentially update (it shouldn't)
		time.Sleep(5 * time.Second)

		currentConfigDump := fixture.GetEnvoyConfigDump("")
		// Snapshots should remain the same
		Expect(string(currentConfigDump)).To(Equal(string(initialConfigDump)),
			"Snapshot should not change when VS becomes invalid")

		By("Step 3: Apply valid VirtualService again and verify status and snapshot update")
		fixture.ApplyManifests("test/testdata/e2e/status_propagation/vs-valid-updated.yaml")

		// Wait for Envoy config to update
		fixture.WaitEnvoyConfigChanged()

		// Verify status is valid again
		By("Verifying VirtualService status is valid again")
		Eventually(func() map[string]interface{} {
			cmd := exec.Command("kubectl", "get", "virtualservice", "test-status-vs", "-n", "envoy-xds-controller", "-o", "json")
			output, err := utils.Run(cmd)
			if err != nil {
				return nil
			}

			statusInvalid := gjson.Get(output, "status.invalid").Bool()
			statusMessage := gjson.Get(output, "status.message").String()

			return map[string]interface{}{
				"invalid": statusInvalid,
				"message": statusMessage,
			}
		}, 30*time.Second, 2*time.Second).Should(Equal(map[string]interface{}{
			"invalid": false,
			"message": "",
		}))

		// Verify that the snapshot has been updated with new configuration
		newExpectations := map[string]string{
			// Check that route config still exists
			"configs.4.dynamic_route_configs.#(route_config.name==\"envoy-xds-controller/test-status-vs\").route_config.name": "envoy-xds-controller/test-status-vs",
			// Check that virtual host has the updated domain
			"configs.4.dynamic_route_configs.#(route_config.name==\"envoy-xds-controller/test-status-vs\").route_config.virtual_hosts.0.domains.0": "updated.local",
		}
		fixture.VerifyEnvoyConfig(newExpectations)

		By("Test completed successfully: valid -> invalid (status only) -> valid (status + snapshot)")
	})
}
