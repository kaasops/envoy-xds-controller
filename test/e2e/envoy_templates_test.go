package e2e

import (
	"strings"

	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// templatesEnvoyContext contains tests for VirtualServiceTemplate functionality
// Note: There is a dependency between this test and the Basic Functionality test.
// The Basic Functionality test applies the templates configuration first, and then
// applies the file access logging configuration. This is necessary because the
// file access logging configuration depends on the templates configuration being
// applied first. The EnvoyFixture.ApplyManifests method will skip manifests that
// have already been applied, so this test will not duplicate the application of
// the templates configuration if it has already been applied by the Basic Functionality test.
func templatesEnvoyContext() {
	var fixture *fixtures.EnvoyFixture

	BeforeEach(func() {
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	It("should apply and verify templates functionality", func() {
		By("applying template manifests")
		manifests := []string{
			"test/testdata/e2e/virtual_service_templates/access-log-config.yaml",
			"test/testdata/e2e/virtual_service_templates/tls-cert.yaml",
			"test/testdata/e2e/virtual_service_templates/listener.yaml",
			"test/testdata/e2e/virtual_service_templates/http-filter.yaml",
			"test/testdata/e2e/virtual_service_templates/virtual-service-template.yaml",
			"test/testdata/e2e/virtual_service_templates/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)

		By("waiting for Envoy config to change")
		fixture.WaitEnvoyConfigChanged()

		By("verifying Envoy configuration")
		// nolint: lll
		expectations := map[string]string{
			"configs.0.bootstrap.node.id":                                                                                                  "test",
			"configs.0.bootstrap.node.cluster":                                                                                             "e2e",
			"configs.0.bootstrap.admin.address.socket_address.port_value":                                                                  "19000",
			"configs.2.dynamic_listeners.0.name":                                                                                           "default/https",
			"configs.2.dynamic_listeners.0.active_state.listener.name":                                                                     "default/https",
			"configs.2.dynamic_listeners.0.active_state.listener.address.socket_address.port_value":                                        "10443",
			"configs.2.dynamic_listeners.0.active_state.listener.listener_filters.0.name":                                                  "envoy.filters.listener.tls_inspector",
			"configs.2.dynamic_listeners.0.active_state.listener.listener_filters.0.typed_config.@type":                                    "type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector",
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filter_chain_match.server_names.0":                        "exc.kaasops.io",
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.http_filters.0.name":               "envoy.filters.http.router",
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.http_filters.0.typed_config.@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.stat_prefix":                       "default/virtual-service",
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.access_log.0.typed_config.@type":   "type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog",
			"configs.4.dynamic_route_configs.0.route_config.name":                                                                          "default/virtual-service",
			"configs.4.dynamic_route_configs.0.route_config.virtual_hosts.#":                                                               "2",
			"configs.4.dynamic_route_configs.0.route_config.virtual_hosts.0.domains.#":                                                     "1",
			"configs.4.dynamic_route_configs.0.route_config.virtual_hosts.1.domains.#":                                                     "1",
			"configs.4.dynamic_route_configs.0.route_config.virtual_hosts.1.domains.0":                                                     "*",
			"configs.4.dynamic_route_configs.0.route_config.virtual_hosts.1.name":                                                          "421vh",
			"configs.4.dynamic_route_configs.0.route_config.virtual_hosts.1.routes.0.direct_response.status":                               "421",
		}
		fixture.VerifyEnvoyConfig(expectations)

		By("ensuring the envoy returns expected response")
		response := fixture.FetchDataFromEnvoy("https://exc.kaasops.io:10443/")
		Expect(strings.TrimSpace(response)).To(Equal("{\"message\":\"Hello from template\"}"))
	})

	It("should ensure that the resources in use cannot be deleted", func() {
		By("trying to delete linked virtual service template")
		err := utils.DeleteManifests("test/testdata/e2e/virtual_service_templates/virtual-service-template.yaml")
		Expect(err).To(HaveOccurred())

		By("trying to delete linked http-filter")
		err = utils.DeleteManifests("test/testdata/e2e/virtual_service_templates/http-filter.yaml")
		Expect(err).To(HaveOccurred())

		By("trying to delete linked access log config")
		err = utils.DeleteManifests("test/testdata/e2e/virtual_service_templates/access-log-config.yaml")
		Expect(err).To(HaveOccurred())
	})

	It("should apply and verify template extra fields functionality", func() {
		By("applying template extra fields manifests")
		manifests := []string{
			"test/testdata/e2e/template_extra_fields/tls-cert.yaml",
			"test/testdata/e2e/template_extra_fields/listener-https.yaml",
			"test/testdata/e2e/template_extra_fields/virtual-service-template.yaml",
			"test/testdata/e2e/template_extra_fields/virtual-service.yaml",
		}
		fixture.ApplyManifests(manifests...)

		By("waiting for Envoy config to change")
		fixture.WaitEnvoyConfigChanged()

		By("verifying Envoy configuration")
		// nolint: lll
		expectations := map[string]string{
			"configs.0.bootstrap.node.id":                                                                                "test",
			"configs.0.bootstrap.node.cluster":                                                                           "e2e",
			"configs.2.dynamic_listeners.0.name":                                                                         "default/https",
			"configs.2.dynamic_listeners.0.active_state.listener.name":                                                   "default/https",
			"configs.2.dynamic_listeners.0.active_state.listener.address.socket_address.port_value":                      "10443",
			"configs.2.dynamic_listeners.0.active_state.listener.listener_filters.0.name":                                "envoy.filters.listener.tls_inspector",
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filter_chain_match.server_names.0":      "exc.kaasops.io",
			"configs.4.dynamic_route_configs.0.route_config.name":                                                        "default/virtual-service",
			"configs.4.dynamic_route_configs.0.route_config.virtual_hosts.0.domains.0":                                   "exc.kaasops.io",
			"configs.4.dynamic_route_configs.0.route_config.virtual_hosts.0.routes.0.direct_response.body.inline_string": "{\"message\":\"Hi!\"}",
			"configs.4.dynamic_route_configs.0.route_config.virtual_hosts.0.routes.0.direct_response.status":             "200",
		}
		fixture.VerifyEnvoyConfig(expectations)

		By("ensuring the envoy returns expected response with default extraField value")
		response := fixture.FetchDataFromEnvoy("https://exc.kaasops.io:10443/")
		Expect(strings.TrimSpace(response)).To(Equal("{\"message\":\"Hi!\"}"))
	})
}
