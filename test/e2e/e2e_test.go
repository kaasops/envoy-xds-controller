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
	"fmt"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kaasops/envoy-xds-controller/test/utils"
)

// namespace where the project is deployed in
const namespace = "envoy-xds-controller"

const controllerPodSelector = "app.kubernetes.io/name=envoy-xds-controller"

// serviceAccountName created for the project
const serviceAccountName = "exc-e2e-envoy-xds-controller"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "exc-e2e-envoy-xds-controller-metrics"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "envoy-xds-controller-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	cmdOutFile, err := os.CreateTemp(os.TempDir(), "exc-e2e-*.txt")
	Expect(err).NotTo(HaveOccurred(), "Failed to create temp file")
	cmdWriter := utils.CmdWriter{File: cmdOutFile}
	GinkgoWriter.TeeTo(&cmdWriter)
	_, _ = fmt.Fprintf(GinkgoWriter, "command logs written to: %s\n", cmdOutFile.Name())

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
		cmd = exec.Command(
			"make", "deploy-e2e",
			fmt.Sprintf("IMG_REPO=%s", projectImage),
			fmt.Sprintf("IMAGE_TAG=%s", projectImageTag),
			fmt.Sprintf("INIT_CERT_IMG_REPO=%s", initCertImage),
		)
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
		cmd = exec.Command("make", "undeploy-e2e")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		// Use --timeout and --wait=false to prevent hanging when namespace has resources with finalizers
		// that can't be processed (controller already stopped)
		cmd = exec.Command("kubectl", "delete", "ns", namespace, "--timeout=60s", "--wait=false")
		_, _ = utils.Run(cmd)

		By("removing metrics role binding")
		cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		if !CurrentSpecReport().Failed() {
			return
		}
		diagnose := func(args ...string) {
			out, err := utils.Run(exec.Command("kubectl", args...))
			if err != nil {
				_, _ = fmt.Fprintln(GinkgoWriter, err)
				return
			}
			_, _ = fmt.Fprintln(GinkgoWriter, out)
		}
		diagnose("logs", "-n", namespace, "-l", controllerPodSelector, "--tail=-1")
		diagnose("get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
		diagnose("logs", "-n", namespace, "curl-metrics")
		diagnose("describe", "pods", "-n", namespace, "-l", controllerPodSelector)
		diagnose("get", "pods", "-n", "default", "-o", "wide")
		diagnose("describe", "pods", "-n", "default", "-l", "app.kubernetes.io/name=envoy")
	})

	SetDefaultEventuallyTimeout(1 * time.Minute)
	SetDefaultEventuallyPollingInterval(500 * time.Millisecond)

	Context("Manager", managerContext(controllerPodSelector))

	// Envoy tests split into multiple contexts for better organization
	Context("Envoy", func() {
		Context("Basic Functionality", basicEnvoyContext)
		Context("Validation", validationEnvoyContext)
		Context("TCP Proxy", tcpProxyEnvoyContext)
		Context("Kafka Broker Filter", kafkaBrokerFilterEnvoyContext)
		Context("Templates", templatesEnvoyContext)
		Context("Tracing", tracingEnvoyContext)
		Context("HTTP2 Protocol Options", http2ProtocolOptionsEnvoyContext)
		Context("Status Propagation", statusPropagationContext)
		Context("Secret Autodiscovery Fallback", secretAutodiscoveryFallbackContext)
		Context("Snapshot Version Stability", snapshotVersionStabilityContext)
		Context("Domain Uniqueness Per Listener", domainPerListenerContext)
		Context("HTTP Filter SDS Secrets", httpFilterSDSSecretsContext)
		Context("Dynamic Modules", dynamicModuleEnvoyContext)
	})

	Context("GRPC_API", grpcAPIContext)
})
