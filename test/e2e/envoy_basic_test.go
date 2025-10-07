package e2e

import (
	"strings"

	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// basicEnvoyContext contains tests for basic Envoy functionality
func basicEnvoyContext() {
	var fixture *fixtures.EnvoyFixture

	BeforeEach(func() {
		By("setting up EnvoyFixture")
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	It("should ensure the envoy listeners config dump is empty initially", func() {
		cfgDump := fixture.GetEnvoyConfigDump("resource=dynamic_listeners")
		str, _ := cfgDump.MarshalJSON()
		Expect(strings.TrimSpace(string(str))).To(Equal("{}"))
	})

	It("should apply virtual service manifests and verify configuration", func() {
		By("applying manifests")
		manifests := []string{
			"test/testdata/e2e/basic_https_service/listener.yaml",
			"test/testdata/e2e/basic_https_service/tls-cert.yaml",
			"test/testdata/e2e/basic_https_service/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)

		fixture.WaitEnvoyConfigChanged()

		By("verifying Envoy configuration")
		// Use gjson filters to find resources by name instead of index to avoid conflicts with previous tests
		// nolint: lll,goconst
		listenerPath := "configs.2.dynamic_listeners.#(name==\"default/https\").active_state.listener"

		// nolint: lll
		expectations := map[string]string{
			"configs.0.bootstrap.node.id":                                                              "test",
			"configs.0.bootstrap.node.cluster":                                                         "e2e",
			"configs.0.bootstrap.admin.address.socket_address.port_value":                              "19000",
			listenerPath + ".name":                                                                     "default/https",
			listenerPath + ".address.socket_address.port_value":                                        "443",
			listenerPath + ".listener_filters.0.name":                                                  "envoy.filters.listener.tls_inspector",
			listenerPath + ".listener_filters.0.typed_config.@type":                                    "type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector",
			listenerPath + ".filter_chains.0.filter_chain_match.server_names.0":                        "exc.kaasops.io",
			listenerPath + ".filter_chains.0.filters.0.typed_config.http_filters.0.name":               "envoy.filters.http.router",
			listenerPath + ".filter_chains.0.filters.0.typed_config.http_filters.0.typed_config.@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
			listenerPath + ".filter_chains.0.filters.0.typed_config.stat_prefix":                       "default/virtual-service",
		}
		fixture.VerifyEnvoyConfig(expectations)
	})

	It("should ensure the envoy returns expected response", func() {
		By("applying manifests")
		manifests := []string{
			"test/testdata/e2e/basic_https_service/listener.yaml",
			"test/testdata/e2e/basic_https_service/tls-cert.yaml",
			"test/testdata/e2e/basic_https_service/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)
		response := fixture.FetchDataFromEnvoy("https://exc.kaasops.io:443/")
		Expect(strings.TrimSpace(response)).To(Equal("{\"message\":\"Hello\"}"))
	})

	It("should ensure that the resources in use cannot be deleted", func() {
		By("trying to delete linked secret")
		err := utils.DeleteManifests("test/testdata/e2e/basic_https_service/tls-cert.yaml")
		Expect(err).To(HaveOccurred())

		By("trying to delete linked listener")
		err = utils.DeleteManifests("test/testdata/e2e/basic_https_service/listener.yaml")
		Expect(err).To(HaveOccurred())
	})

	It("should apply access log config manifest", func() {
		// Now apply the file access logging configuration
		By("applying file access logging configuration")
		fixture.ApplyManifests(
			"test/testdata/e2e/basic_https_service/listener.yaml",
			"test/testdata/e2e/basic_https_service/tls-cert.yaml",
			"test/testdata/e2e/basic_https_service/access-log-config.yaml",
			"test/testdata/e2e/basic_https_service/virtual-service-v2.yaml",
		)

		fixture.WaitEnvoyConfigChanged()

		By("verifying access log config in Envoy")
		// Use gjson filter to find listener by name
		// nolint: lll,goconst
		listenerPath := "configs.2.dynamic_listeners.#(name==\"default/https\").active_state.listener"

		// nolint: lll
		expectations := map[string]string{
			listenerPath + ".filter_chains.0.filters.0.typed_config.access_log.0.typed_config.@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
			listenerPath + ".filter_chains.0.filters.0.typed_config.access_log.0.typed_config.path":  "/tmp/virtual-service.log",
		}
		fixture.VerifyEnvoyConfig(expectations)
	})

	It("should apply http config", func() {
		By("applying manifests")
		fixture.ApplyManifests(
			"test/testdata/e2e/http_service/http-listener.yaml",
			"test/testdata/e2e/http_service/virtual-service.yaml",
		)

		By("waiting for Envoy config to change")
		fixture.WaitEnvoyConfigChanged()

		By("verifying Envoy configuration")
		// nolint: goconst
		// Use gjson filter to find listener by name
		listenerPath := "configs.2.dynamic_listeners.#(name==\"default/http\").active_state.listener"

		expectations := map[string]string{
			listenerPath + ".name":                              "default/http",
			listenerPath + ".address.socket_address.port_value": "8080",
		}
		fixture.VerifyEnvoyConfig(expectations)

		By("ensuring the envoy returns expected response")
		response := fixture.FetchDataFromEnvoy("http://test.kaasops.io:8080/")
		Expect(strings.TrimSpace(response)).To(Equal(`{"message":"test"}`))
	})
}
