package e2e

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"os/exec"
	"strings"
	"time"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	"github.com/tidwall/gjson"
)

func envoyContext() {

	var actualCfgDump json.RawMessage

	It("should ensure the envoy listeners config dump is empty", func() {
		cfgDump := getEnvoyConfigDump("resource=dynamic_listeners")
		str, _ := cfgDump.MarshalJSON()
		Expect(strings.TrimSpace(string(str))).To(Equal("{}"))
		actualCfgDump = getEnvoyConfigDump("")
	})

	It("should applied only valid manifests", func() {
		By("applying base manifests")

		baseManifests := []string{
			"test/testdata/base/listener-http.yaml",
			"test/testdata/base/listener-https.yaml",
			"test/testdata/base/route-static.yaml",
			"test/testdata/base/route-default.yaml",
			"test/testdata/base/httpfilter-router.yaml",
			"test/testdata/base/accesslogconfig-stdout.yaml",
		}

		for _, manifest := range baseManifests {
			err := utils.ApplyManifests(manifest)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply manifest: "+manifest)
		}

		for _, tc := range []struct {
			applyBefore     []string
			manifest        string
			expectedErrText string
			cleanup         bool
		}{
			{
				manifest:        "test/testdata/conformance/accesslogconfig-auto-generated-filename-not-bool.yaml",
				expectedErrText: v1alpha1.ErrInvalidAnnotationAutogenFilenameValue.Error(),
			},
			{
				manifest:        "test/testdata/conformance/accesslogconfig-auto-generated-filename-stdout.yaml",
				expectedErrText: v1alpha1.ErrInvalidAccessLogConfigType.Error(),
			},
			{
				manifest:        "test/testdata/conformance/accesslogconfig-empty-spec.yaml",
				expectedErrText: v1alpha1.ErrSpecNil.Error(),
			},
			{
				manifest:        "test/testdata/conformance/accesslogconfig-invalid-spec.yaml",
				expectedErrText: `unknown field "foo"`,
			},
			{
				manifest:        "test/testdata/conformance/cluster-empty-spec.yaml",
				expectedErrText: v1alpha1.ErrSpecNil.Error(),
			},
			{
				manifest:        "test/testdata/conformance/cluster-invalid-spec.yaml",
				expectedErrText: `unknown field "foo"`,
			},
			{
				manifest:        "test/testdata/conformance/httpfilter-empty-spec.yaml",
				expectedErrText: v1alpha1.ErrSpecNil.Error(),
			},
			{
				manifest:        "test/testdata/conformance/httpfilter-invalid-spec.yaml",
				expectedErrText: `unknown field "foo"`,
			},
			{
				manifest:        "test/testdata/conformance/policy-spec-empty.yaml",
				expectedErrText: v1alpha1.ErrSpecNil.Error(),
			},
			{
				manifest:        "test/testdata/conformance/policy-spec-invalid.yaml",
				expectedErrText: `unknown field "field"`,
			},
			{
				manifest:        "test/testdata/conformance/route-empty-spec.yaml",
				expectedErrText: v1alpha1.ErrSpecNil.Error(),
			},
			{
				manifest:        "test/testdata/conformance/route-invalid-spec.yaml",
				expectedErrText: `unknown field "foo"`,
			},
			{
				manifest:        "test/testdata/conformance/virtualservice-empty-nodeids.yaml",
				expectedErrText: "nodeIDs is required",
			},
			{
				manifest:        "test/testdata/conformance/virtualservice-empty-domains.yaml",
				expectedErrText: "invalid VirtualHost.Domains: value must contain at least 1 item(s)",
			},
			{
				manifest:        "test/testdata/conformance/virtualservice-empty-virtualhost.yaml",
				expectedErrText: "listener is nil",
			},
			{
				manifest:        "test/testdata/conformance/virtualservice-invalid-virtualhost.yaml",
				expectedErrText: `unknown field "foo"`,
			},
			{
				manifest:        "test/testdata/conformance/virtualservice-empty-object-virtualhost.yaml",
				expectedErrText: "invalid VirtualHost.Domains: value must contain at least 1 item(s)",
			},
			{
				applyBefore:     []string{"test/testdata/certificates/exc-kaasops-io.yaml"},
				manifest:        "test/testdata/conformance/virtualservice-secret-control-autoDiscovery.yaml",
				expectedErrText: "",
				cleanup:         true,
			},
			{
				applyBefore:     []string{"test/testdata/certificates/exc-kaasops-io.yaml"},
				manifest:        "test/testdata/conformance/virtualservice-secret-control-secretRef.yaml",
				expectedErrText: "",
				cleanup:         true,
			},
			{
				applyBefore:     []string{"test/testdata/conformance/misc/policy.yaml"},
				manifest:        "test/testdata/conformance/vsvc-rbac-collision-policies-names.yaml",
				expectedErrText: `policy 'demo-policy' already exist in RBAC`,
			},
			{
				manifest:        "test/testdata/conformance/vsvc-rbac-empty.yaml",
				expectedErrText: "rbac action is empty",
			},
			{
				manifest:        "test/testdata/conformance/vsvc-rbac-empty-action.yaml",
				expectedErrText: "rbac action is empty",
			},
			{
				manifest:        "test/testdata/conformance/vsvc-rbac-empty-permissions.yaml",
				expectedErrText: "invalid Policy.Permissions: value must contain at least 1 item(s)",
			},
			{
				manifest:        "test/testdata/conformance/vsvc-rbac-empty-policies.yaml",
				expectedErrText: "rbac policies is empty",
			},
			{
				manifest:        "test/testdata/conformance/vsvc-rbac-empty-policy.yaml",
				expectedErrText: "invalid Policy.Permissions: value must contain at least 1 item(s)",
			},
			{
				manifest:        "test/testdata/conformance/vsvc-rbac-empty-principals.yaml",
				expectedErrText: "invalid Policy.Principals: value must contain at least 1 item(s)",
			},
			{
				manifest:        "test/testdata/conformance/vsvc-rbac-unknown-additional-policy.yaml",
				expectedErrText: "rbac policy default/test not found",
			},
			{
				manifest:        "test/testdata/conformance/vsvc-template-not-found.yaml",
				expectedErrText: "virtual service template default/unknown-template-name not found",
			},
			{
				manifest:        "test/testdata/conformance/accesslogconfig-auto-generated-filename.yaml",
				expectedErrText: "",
				cleanup:         true,
			},
			{
				manifest:        "test/testdata/conformance/tcp-proxy/vs-virtual-host.yaml",
				expectedErrText: "conflict: virtual host is set, but filter chains are found in listener",
				applyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				manifest:        "test/testdata/conformance/tcp-proxy/vs-additional-routes.yaml",
				expectedErrText: "conflict: additional routes are set, but filter chains are found in listener",
				applyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				manifest:        "test/testdata/conformance/tcp-proxy/vs-http-filters.yaml",
				expectedErrText: "conflict: http filters are set, but filter chains are found in listener",
				applyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				manifest:        "test/testdata/conformance/tcp-proxy/vs-additional-http-filters.yaml",
				expectedErrText: "conflict: additional http filters are set, but filter chains are found in listener",
				applyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				manifest:        "test/testdata/conformance/tcp-proxy/vs-additional-http-filters.yaml",
				expectedErrText: "conflict: additional http filters are set, but filter chains are found in listener",
				applyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				manifest:        "test/testdata/conformance/tcp-proxy/vs-tls-config.yaml",
				expectedErrText: "conflict: tls config is set, but filter chains are found in listener",
				applyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				manifest:        "test/testdata/conformance/tcp-proxy/vs-rbac.yaml",
				expectedErrText: "conflict: rbac is set, but filter chains are found in listener",
				applyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				manifest:        "test/testdata/conformance/tcp-proxy/vs-use-remote-address.yaml",
				expectedErrText: "conflict: use remote address is set, but filter chains are found in listener",
				applyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				manifest:        "test/testdata/conformance/tcp-proxy/vs-upgrade-configs.yaml",
				expectedErrText: "conflict: upgrade configs is set, but filter chains are found in listener",
				applyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				manifest:        "test/testdata/conformance/tcp-proxy/vs-access-log.yaml",
				expectedErrText: "conflict: access log is set, but filter chains are found in listener",
				applyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				manifest:        "test/testdata/conformance/tcp-proxy/vs-access-log-config.yaml",
				expectedErrText: "conflict: access log config is set, but filter chains are found in listener",
				applyBefore: []string{
					"test/testdata/conformance/tcp-proxy/cluster.yaml",
					"test/testdata/conformance/tcp-proxy/listener.yaml",
				},
			},
			{
				manifest:        "test/testdata/conformance/virtual-service-https-without-tls.yaml",
				expectedErrText: "tls listener not configured, virtual service has not tls config",
			},
			{
				manifest:        "test/testdata/conformance/virtual-service-http-with-tls.yaml",
				expectedErrText: "listener is not tls, virtual service has tls config",
			},
		} {
			if len(tc.applyBefore) > 0 {
				for _, f := range tc.applyBefore {
					err := utils.ApplyManifests(f)
					Expect(err).NotTo(HaveOccurred())
				}
			}
			err := utils.ApplyManifests(tc.manifest)
			if tc.expectedErrText == "" {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(tc.expectedErrText))
			}
			if tc.cleanup {
				err := utils.DeleteManifests(tc.manifest)
				Expect(err).NotTo(HaveOccurred())
			}
			if len(tc.applyBefore) > 0 {
				for _, f := range tc.applyBefore {
					err := utils.DeleteManifests(f)
					Expect(err).NotTo(HaveOccurred())
				}
			}
		}

		By("cleanup base manifests")
		for _, manifest := range baseManifests {
			err := utils.DeleteManifests(manifest)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply manifest: "+manifest)
		}
	})

	It("should applied virtual service manifests", func() {
		By("apply manifests")

		manifests := []string{
			"test/testdata/e2e/vs1/listener.yaml",
			"test/testdata/e2e/vs1/tls-cert.yaml",
			"test/testdata/e2e/vs1/virtual-service.yaml",
		}

		for _, manifest := range manifests {
			err := utils.ApplyManifests(manifest)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply manifest: "+manifest)
		}
	})

	It("should applied configs to envoy", func() {
		waitEnvoyConfigChanged(&actualCfgDump)
		dump := string(actualCfgDump)

		verifyConfigUpdated := func(g Gomega) {
			// nolint: lll
			for path, value := range map[string]string{
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
			} {
				Expect(value).To(Equal(gjson.Get(dump, path).String()))
			}
		}
		Eventually(verifyConfigUpdated).Should(Succeed())
	})

	It("should ensure the envoy return expected response", func() {
		response := fetchDataFromEnvoy("https://exc.kaasops.io:10443/")
		Expect(strings.TrimSpace(response)).To(Equal("{\"message\":\"Hello\"}"))
	})

	It("should ensure the envoy config changed (templates)", func() {
		By("apply manifests")

		manifests := []string{
			"test/testdata/e2e/vs2/access-log-config.yaml",
			"test/testdata/e2e/vs2/http-filter.yaml",
			"test/testdata/e2e/vs2/virtual-service-template.yaml",
			"test/testdata/e2e/vs2/virtual-service.yaml",
		}

		for _, manifest := range manifests {
			err := utils.ApplyManifests(manifest)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply manifest: "+manifest)
		}
	})

	It("should applied configs to envoy", func() {
		waitEnvoyConfigChanged(&actualCfgDump)

		verifyConfigUpdated := func(g Gomega) {
			dump := string(actualCfgDump)
			// nolint: lll
			for path, value := range map[string]string{
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
			} {
				Expect(value).To(Equal(gjson.Get(dump, path).String()), fmt.Sprintf("path: %s, value: %s", path, value))
			}
		}
		Eventually(verifyConfigUpdated).Should(Succeed())
	})

	It("should ensure the envoy return expected response", func() {
		response := fetchDataFromEnvoy("https://exc.kaasops.io:10443/")
		Expect(strings.TrimSpace(response)).To(Equal("{\"message\":\"Hello from template\"}"))
	})

	It("should ensure that the resources in use cannot be deleted", func() {
		By("try to delete linked secret")
		err := utils.DeleteManifests("test/testdata/e2e/vs1/tls-cert.yaml")
		Expect(err).To(HaveOccurred())

		By("try to delete linked virtual service template")
		err = utils.DeleteManifests("test/testdata/e2e/vs2/virtual-service-template.yaml")
		Expect(err).To(HaveOccurred())

		By("try to delete linked listener")
		err = utils.DeleteManifests("test/testdata/e2e/vs1/listener.yaml")
		Expect(err).To(HaveOccurred())

		By("try to delete linked http-filter")
		err = utils.DeleteManifests("test/testdata/e2e/vs2/http-filter.yaml")
		Expect(err).To(HaveOccurred())

		By("try to delete linked access log config")
		err = utils.DeleteManifests("test/testdata/e2e/vs2/access-log-config.yaml")
		Expect(err).To(HaveOccurred())
	})

	It("should apply access log config manifest", func() {
		err := utils.ApplyManifests("test/testdata/e2e/vs3/access-log-config.yaml")
		Expect(err).NotTo(HaveOccurred(), "Failed to apply manifest")
	})

	It("should applied access log config to envoy", func() {
		waitEnvoyConfigChanged(&actualCfgDump)
		dump := string(actualCfgDump)

		verifyConfigUpdated := func(g Gomega) {
			// nolint: lll
			for path, value := range map[string]string{
				"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.access_log.0.typed_config.@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
				"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.access_log.0.typed_config.path":  "/tmp/virtual-service.log",
			} {
				Expect(value).To(Equal(gjson.Get(dump, path).String()))
			}
		}
		Eventually(verifyConfigUpdated).Should(Succeed())
	})

	It("should ensure the envoy config changed (proxy)", func() {
		By("apply manifests")

		manifests := []string{
			"test/testdata/e2e/vs4/tcp-echo-server.yaml",
			"test/testdata/e2e/vs4/cluster.yaml",
			"test/testdata/e2e/vs4/listener.yaml",
			"test/testdata/e2e/vs4/virtual-service.yaml",
		}

		for _, manifest := range manifests {
			err := utils.ApplyManifests(manifest)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply manifest: "+manifest)
		}

		By("wait for tcp echo server ready")
		checkReady := func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pods",
				"-l", "app=tcp-echo", "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
			out, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(out).To(Equal("true"))
		}
		Eventually(checkReady, time.Minute).Should(Succeed())
	})

	It("should applied configs to envoy", func() {
		waitEnvoyConfigChanged(&actualCfgDump)

		verifyConfigUpdated := func(g Gomega) {
			dump := string(actualCfgDump)
			// nolint: lll
			for path, value := range map[string]string{
				"configs.0.bootstrap.node.id":                                 "test",
				"configs.0.bootstrap.node.cluster":                            "e2e",
				"configs.0.bootstrap.admin.address.socket_address.port_value": "19000",
				"configs.2.dynamic_listeners.0.name":                          "default/tcp-proxy",
			} {
				Expect(value).To(Equal(gjson.Get(dump, path).String()))
			}
		}
		Eventually(verifyConfigUpdated).Should(Succeed())
	})

	It("should ensure the proxy return expected response", func() {
		podName := "dummy"
		cmd := exec.Command("kubectl", "run", "--restart=Never", podName,
			"--image=busybox", "--", "sh", "-c", "echo world | nc tcp-echo 9001")
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		checkReady := func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pods", podName, "-o", "jsonpath={.status.phase}")
			out, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(out).To(Equal("Succeeded"))
		}
		Eventually(checkReady, time.Minute).Should(Succeed())

		out, err := utils.Run(exec.Command("kubectl", "logs", podName))
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("hello world"))

		_, err = utils.Run(exec.Command("kubectl", "delete", "pod", podName))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should apply http config", func() {
		By("apply manifests")
		for _, manifest := range []string{
			"test/testdata/e2e/vs5/http-listener.yaml",
			"test/testdata/e2e/vs5/vs.yaml",
		} {
			err := utils.ApplyManifests(manifest)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply manifest: "+manifest)
		}

		By("wait for envoy config changed")
		waitEnvoyConfigChanged(&actualCfgDump)

		By("ensure the envoy config changed")
		verifyConfigUpdated := func(g Gomega) {
			dump := string(actualCfgDump)
			for path, value := range map[string]string{
				"configs.2.dynamic_listeners.1.name":                                                    "default/http",
				"configs.2.dynamic_listeners.1.active_state.listener.address.socket_address.port_value": "8080",
			} {
				Expect(value).To(Equal(gjson.Get(dump, path).String()))
			}
		}
		Eventually(verifyConfigUpdated).Should(Succeed())

		By("ensure the envoy return expected response")
		response := fetchDataFromEnvoy("http://test.kaasops.io:8080/")
		Expect(strings.TrimSpace(response)).To(Equal(`{"message":"test"}`))
	})

	It("should cleanup resources", func() {
		for _, manifest := range []string{
			"test/testdata/e2e/vs5/vs.yaml",
			"test/testdata/e2e/vs5/http-listener.yaml",
			"test/testdata/e2e/vs4/virtual-service.yaml",
			"test/testdata/e2e/vs4/tcp-echo-server.yaml",
			"test/testdata/e2e/vs4/listener.yaml",
			"test/testdata/e2e/vs4/cluster.yaml",
			"test/testdata/e2e/vs2/virtual-service-template.yaml",
			"test/testdata/e2e/vs2/access-log-config.yaml",
			"test/testdata/e2e/vs2/http-filter.yaml",
			"test/testdata/e2e/vs1/listener.yaml",
			"test/testdata/e2e/vs1/tls-cert.yaml",
		} {
			err := utils.DeleteManifests(manifest)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete manifest: "+manifest)
		}
	})
}
