/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/tidwall/gjson"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kaasops/envoy-xds-controller/test/utils"
)

// namespace where the project is deployed in
const namespace = "envoy-xds-controller"

// serviceAccountName created for the project
const serviceAccountName = "envoy-xds-controller-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "envoy-xds-controller-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "envoy-xds-controller-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// installing CRDs, and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)

		By("removing metrics role binding")
		cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=envoy-xds-controller-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("validating that the ServiceMonitor for Prometheus is applied in the namespace")
			cmd = exec.Command("kubectl", "get", "ServiceMonitor", "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "ServiceMonitor should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady, time.Minute*2).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Serving metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:7.78.0",
				"--", "/bin/sh", "-c", fmt.Sprintf(
					"curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics",
					token, metricsServiceName, namespace))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			metricsOutput := getMetricsOutput()
			Expect(metricsOutput).To(ContainSubstring(
				"controller_runtime_webhook_requests_total",
			))
		})

		It("should have CA injection for validating webhooks", func() {
			By("checking CA injection for validating webhooks")
			verifyCAInjection := func(g Gomega) {
				cmd := exec.Command("kubectl", "get",
					"validatingwebhookconfigurations.admissionregistration.k8s.io",
					"envoy-xds-controller-validating-webhook-configuration",
					"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
				vwhOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(vwhOutput)).To(BeNumerically(">", 10))
			}
			Eventually(verifyCAInjection).Should(Succeed())
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks

		// TODO: Customize the e2e test suite with scenarios specific to your project.
		// Consider applying sample/CR(s) and check their status and/or verifying
		// the reconciliation by using the metrics, i.e.:
		// metricsOutput := getMetricsOutput()
		// Expect(metricsOutput).To(ContainSubstring(
		//    fmt.Sprintf(`controller_runtime_reconcile_total{controller="%s",result="success"} 1`,
		//    strings.ToLower(<Kind>),
		// ))
	})

	Context("Envoy", func() {

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
					expectedErrText: "invalid VirtualHost.Name: value length must be at least 1 runes",
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
			response := fetchDataFromEnvoy()
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
				} {
					Expect(value).To(Equal(gjson.Get(dump, path).String()))
				}
			}
			Eventually(verifyConfigUpdated).Should(Succeed())
		})

		It("should ensure the envoy return expected response", func() {
			response := fetchDataFromEnvoy()
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

	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return metricsOutput
}

func getEnvoyConfigDump(queryParams string) json.RawMessage {
	podName := "curl-config-dump"
	defer func() {
		cmd := exec.Command("kubectl", "delete", "pod", podName)
		_, _ = utils.Run(cmd)
	}()

	url := "http://envoy.default.svc.cluster.local:19000/config_dump"
	if queryParams != "" {
		url += "?" + queryParams
	}
	cmd := exec.Command("kubectl", "run", podName, "--restart=Never",
		"--image=curlimages/curl:7.78.0",
		"--", "/bin/sh", "-c", "curl -s "+url)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to create curl-config-dump pod")

	Eventually(func() string {
		cmd := exec.Command("kubectl", "get", "pods", podName, "-o", "jsonpath={.status.phase}")
		output, err := utils.Run(cmd)
		if err != nil {
			return ""
		}
		return output
	}, 30*time.Second).Should(Equal("Succeeded"))

	cmd = exec.Command("kubectl", "logs", podName)
	data, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve output from curl pod")
	dump := json.RawMessage{}
	err = dump.UnmarshalJSON([]byte(data))
	Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal config dump")
	return dump
}
func fetchDataFromEnvoy() string {
	podName := "curl-fetch-data"

	By("resolve ip address of the envoy pod")
	// nolint: lll
	cmd := exec.Command("kubectl", "get", "pods", "-l", "app.kubernetes.io/name=envoy", "-o", "jsonpath='{.items[0].status.podIP}'")
	output, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to resolve ip address of the envoy")
	envoyIP := strings.Trim(strings.TrimSpace(output), "'")

	By("creating the curl-config-dump pod to access config dump")
	cmd = exec.Command("kubectl", "run", podName, "--restart=Never",
		"--image=curlimages/curl:7.78.0",
		"--", "/bin/sh", "-c", "curl -s -k https://exc.kaasops.io:10443/"+" --resolve exc.kaasops.io:10443:"+envoyIP)
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to create curl-fetch-data pod")

	By("waiting for the curl-fetch-data pod to complete.")
	verifyCurlUp := func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pods", podName,
			"-o", "jsonpath={.status.phase}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
	}
	Eventually(verifyCurlUp, 30*time.Second).Should(Succeed())

	By("getting the curl-fetch-data")
	cmd = exec.Command("kubectl", "logs", podName)
	response, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve output from curl pod")

	By("cleaning up the curl pod for getting envoy config dump")
	cmd = exec.Command("kubectl", "delete", "pod", podName)
	_, _ = utils.Run(cmd)
	return response
}

func waitEnvoyConfigChanged(actualCfgDump *json.RawMessage) {
	By("waiting envoy config changed")
	Eventually(func() bool {
		cfgDump := getEnvoyConfigDump("")
		if compareJSON(cfgDump, *actualCfgDump) {
			return false
		}
		_ = os.WriteFile("/tmp/prev-dump.json", *actualCfgDump, 0644)
		*actualCfgDump = cfgDump
		_ = os.WriteFile("/tmp/actual-dump.json", *actualCfgDump, 0644)
		return true
	}, time.Minute).Should(BeTrue())
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}

func compareJSON(raw1, raw2 json.RawMessage) bool {
	var obj1, obj2 interface{}

	if err := json.Unmarshal(raw1, &obj1); err != nil {
		return false
	}
	if err := json.Unmarshal(raw2, &obj2); err != nil {
		return false
	}

	norm1, _ := json.Marshal(obj1)
	norm2, _ := json.Marshal(obj2)

	return bytes.Equal(norm1, norm2)
}
