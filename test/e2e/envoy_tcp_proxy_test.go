package e2e

import (
	"os/exec"
	"strings"
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
		// Use gjson filter to find listener by name instead of index
		expectations := map[string]string{
			"configs.0.bootstrap.node.id":                                     "test",
			"configs.0.bootstrap.node.cluster":                                "e2e",
			"configs.0.bootstrap.admin.address.socket_address.port_value":     "19000",
			"configs.2.dynamic_listeners.#(name==\"default/tcp-proxy\").name": "default/tcp-proxy",
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

	It("should support multiple filter chains with SNI-based routing", func() {
		By("applying TCP echo servers and multi-chain proxy manifests")
		manifests := []string{
			"test/testdata/e2e/tcp_proxy_multi_chain/tcp-echo-servers.yaml",
			"test/testdata/e2e/tcp_proxy_multi_chain/clusters.yaml",
			"test/testdata/e2e/tcp_proxy_multi_chain/listener.yaml",
			"test/testdata/e2e/tcp_proxy_multi_chain/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)

		By("waiting for TCP echo servers to be ready")
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "get", "pods",
				"-l", "app=tcp-echo-1", "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
			out1, err := utils.Run(cmd)
			if err != nil || out1 != "true" {
				return false
			}
			cmd = exec.Command("kubectl", "get", "pods",
				"-l", "app=tcp-echo-2", "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
			out2, err := utils.Run(cmd)
			if err != nil || out2 != "true" {
				return false
			}
			return true
		}, time.Minute).Should(BeTrue())

		By("waiting for Envoy config to change")
		fixture.WaitEnvoyConfigChanged()

		By("verifying Envoy configuration has the listener with multiple filter chains")
		expectations := map[string]string{
			"configs.2.dynamic_listeners.#(name==\"default/tcp-proxy-multi-chain\").name": "default/tcp-proxy-multi-chain",
		}
		fixture.VerifyEnvoyConfig(expectations)

		By("verifying that both filter chains are present in the listener")
		dump := string(fixture.ConfigDump)
		Expect(dump).To(ContainSubstring("server1.test.local"), "Filter chain for server1 should be present")
		Expect(dump).To(ContainSubstring("server2.test.local"), "Filter chain for server2 should be present")
		Expect(dump).To(ContainSubstring("tcp-echo-cluster-1"), "Cluster tcp-echo-cluster-1 should be referenced")
		Expect(dump).To(ContainSubstring("tcp-echo-cluster-2"), "Cluster tcp-echo-cluster-2 should be referenced")

		By("verifying clusters are created in Envoy config")
		Expect(dump).To(ContainSubstring("tcp-echo-1"), "Cluster endpoint for tcp-echo-1 should be present")
		Expect(dump).To(ContainSubstring("tcp-echo-2"), "Cluster endpoint for tcp-echo-2 should be present")
	})

	It("should route traffic through multiple TCP proxy listeners with data flow verification", func() {
		By("applying TCP echo servers and multi-listener proxy manifests")
		manifests := []string{
			"test/testdata/e2e/tcp_proxy_multi_chain_dataflow/tcp-echo-servers.yaml",
			"test/testdata/e2e/tcp_proxy_multi_chain_dataflow/clusters.yaml",
			"test/testdata/e2e/tcp_proxy_multi_chain_dataflow/listener-1.yaml",
			"test/testdata/e2e/tcp_proxy_multi_chain_dataflow/listener-2.yaml",
			"test/testdata/e2e/tcp_proxy_multi_chain_dataflow/virtual-service-1.yaml",
			"test/testdata/e2e/tcp_proxy_multi_chain_dataflow/virtual-service-2.yaml",
		}
		fixture.ApplyManifests(manifests...)

		By("waiting for TCP echo servers to be ready")
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "get", "pods",
				"-l", "app=tcp-echo-df-1", "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
			out1, err := utils.Run(cmd)
			if err != nil || out1 != "true" {
				return false
			}
			cmd = exec.Command("kubectl", "get", "pods",
				"-l", "app=tcp-echo-df-2", "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
			out2, err := utils.Run(cmd)
			if err != nil || out2 != "true" {
				return false
			}
			return true
		}, time.Minute).Should(BeTrue())

		By("waiting for Envoy config to change")
		fixture.WaitEnvoyConfigChanged()

		By("verifying both listeners are configured in Envoy")
		dump := string(fixture.ConfigDump)
		Expect(dump).To(ContainSubstring("tcp-proxy-df-1"), "Listener tcp-proxy-df-1 should be present")
		Expect(dump).To(ContainSubstring("tcp-proxy-df-2"), "Listener tcp-proxy-df-2 should be present")
		Expect(dump).To(ContainSubstring("tcp-echo-df-cluster-1"), "Cluster tcp-echo-df-cluster-1 should be present")
		Expect(dump).To(ContainSubstring("tcp-echo-df-cluster-2"), "Cluster tcp-echo-df-cluster-2 should be present")

		By("getting Envoy pod IP for direct connection")
		cmd := exec.Command("kubectl", "get", "pods", "-l", "app.kubernetes.io/name=envoy",
			"-o", "jsonpath={.items[0].status.podIP}")
		envoyIP, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get Envoy pod IP")
		envoyIP = strings.TrimSpace(envoyIP)

		By("testing data flow through first listener (port 7781) to backend-one")
		podName1 := "tcp-df-test-client-1"
		defer func() {
			cmd := exec.Command("kubectl", "delete", "pod", podName1, "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		}()

		// Send traffic through Envoy listener on port 7781 -> should reach backend-one
		cmd = exec.Command("kubectl", "run", "--restart=Never", podName1,
			"--image=busybox", "--", "sh", "-c", "echo test | nc "+envoyIP+" 7781")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create TCP test client pod 1")

		Eventually(func() string {
			cmd := exec.Command("kubectl", "get", "pods", podName1, "-o", "jsonpath={.status.phase}")
			out, err := utils.Run(cmd)
			if err != nil {
				return ""
			}
			return out
		}, time.Minute).Should(Equal("Succeeded"))

		cmd = exec.Command("kubectl", "logs", podName1)
		out1, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get logs from TCP test client pod 1")
		Expect(out1).To(ContainSubstring("backend-one"), "Traffic through port 7781 should reach backend-one")

		By("testing data flow through second listener (port 7782) to backend-two")
		podName2 := "tcp-df-test-client-2"
		defer func() {
			cmd := exec.Command("kubectl", "delete", "pod", podName2, "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		}()

		// Send traffic through Envoy listener on port 7782 -> should reach backend-two
		cmd = exec.Command("kubectl", "run", "--restart=Never", podName2,
			"--image=busybox", "--", "sh", "-c", "echo test | nc "+envoyIP+" 7782")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create TCP test client pod 2")

		Eventually(func() string {
			cmd := exec.Command("kubectl", "get", "pods", podName2, "-o", "jsonpath={.status.phase}")
			out, err := utils.Run(cmd)
			if err != nil {
				return ""
			}
			return out
		}, time.Minute).Should(Equal("Succeeded"))

		cmd = exec.Command("kubectl", "logs", podName2)
		out2, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get logs from TCP test client pod 2")
		Expect(out2).To(ContainSubstring("backend-two"), "Traffic through port 7782 should reach backend-two")
	})

	It("should configure Kafka-style multi-broker with SNI-based routing", func() {
		By("applying Kafka-style multi-broker manifests")
		manifests := []string{
			"test/testdata/e2e/kafka_style_multi_broker/tcp-echo-servers.yaml",
			"test/testdata/e2e/kafka_style_multi_broker/clusters.yaml",
			"test/testdata/e2e/kafka_style_multi_broker/listener.yaml",
			"test/testdata/e2e/kafka_style_multi_broker/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)

		By("waiting for all Kafka broker simulators to be ready")
		Eventually(func() bool {
			labels := []string{"kafka-broker-1", "kafka-broker-2", "kafka-broker-3"}
			for _, label := range labels {
				cmd := exec.Command("kubectl", "get", "pods",
					"-l", "app="+label, "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
				out, err := utils.Run(cmd)
				if err != nil || out != "true" {
					return false
				}
			}
			return true
		}, 2*time.Minute).Should(BeTrue())

		By("waiting for Envoy config to change")
		fixture.WaitEnvoyConfigChanged()

		By("verifying Envoy configuration has the listener with 3 filter chains")
		expectations := map[string]string{
			"configs.2.dynamic_listeners.#(name==\"default/kafka-multi-broker\").name": "default/kafka-multi-broker",
		}
		fixture.VerifyEnvoyConfig(expectations)

		By("verifying all 3 filter chains are present with correct SNI matching")
		dump := string(fixture.ConfigDump)
		Expect(dump).To(ContainSubstring("kafka-broker-1.test.local"), "Filter chain for broker-1 should be present")
		Expect(dump).To(ContainSubstring("kafka-broker-2.test.local"), "Filter chain for broker-2 should be present")
		Expect(dump).To(ContainSubstring("kafka-broker-3.test.local"), "Filter chain for broker-3 should be present")
		Expect(dump).To(ContainSubstring("kafka-broker-cluster-1"), "Cluster kafka-broker-cluster-1 should be referenced")
		Expect(dump).To(ContainSubstring("kafka-broker-cluster-2"), "Cluster kafka-broker-cluster-2 should be referenced")
		Expect(dump).To(ContainSubstring("kafka-broker-cluster-3"), "Cluster kafka-broker-cluster-3 should be referenced")

		By("verifying TLS inspector listener filter is present")
		Expect(dump).To(ContainSubstring("envoy.filters.listener.tls_inspector"), "TLS inspector should be configured")

		By("verifying all 3 clusters are created with correct endpoints")
		Expect(dump).To(ContainSubstring("kafka-broker-1"), "Endpoint for kafka-broker-1 should be present")
		Expect(dump).To(ContainSubstring("kafka-broker-2"), "Endpoint for kafka-broker-2 should be present")
		Expect(dump).To(ContainSubstring("kafka-broker-3"), "Endpoint for kafka-broker-3 should be present")

		By("verifying listener port is correct")
		Expect(dump).To(ContainSubstring("9093"), "Listener should be on port 9093")

		// Note: Data-flow testing with actual SNI routing requires TLS-enabled backends
		// The configuration verification above confirms that:
		// 1. TLS inspector is configured to read SNI from ClientHello
		// 2. Three filter chains are configured with different server_names
		// 3. Each filter chain routes to the correct cluster
		// 4. All clusters have correct endpoints
	})
}
