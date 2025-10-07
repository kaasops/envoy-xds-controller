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
		routeConfigPath := "configs.4.dynamic_route_configs.#(route_config.name==\"envoy-xds-controller/test-status-vs\")"
		expectations := map[string]string{
			// Check that route config exists with correct name
			routeConfigPath + ".route_config.name": "envoy-xds-controller/test-status-vs",
			// Check that virtual host has the correct domain
			routeConfigPath + ".route_config.virtual_hosts.0.domains.0": "example.local",
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

		// Wait briefly for config to propagate to Envoy
		By("Waiting for config to propagate")
		time.Sleep(5 * time.Second)

		// Store the current snapshot version for comparison
		initialConfigDump := fixture.GetEnvoyConfigDump("")

		By("Step 2: Apply invalid VirtualService (with skip-validation annotation) and verify status changes")
		// Apply invalid VS with skip-validation annotation to bypass webhook
		// This allows testing status propagation even when webhooks are enabled
		err := utils.ApplyManifests("test/testdata/e2e/status_propagation/vs-invalid.yaml")
		Expect(err).NotTo(HaveOccurred(), "Failed to apply invalid VS with skip-validation annotation")

		// Verify status is marked as invalid by the controller
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
		// Give controller time to process the invalid VS and mark status as invalid
		// but NOT update the Envoy snapshot. Also wait for any pending reconciliations
		// from previous tests to settle.
		// Use a short Eventually to allow controller to process, then verify stability
		Eventually(func() bool {
			return true // Just wait for a bit
		}, 3*time.Second, 500*time.Millisecond).Should(BeTrue())

		// nolint: lll
		// Instead of checking entire snapshot equality (which can have timestamp/version changes),
		// verify that the specific VirtualService route config hasn't changed
		const routeConfigFilter = "configs.4.dynamic_route_configs.#(route_config.name==\"envoy-xds-controller/test-status-vs\").route_config"
		initialRouteConfig := gjson.Get(string(initialConfigDump), routeConfigFilter).String()

		// Verify the route config remains stable
		Consistently(func() string {
			dump := fixture.GetEnvoyConfigDump("")
			return gjson.Get(string(dump), routeConfigFilter).String()
		}, 5*time.Second, 1*time.Second).Should(Equal(initialRouteConfig),
			"VirtualService route config should remain stable when VS becomes invalid")

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
			routeConfigPath + ".route_config.name": "envoy-xds-controller/test-status-vs",
			// Check that virtual host has the updated domain
			routeConfigPath + ".route_config.virtual_hosts.0.domains.0": "updated.local",
		}
		fixture.VerifyEnvoyConfig(newExpectations)

		By("Test completed successfully: valid -> invalid (status only) -> valid (status + snapshot)")
	})
}
