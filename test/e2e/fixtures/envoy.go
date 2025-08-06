package fixtures

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

// EnvoyFixture encapsulates the setup and teardown logic for Envoy tests
type EnvoyFixture struct {
	// ConfigDump stores the latest Envoy config dump
	ConfigDump json.RawMessage
	// AppliedManifests tracks manifests that have been applied
	AppliedManifests []string
}

// NewEnvoyFixture creates a new EnvoyFixture
func NewEnvoyFixture() *EnvoyFixture {
	return &EnvoyFixture{
		ConfigDump:       json.RawMessage{},
		AppliedManifests: []string{},
	}
}

// Setup initializes the Envoy test environment
func (f *EnvoyFixture) Setup() {
	// Get initial config dump to use as baseline
	f.ConfigDump = f.GetEnvoyConfigDump("")
	_ = os.WriteFile("/tmp/actual-dump.json", f.ConfigDump, 0644)
}

// Teardown cleans up resources created during tests
func (f *EnvoyFixture) Teardown() {
	// Clean up any manifests that were applied
	for i := len(f.AppliedManifests) - 1; i >= 0; i-- {
		manifest := f.AppliedManifests[i]
		err := utils.DeleteManifests(manifest)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Failed to delete manifest: %s, error: %v\n", manifest, err)
		}
	}
	f.AppliedManifests = []string{}
}

// IsManifestApplied checks if a manifest is already in the applied manifests list
func (f *EnvoyFixture) IsManifestApplied(manifest string) bool {
	for _, m := range f.AppliedManifests {
		if m == manifest {
			return true
		}
	}
	return false
}

// ApplyManifests applies the given manifests and adds them to the tracking list
func (f *EnvoyFixture) ApplyManifests(manifests ...string) {
	for _, manifest := range manifests {
		// Skip if already applied
		if f.IsManifestApplied(manifest) {
			By(fmt.Sprintf("Skipping already applied manifest: %s", manifest))
			continue
		}

		err := utils.ApplyManifests(manifest)
		Expect(err).NotTo(HaveOccurred(), "Failed to apply manifest: "+manifest)
		f.AppliedManifests = append(f.AppliedManifests, manifest)
	}
}

// ApplyManifestsWithError applies manifests and expects an error containing the given text
func (f *EnvoyFixture) ApplyManifestsWithError(expectedErrText string, manifest string) {
	err := utils.ApplyManifests(manifest)
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(expectedErrText))
}

// DeleteManifests deletes the given manifests and removes them from the tracking list
func (f *EnvoyFixture) DeleteManifests(manifests ...string) {
	for _, manifest := range manifests {
		err := utils.DeleteManifests(manifest)
		Expect(err).NotTo(HaveOccurred(), "Failed to delete manifest: "+manifest)

		// Remove from tracking list
		for i, m := range f.AppliedManifests {
			if m == manifest {
				f.AppliedManifests = append(f.AppliedManifests[:i], f.AppliedManifests[i+1:]...)
				break
			}
		}
	}
}

// GetEnvoyConfigDump retrieves the Envoy config dump with optional query parameters
func (f *EnvoyFixture) GetEnvoyConfigDump(queryParams string) json.RawMessage {
	podName := "curl-config-dump"
	defer func() {
		cmd := exec.Command("kubectl", "delete", "pod", podName, "--ignore-not-found=true")
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

// WaitEnvoyConfigChanged waits for the Envoy config to change from the current state
func (f *EnvoyFixture) WaitEnvoyConfigChanged() {
	By("waiting for Envoy config to change")
	Eventually(func() bool {
		cfgDump := f.GetEnvoyConfigDump("")
		if jsonMessagesEqual(cfgDump, f.ConfigDump) {
			return false
		}
		_ = os.WriteFile("/tmp/prev-dump.json", f.ConfigDump, 0644)
		f.ConfigDump = cfgDump
		_ = os.WriteFile("/tmp/actual-dump.json", f.ConfigDump, 0644)
		return true
	}, LongTimeout, DefaultPollingInterval).Should(BeTrue())
}

// VerifyEnvoyConfig verifies that the Envoy config contains the expected values
func (f *EnvoyFixture) VerifyEnvoyConfig(expectations map[string]string) {
	dump := string(f.ConfigDump)
	for path, value := range expectations {
		Expect(value).To(Equal(gjson.Get(dump, path).String()),
			fmt.Sprintf("path: %s, value: %s", path, value))
	}
}

// FetchDataFromEnvoy sends a request to Envoy and returns the response
func (f *EnvoyFixture) FetchDataFromEnvoy(address string) string {
	podName := "curl-fetch-data"
	defer func() {
		cmd := exec.Command("kubectl", "delete", "pod", podName, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)
	}()

	parsed, err := url.Parse(address)
	Expect(err).NotTo(HaveOccurred(), "Failed to parse URL: "+address)

	By("resolving IP address of the Envoy pod")
	cmd := exec.Command("kubectl", "get", "pods", "-l", "app.kubernetes.io/name=envoy",
		"-o", "jsonpath='{.items[0].status.podIP}'")
	output, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to resolve IP address of Envoy")
	envoyIP := strings.Trim(strings.TrimSpace(output), "'")

	hostname := parsed.Hostname()
	host := parsed.Host

	By("creating the curl pod to access Envoy")
	cmd = exec.Command("kubectl", "run", podName, "--restart=Never",
		"--image=curlimages/curl:7.78.0",
		"--", "/bin/sh", "-c", "curl -s -k "+address+" --resolve "+
			host+":"+envoyIP+" -H 'Host: "+hostname+"'")
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to create curl pod")

	By("waiting for the curl pod to complete")
	Eventually(func() string {
		cmd := exec.Command("kubectl", "get", "pods", podName,
			"-o", "jsonpath={.status.phase}")
		output, err := utils.Run(cmd)
		if err != nil {
			return ""
		}
		return output
	}, 30*time.Second).Should(Equal("Succeeded"))

	By("getting the curl pod logs")
	cmd = exec.Command("kubectl", "logs", podName)
	response, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve output from curl pod")

	return response
}

// Helper function to compare JSON objects
func jsonMessagesEqual(raw1, raw2 json.RawMessage) bool {
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
