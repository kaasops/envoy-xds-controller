package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
	fmt.Println(metricsOutput)
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return metricsOutput
}

func getEnvoyConfigDump(queryParams string) json.RawMessage {
	podName := "curl-config-dump"
	defer func() {
		cmd := exec.Command("kubectl", "delete", "pod", podName)
		_, _ = utils.Run(cmd)
	}()

	address := "http://envoy.default.svc.cluster.local:19000/config_dump"
	if queryParams != "" {
		address += "?" + queryParams
	}
	cmd := exec.Command("kubectl", "run", podName, "--restart=Never",
		"--image=curlimages/curl:7.78.0",
		"--", "/bin/sh", "-c", "curl -s "+address)
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
func fetchDataFromEnvoy(address string) string {
	podName := "curl-fetch-data"

	parsed, _ := url.Parse(address)

	By("resolve ip address of the envoy pod")
	// nolint: lll
	cmd := exec.Command("kubectl", "get", "pods", "-l", "app.kubernetes.io/name=envoy", "-o", "jsonpath='{.items[0].status.podIP}'")
	output, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to resolve ip address of the envoy")
	envoyIP := strings.Trim(strings.TrimSpace(output), "'")

	By("creating the curl-config-dump pod to access config dump")
	cmd = exec.Command("kubectl", "run", podName, "--restart=Never",
		"--image=curlimages/curl:7.78.0",
		"--", "/bin/sh", "-c", "curl -s -k "+address+" --resolve "+
			parsed.Host+":"+envoyIP+" -H 'Host: "+parsed.Hostname()+"'")
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
