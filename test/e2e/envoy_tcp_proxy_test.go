package e2e

import (
	"os/exec"
	"time"

	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// tcpProxyEnvoyContext contains tests for TCP proxy functionality
func tcpProxyEnvoyContext() {
	var fixture *fixtures.EnvoyFixture

	BeforeEach(func() {
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	It("should configure and verify TCP proxy functionality", func() {
		By("applying TCP echo server and proxy manifests")
		manifests := []string{
			"test/testdata/e2e/tcp_proxy/tcp-echo-server.yaml",
			"test/testdata/e2e/tcp_proxy/cluster.yaml",
			"test/testdata/e2e/tcp_proxy/listener.yaml",
			"test/testdata/e2e/tcp_proxy/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)

		By("waiting for TCP echo server to be ready")
		Eventually(func() string {
			cmd := exec.Command("kubectl", "get", "pods",
				"-l", "app=tcp-echo", "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
			out, err := utils.Run(cmd)
			if err != nil {
				return ""
			}
			return out
		}, time.Minute).Should(Equal("true"))

		By("waiting for Envoy config to change")
		fixture.WaitEnvoyConfigChanged()

		By("verifying Envoy configuration")
		expectations := map[string]string{
			"configs.0.bootstrap.node.id":                                 "test",
			"configs.0.bootstrap.node.cluster":                            "e2e",
			"configs.0.bootstrap.admin.address.socket_address.port_value": "19000",
			"configs.2.dynamic_listeners.0.name":                          "default/tcp-proxy",
		}
		fixture.VerifyEnvoyConfig(expectations)

		By("testing the TCP proxy functionality")
		podName := "tcp-test-client"
		defer func() {
			cmd := exec.Command("kubectl", "delete", "pod", podName, "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		}()

		// Create a pod to test the TCP proxy
		cmd := exec.Command("kubectl", "run", "--restart=Never", podName,
			"--image=busybox", "--", "sh", "-c", "echo world | nc tcp-echo 9001")
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create TCP test client pod")

		// Wait for the pod to complete
		Eventually(func() string {
			cmd := exec.Command("kubectl", "get", "pods", podName, "-o", "jsonpath={.status.phase}")
			out, err := utils.Run(cmd)
			if err != nil {
				return ""
			}
			return out
		}, time.Minute).Should(Equal("Succeeded"))

		// Check the output
		cmd = exec.Command("kubectl", "logs", podName)
		out, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get logs from TCP test client pod")
		Expect(out).To(ContainSubstring("hello world"), "TCP proxy did not return expected response")
	})
}
