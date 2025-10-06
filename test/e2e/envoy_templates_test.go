package e2e

import (
	"fmt"
	"strings"
	"time"

	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
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

		// nolint: lll
		// Use gjson filters to find resources by name instead of index to avoid conflicts with previous tests
		listenerPath := "configs.2.dynamic_listeners.#(name==\"default/https\").active_state.listener"
		routePath := "configs.4.dynamic_route_configs.#(route_config.name==\"default/virtual-service\").route_config"

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
			listenerPath + ".filter_chains.0.filters.0.typed_config.access_log.0.typed_config.@type":   "type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog",
			routePath + ".name":                                            "default/virtual-service",
			routePath + ".virtual_hosts.#":                                 "2",
			routePath + ".virtual_hosts.0.domains.#":                       "1",
			routePath + ".virtual_hosts.1.domains.#":                       "1",
			routePath + ".virtual_hosts.1.domains.0":                       "*",
			routePath + ".virtual_hosts.1.name":                            "421vh",
			routePath + ".virtual_hosts.1.routes.0.direct_response.status": "421",
		}

		By("waiting for Envoy config to change")
		fixture.WaitEnvoyConfigChanged()

		By("verifying Envoy configuration with retries")
		// Use Eventually to allow for potential delays in configuration propagation
		Eventually(func() error {
			// Refresh config dump on each retry
			fixture.ConfigDump = fixture.GetEnvoyConfigDump("")
			dump := string(fixture.ConfigDump)

			for path, expectedValue := range expectations {
				actualValue := gjson.Get(dump, path).String()
				if actualValue != expectedValue {
					return fmt.Errorf("path %s: expected %q, got %q", path, expectedValue, actualValue)
				}
			}
			return nil
		}, 60*time.Second, 3*time.Second).Should(Succeed(), "Envoy config should match expectations")

		By("ensuring the envoy returns expected response")
		response := fixture.FetchDataFromEnvoy("https://exc.kaasops.io:443/")
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

		// nolint: lll
		// Use gjson filters to find resources by name instead of index to avoid conflicts with previous tests
		listenerPath := "configs.2.dynamic_listeners.#(name==\"default/https\").active_state.listener"
		routePath := "configs.4.dynamic_route_configs.#(route_config.name==\"default/virtual-service\").route_config"

		expectations := map[string]string{
			"configs.0.bootstrap.node.id":                                              "test",
			"configs.0.bootstrap.node.cluster":                                         "e2e",
			listenerPath + ".name":                                                     "default/https",
			listenerPath + ".address.socket_address.port_value":                        "443",
			listenerPath + ".listener_filters.0.name":                                  "envoy.filters.listener.tls_inspector",
			listenerPath + ".filter_chains.0.filter_chain_match.server_names.0":        "exc.kaasops.io",
			routePath + ".name":                                                        "default/virtual-service",
			routePath + ".virtual_hosts.0.domains.0":                                   "exc.kaasops.io",
			routePath + ".virtual_hosts.0.routes.0.direct_response.body.inline_string": "{\"message\":\"Hi!\"}",
			routePath + ".virtual_hosts.0.routes.0.direct_response.status":             "200",
		}

		By("waiting for Envoy config to change")
		fixture.WaitEnvoyConfigChanged()

		By("verifying Envoy configuration with retries")
		// Use Eventually to allow for potential delays in configuration propagation
		Eventually(func() error {
			// Refresh config dump on each retry
			fixture.ConfigDump = fixture.GetEnvoyConfigDump("")
			dump := string(fixture.ConfigDump)

			for path, expectedValue := range expectations {
				actualValue := gjson.Get(dump, path).String()
				if actualValue != expectedValue {
					return fmt.Errorf("path %s: expected %q, got %q", path, expectedValue, actualValue)
				}
			}
			return nil
		}, 60*time.Second, 3*time.Second).Should(Succeed(), "Envoy config should match expectations")

		By("ensuring the envoy returns expected response with default extraField value")
		response := fixture.FetchDataFromEnvoy("https://exc.kaasops.io:443/")
		Expect(strings.TrimSpace(response)).To(Equal("{\"message\":\"Hi!\"}"))
	})
}
