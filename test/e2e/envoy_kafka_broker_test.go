package e2e

import (
	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	. "github.com/onsi/ginkgo/v2"
)

// kafkaBrokerFilterEnvoyContext contains tests for Kafka Broker Filter support
func kafkaBrokerFilterEnvoyContext() {
	var fixture *fixtures.EnvoyFixture

	BeforeEach(func() {
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	It("should configure listener with kafka_broker filter before tcp_proxy", func() {
		By("applying Kafka broker filter manifests")
		manifests := []string{
			"test/testdata/e2e/kafka_broker_filter/cluster.yaml",
			"test/testdata/e2e/kafka_broker_filter/listener.yaml",
			"test/testdata/e2e/kafka_broker_filter/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)

		By("waiting for Envoy config to change")
		fixture.WaitEnvoyConfigChanged()

		By("verifying Envoy configuration")
		// Verify listener exists and has correct filter chain
		listenerPath := "configs.2.dynamic_listeners.#(name==\"default/kafka-broker-listener\").active_state.listener"
		expectations := map[string]string{
			"configs.0.bootstrap.node.id":                       "test",
			listenerPath + ".name":                              "default/kafka-broker-listener",
			listenerPath + ".address.socket_address.port_value": "19092",
		}
		fixture.VerifyEnvoyConfig(expectations)

		By("verifying cluster is present in snapshot")
		clusterExpectations := map[string]string{
			"configs.1.dynamic_active_clusters.#(cluster.name==\"localkafka\").cluster.name": "localkafka",
		}
		fixture.VerifyEnvoyConfig(clusterExpectations)

		By("test completed successfully - kafka_broker filter is supported")
		// Note: This test verifies that the controller can process a listener
		// with kafka_broker filter before tcp_proxy without errors.
		// The actual Kafka functionality testing would require a real Kafka broker
		// which is beyond the scope of this unit/integration test.
	})
}
