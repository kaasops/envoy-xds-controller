package e2e

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// nodeIDWithVersions represents a single node's version info from /api/v1/nodeIDs/versions
type nodeIDWithVersions struct {
	NodeID   string            `json:"node_id"`
	Versions map[string]string `json:"versions"`
}

const (
	// Keys returned by the /api/v1/nodeIDs/versions endpoint
	listenersKey = "listeners"
	clustersKey  = "clusters"
	routesKey    = "routes"
)

// snapshotVersionStabilityContext contains tests for snapshot version stability
func snapshotVersionStabilityContext() {
	var fixture *fixtures.EnvoyFixture

	BeforeEach(func() {
		By("setting up EnvoyFixture")
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	It("should not increment versions when VirtualService is re-applied without changes", func() {
		By("applying initial manifests")
		manifests := []string{
			"test/testdata/e2e/basic_https_service/listener.yaml",
			"test/testdata/e2e/basic_https_service/tls-cert.yaml",
			"test/testdata/e2e/basic_https_service/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)
		fixture.WaitEnvoyConfigChanged()

		By("getting initial versions")
		initialVersions := getVersionsForNode("test")
		Expect(initialVersions).NotTo(BeNil(), "Should get initial versions")
		Expect(initialVersions[listenersKey]).NotTo(BeEmpty(), "Initial listeners version should not be empty")

		By("triggering reconciliation by adding a test annotation")
		// Adding an annotation forces K8s to update resourceVersion and trigger reconciliation
		// This is more reliable than re-applying the same manifest (which K8s may skip)
		triggerReconciliation("virtualservice", "virtual-service")

		By("verifying versions remain stable after reconciliation")
		// Use Consistently to verify versions don't change over a period of time
		// This ensures the controller has had time to process and we're not just
		// checking before reconciliation completes
		Consistently(func() map[string]string {
			return getVersionsForNode("test")
		}, 5*time.Second, 500*time.Millisecond).Should(SatisfyAll(
			HaveKeyWithValue(listenersKey, initialVersions[listenersKey]),
			HaveKeyWithValue(routesKey, initialVersions[routesKey]),
			HaveKeyWithValue(clustersKey, initialVersions[clustersKey]),
		), "Versions should remain unchanged after reconciliation without spec changes")
	})

	It("should not increment versions when VirtualServiceTemplate is re-applied without changes", func() {
		By("applying template-based manifests")
		manifests := []string{
			"test/testdata/e2e/virtual_service_templates/listener.yaml",
			"test/testdata/e2e/virtual_service_templates/tls-cert.yaml",
			"test/testdata/e2e/virtual_service_templates/http-filter.yaml",
			"test/testdata/e2e/virtual_service_templates/access-log-config.yaml",
			"test/testdata/e2e/virtual_service_templates/virtual-service-template.yaml",
			"test/testdata/e2e/virtual_service_templates/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)
		fixture.WaitEnvoyConfigChanged()

		By("getting initial versions")
		initialVersions := getVersionsForNode("test")
		Expect(initialVersions).NotTo(BeNil(), "Should get initial versions")
		Expect(initialVersions[listenersKey]).NotTo(BeEmpty(), "Initial listeners version should not be empty")

		By("triggering reconciliation by adding a test annotation")
		triggerReconciliation("virtualservicetemplate", "virtual-service-template")

		By("verifying versions remain stable after reconciliation")
		Consistently(func() map[string]string {
			return getVersionsForNode("test")
		}, 5*time.Second, 500*time.Millisecond).Should(SatisfyAll(
			HaveKeyWithValue(listenersKey, initialVersions[listenersKey]),
			HaveKeyWithValue(routesKey, initialVersions[routesKey]),
		), "Versions should remain unchanged after reconciliation without spec changes")
	})

	It("should increment listeners version when accessLogConfig is added", func() {
		By("applying initial manifests (VirtualService WITHOUT accessLogConfig)")
		manifests := []string{
			"test/testdata/e2e/basic_https_service/listener.yaml",
			"test/testdata/e2e/basic_https_service/tls-cert.yaml",
			"test/testdata/e2e/basic_https_service/access-log-config.yaml",
			"test/testdata/e2e/basic_https_service/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)
		fixture.WaitEnvoyConfigChanged()

		By("getting initial versions")
		initialVersions := getVersionsForNode("test")
		Expect(initialVersions).NotTo(BeNil(), "Should get initial versions")
		Expect(initialVersions[listenersKey]).NotTo(BeEmpty(), "Initial listeners version should not be empty")

		_, _ = fmt.Fprintf(GinkgoWriter, "Initial versions: listeners=%s, routes=%s, clusters=%s\n",
			initialVersions[listenersKey], initialVersions[routesKey], initialVersions[clustersKey])

		By("applying modified VirtualService WITH accessLogConfig")
		// virtual-service-v2.yaml adds accessLogConfig reference
		// This affects the HttpConnectionManager (HCM) which is part of the Listener
		fixture.ApplyManifests("test/testdata/e2e/basic_https_service/virtual-service-v2.yaml")
		fixture.WaitEnvoyConfigChanged()

		By("verifying version changes match expected behavior")
		finalVersions := getVersionsForNode("test")
		Expect(finalVersions).NotTo(BeNil(), "Should get final versions")

		_, _ = fmt.Fprintf(GinkgoWriter, "Final versions: listeners=%s, routes=%s, clusters=%s\n",
			finalVersions[listenersKey], finalVersions[routesKey], finalVersions[clustersKey])

		// AccessLogConfig is configured in HttpConnectionManager (HCM), which is part of the Listener.
		// Therefore, adding accessLogConfig MUST change the listeners version.
		Expect(finalVersions[listenersKey]).NotTo(Equal(initialVersions[listenersKey]),
			"Listeners version MUST change when accessLogConfig is added (access logs are part of HCM in Listener)")

		// Routes are separate xDS resources (RouteConfiguration) and don't contain access log settings.
		// Therefore, routes version should NOT change.
		Expect(finalVersions[routesKey]).To(Equal(initialVersions[routesKey]),
			"Routes version should NOT change (accessLogConfig doesn't affect RouteConfiguration)")

		// Clusters are upstream definitions and have no relation to access logging.
		// Therefore, clusters version should NOT change.
		Expect(finalVersions[clustersKey]).To(Equal(initialVersions[clustersKey]),
			"Clusters version should NOT change (accessLogConfig doesn't affect Cluster definitions)")
	})

	It("should increment routes version when route response body is changed", func() {
		By("applying initial manifests")
		manifests := []string{
			"test/testdata/e2e/basic_https_service/listener.yaml",
			"test/testdata/e2e/basic_https_service/tls-cert.yaml",
			"test/testdata/e2e/basic_https_service/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)
		fixture.WaitEnvoyConfigChanged()

		By("getting initial versions")
		initialVersions := getVersionsForNode("test")
		Expect(initialVersions).NotTo(BeNil(), "Should get initial versions")
		Expect(initialVersions[routesKey]).NotTo(BeEmpty(), "Initial routes version should not be empty")

		_, _ = fmt.Fprintf(GinkgoWriter, "Initial versions: listeners=%s, routes=%s, clusters=%s\n",
			initialVersions[listenersKey], initialVersions[routesKey], initialVersions[clustersKey])

		By("applying VirtualService with modified route response body")
		// virtual-service-v3-routes.yaml changes the direct_response body from "Hello" to "Hello v3"
		// This affects the RouteConfiguration (route action is part of RouteConfiguration)
		fixture.ApplyManifests("test/testdata/e2e/basic_https_service/virtual-service-v3-routes.yaml")
		fixture.WaitEnvoyConfigChanged()

		By("verifying version changes match expected behavior")
		finalVersions := getVersionsForNode("test")
		Expect(finalVersions).NotTo(BeNil(), "Should get final versions")

		_, _ = fmt.Fprintf(GinkgoWriter, "Final versions: listeners=%s, routes=%s, clusters=%s\n",
			finalVersions[listenersKey], finalVersions[routesKey], finalVersions[clustersKey])

		// Route action (direct_response body) is part of RouteConfiguration in xDS.
		// Therefore, changing the response body MUST change the routes version.
		Expect(finalVersions[routesKey]).NotTo(Equal(initialVersions[routesKey]),
			"Routes version MUST change when route response body is modified")

		// Listeners contain HCM which references RouteConfiguration by name, but the Listener itself
		// doesn't change when RouteConfiguration content changes.
		// Therefore, listeners version should NOT change.
		Expect(finalVersions[listenersKey]).To(Equal(initialVersions[listenersKey]),
			"Listeners version should NOT change (route content changes don't affect Listener/HCM config)")

		// Clusters are upstream definitions and are not affected by route content changes.
		// Therefore, clusters version should NOT change.
		Expect(finalVersions[clustersKey]).To(Equal(initialVersions[clustersKey]),
			"Clusters version should NOT change (route content changes don't affect Cluster definitions)")
	})

	It("should increment clusters version when route with cluster is added", func() {
		By("applying initial manifests with direct_response route")
		manifests := []string{
			"test/testdata/e2e/basic_https_service/listener.yaml",
			"test/testdata/e2e/basic_https_service/tls-cert.yaml",
			"test/testdata/e2e/basic_https_service/cluster.yaml",
			"test/testdata/e2e/basic_https_service/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)
		fixture.WaitEnvoyConfigChanged()

		By("getting initial versions")
		initialVersions := getVersionsForNode("test")
		Expect(initialVersions).NotTo(BeNil(), "Should get initial versions")

		_, _ = fmt.Fprintf(GinkgoWriter, "Initial versions: listeners=%s, routes=%s, clusters=%s\n",
			initialVersions[listenersKey], initialVersions[routesKey], initialVersions[clustersKey])

		By("applying VirtualService that routes to cluster instead of direct_response")
		// virtual-service-v4-cluster.yaml changes route from direct_response to cluster reference
		// This affects both RouteConfiguration (route action changes) and Clusters (cluster is now used)
		fixture.ApplyManifests("test/testdata/e2e/basic_https_service/virtual-service-v4-cluster.yaml")
		fixture.WaitEnvoyConfigChanged()

		By("verifying version changes match expected behavior")
		finalVersions := getVersionsForNode("test")
		Expect(finalVersions).NotTo(BeNil(), "Should get final versions")

		_, _ = fmt.Fprintf(GinkgoWriter, "Final versions: listeners=%s, routes=%s, clusters=%s\n",
			finalVersions[listenersKey], finalVersions[routesKey], finalVersions[clustersKey])

		// Route action changed from direct_response to cluster reference.
		// This is part of RouteConfiguration, so routes version MUST change.
		Expect(finalVersions[routesKey]).NotTo(Equal(initialVersions[routesKey]),
			"Routes version MUST change when route action changes from direct_response to cluster")

		// The Cluster CR was already applied, but it wasn't referenced by any VirtualService.
		// Now that a route references it, it becomes part of the active snapshot.
		// Therefore, clusters version MUST change.
		Expect(finalVersions[clustersKey]).NotTo(Equal(initialVersions[clustersKey]),
			"Clusters version MUST change when a cluster becomes referenced by a route")

		// Listeners/HCM configuration doesn't change - only the RouteConfiguration and Clusters change.
		// Therefore, listeners version should NOT change.
		Expect(finalVersions[listenersKey]).To(Equal(initialVersions[listenersKey]),
			"Listeners version should NOT change (route target change doesn't affect Listener/HCM)")
	})
}

// getVersionsForNode fetches snapshot versions for a given nodeID from the cache-api
//
//nolint:unparam // nodeID is parameterized for flexibility in future tests
func getVersionsForNode(nodeID string) map[string]string {
	podName := fmt.Sprintf("curl-versions-%d", time.Now().UnixNano())

	defer func() {
		cmd := exec.Command("kubectl", "delete", "pod", podName,
			"--ignore-not-found=true", "--wait=false")
		_, _ = utils.Run(cmd)
	}()

	cacheAPIHost := "exc-e2e-envoy-xds-controller-cache-api.envoy-xds-controller:9999"
	address := fmt.Sprintf("http://%s/api/v1/nodeIDs/versions", cacheAPIHost)
	cmd := exec.Command("kubectl", "run", podName, "--restart=Never",
		"--image=curlimages/curl:7.78.0",
		"--", "/bin/sh", "-c", "curl -s "+address)
	_, err := utils.Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Failed to create curl pod: %v\n", err)
		return nil
	}

	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pods", podName, "-o", "jsonpath={.status.phase}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred(), "Failed to get pod status")
		g.Expect(output).To(Equal("Succeeded"), "Pod should complete successfully")
	}, 60*time.Second, 2*time.Second).Should(Succeed(),
		"Timed out waiting for versions pod to complete")

	cmd = exec.Command("kubectl", "logs", podName)
	data, err := utils.Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get pod logs: %v\n", err)
		return nil
	}

	var allVersions []nodeIDWithVersions
	if err := json.Unmarshal([]byte(data), &allVersions); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Failed to unmarshal versions: %v, data: %s\n", err, data)
		return nil
	}

	// Find the requested nodeID
	for _, v := range allVersions {
		if v.NodeID == nodeID {
			return v.Versions
		}
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "NodeID %s not found in versions response\n", nodeID)
	return nil
}

// triggerReconciliation forces a reconciliation by adding/updating a test annotation
// This is more reliable than re-applying the same manifest because:
// 1. K8s skips updates when spec is unchanged (resourceVersion stays the same)
// 2. Adding an annotation always updates resourceVersion and triggers the controller
func triggerReconciliation(kind, name string) {
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
	cmd := exec.Command("kubectl", "annotate", kind, name,
		"test.reconcile-trigger="+timestamp, "--overwrite")
	output, err := utils.Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Failed to annotate %s/%s: %v, output: %s\n", kind, name, err, output)
	}

	// Wait a moment for the controller to receive and process the event
	// The annotation change triggers reconciliation immediately
	time.Sleep(500 * time.Millisecond)
}
