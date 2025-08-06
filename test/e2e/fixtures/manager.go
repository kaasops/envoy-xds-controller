package fixtures

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ManagerFixture uses constants defined in constants.go

// ManagerFixture encapsulates the setup and teardown logic for Manager tests
type ManagerFixture struct {
	// ControllerPodName is the name of the controller pod
	ControllerPodName string
}

// NewManagerFixture creates a new ManagerFixture
func NewManagerFixture() *ManagerFixture {
	return &ManagerFixture{}
}

// Setup initializes the Manager test environment
func (f *ManagerFixture) Setup() {
	By("validating that the controller-manager pod is running")
	f.verifyControllerUp()
}

// Teardown cleans up resources created during tests
func (f *ManagerFixture) Teardown() {
	// Clean up any resources created during tests
	By("cleaning up the curl pod for metrics")
	cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", Namespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)
}

// verifyControllerUp checks that the controller pod is running and sets the pod name
func (f *ManagerFixture) verifyControllerUp() {
	verifyControllerUp := func(g Gomega) {
		// Get the name of the controller-manager pod
		cmd := exec.Command("kubectl", "get",
			"pods", "-l", "app.kubernetes.io/name=envoy-xds-controller",
			"-o", "go-template={{ range .items }}"+
				"{{ if not .metadata.deletionTimestamp }}"+
				"{{ .metadata.name }}"+
				"{{ \"\\n\" }}{{ end }}{{ end }}",
			"-n", Namespace,
		)

		podOutput, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
		podNames := utils.GetNonEmptyLines(podOutput)
		g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
		f.ControllerPodName = podNames[0]
		g.Expect(f.ControllerPodName).To(ContainSubstring("envoy-xds-controller"))

		// Validate the pod's status
		cmd = exec.Command("kubectl", "get",
			"pods", f.ControllerPodName, "-o", "jsonpath={.status.phase}",
			"-n", Namespace,
		)
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
	}
	Eventually(verifyControllerUp, DefaultTimeout, DefaultPollingInterval).Should(Succeed())
}

// CreateMetricsRoleBinding creates a ClusterRoleBinding for the service account to access metrics
func (f *ManagerFixture) CreateMetricsRoleBinding() {
	By("creating a ClusterRoleBinding for the service account to allow access to metrics")
	cmd := exec.Command("kubectl", "create", "clusterrolebinding", MetricsRoleBindingName,
		"--clusterrole=exc-e2e-envoy-xds-controller-metrics-reader",
		fmt.Sprintf("--serviceaccount=%s:%s", Namespace, ServiceAccountName),
	)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")
}

// VerifyMetricsService checks that the metrics service is available
func (f *ManagerFixture) VerifyMetricsService() {
	By("validating that the metrics service is available")
	cmd := exec.Command("kubectl", "get", "service", MetricsServiceName, "-n", Namespace)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")
}

// VerifyServiceMonitor checks that the ServiceMonitor for Prometheus is applied
func (f *ManagerFixture) VerifyServiceMonitor() {
	By("validating that the ServiceMonitor for Prometheus is applied in the namespace")
	cmd := exec.Command("kubectl", "get", "ServiceMonitor", "-n", Namespace)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "ServiceMonitor should exist")
}

// GetServiceAccountToken returns a token for the service account
func (f *ManagerFixture) GetServiceAccountToken() string {
	By("getting the service account token")

	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", ServiceAccountName)
	tokenRequestFile := fmt.Sprintf("/tmp/%s", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), 0644)
	Expect(err).NotTo(HaveOccurred(), "Failed to write token request file")

	var token string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			Namespace,
			ServiceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var tokenReq struct {
			Status struct {
				Token string `json:"token"`
			} `json:"status"`
		}
		err = json.Unmarshal(output, &tokenReq)
		g.Expect(err).NotTo(HaveOccurred())

		token = tokenReq.Status.Token
		g.Expect(token).NotTo(BeEmpty())
	}
	Eventually(verifyTokenCreation, DefaultTimeout, DefaultPollingInterval).Should(Succeed())

	return token
}

// VerifyMetricsEndpoint checks that the metrics endpoint is ready
func (f *ManagerFixture) VerifyMetricsEndpoint() {
	By("waiting for the metrics endpoint to be ready")
	verifyMetricsEndpointReady := func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "endpoints", MetricsServiceName, "-n", Namespace)
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
	}
	Eventually(verifyMetricsEndpointReady, LongTimeout, DefaultPollingInterval).Should(Succeed())
}

// VerifyMetricsServer checks that the controller is serving metrics
func (f *ManagerFixture) VerifyMetricsServer() {
	By("verifying that the controller manager is serving the metrics server")
	verifyMetricsServerStarted := func(g Gomega) {
		cmd := exec.Command("kubectl", "logs", f.ControllerPodName, "-n", Namespace)
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(ContainSubstring("Serving metrics server"),
			"Metrics server not yet started")
	}
	Eventually(verifyMetricsServerStarted, DefaultTimeout, DefaultPollingInterval).Should(Succeed())
}

// AccessMetricsEndpoint creates a pod to access the metrics endpoint and returns the output
func (f *ManagerFixture) AccessMetricsEndpoint(token string) string {
	By("creating the curl-metrics pod to access the metrics endpoint")
	cmd := exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
		"--namespace", Namespace,
		"--image=curlimages/curl:7.78.0",
		"--", "/bin/sh", "-c", fmt.Sprintf(
			"curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics",
			token, MetricsServiceName, Namespace))
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

	By("waiting for the curl-metrics pod to complete")
	verifyCurlUp := func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
			"-o", "jsonpath={.status.phase}",
			"-n", Namespace)
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
	}
	Eventually(verifyCurlUp, LongTimeout, DefaultPollingInterval).Should(Succeed())

	By("getting the metrics by checking curl-metrics logs")
	cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", Namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))

	return metricsOutput
}

// VerifyCAInjection checks that CA injection for validating webhooks is working
func (f *ManagerFixture) VerifyCAInjection() {
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
	Eventually(verifyCAInjection, DefaultTimeout, DefaultPollingInterval).Should(Succeed())
}
