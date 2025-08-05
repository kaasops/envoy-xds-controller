package fixtures

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

// GRPCAPIFixture encapsulates the setup and teardown logic for gRPC API tests
type GRPCAPIFixture struct {
	// PodName is the name of the pod used to make gRPC requests
	PodName string
	// AppliedManifests tracks manifests that have been applied
	AppliedManifests []string
}

// NewGRPCAPIFixture creates a new GRPCAPIFixture
func NewGRPCAPIFixture() *GRPCAPIFixture {
	return &GRPCAPIFixture{
		PodName:          "grpc-client",
		AppliedManifests: []string{},
	}
}

// Setup initializes the gRPC API test environment
func (f *GRPCAPIFixture) Setup() {
	// No need to create a pod here anymore as it will be created on-demand in FetchDataViaGRPC
}

// Teardown cleans up resources created during tests
func (f *GRPCAPIFixture) Teardown() {
	// Clean up any manifests that were applied
	for i := len(f.AppliedManifests) - 1; i >= 0; i-- {
		manifest := f.AppliedManifests[i]
		err := utils.DeleteManifests(manifest)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Failed to delete manifest: %s, error: %v\n", manifest, err)
		}
	}
	f.AppliedManifests = []string{}

	// No need to delete the pod here as it's now created and deleted on-demand in FetchDataViaGRPC
}

// ApplyManifests applies the given manifests and adds them to the tracking list
func (f *GRPCAPIFixture) ApplyManifests(manifests ...string) {
	for _, manifest := range manifests {
		err := utils.ApplyManifests(manifest)
		Expect(err).NotTo(HaveOccurred(), "Failed to apply manifest: "+manifest)
		f.AppliedManifests = append(f.AppliedManifests, manifest)
	}
}

// DeleteManifests deletes the given manifests and removes them from the tracking list
func (f *GRPCAPIFixture) DeleteManifests(manifests ...string) {
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

// FetchDataViaGRPC sends a gRPC request and returns the response
// It creates a pod on-demand, executes the request, and then deletes the pod
func (f *GRPCAPIFixture) FetchDataViaGRPC(params, method string) string {
	By(fmt.Sprintf("sending gRPC request to method %s", method))

	// Create a temporary pod for the gRPC request
	By("creating a temporary pod for gRPC API request")
	tempPodName := fmt.Sprintf("%s-%d", f.PodName, time.Now().UnixNano())
	createCmd := exec.Command("kubectl", "run", tempPodName, "-n", Namespace, "--restart=Never",
		"--image=fullstorydev/grpcurl:v1.9.3-alpine",
		"--", "-plaintext", "-d", params,
		"exc-e2e-envoy-xds-controller-resource-api:10000",
		method)
	_, err := utils.Run(createCmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to create temporary gRPC client pod")

	// Make sure to clean up the pod when we're done
	defer func() {
		deleteCmd := exec.Command("kubectl", "delete", "pod", "-n", Namespace, tempPodName, "--ignore-not-found=true")
		_, _ = utils.Run(deleteCmd)
	}()

	// Wait for the pod to be ready
	Eventually(func() string {
		cmd := exec.Command("kubectl", "get", "pods", "-n", Namespace, tempPodName, "-o", "jsonpath={.status.phase}")
		output, err := utils.Run(cmd)
		if err != nil {
			return ""
		}
		return output
	}, DefaultTimeout, DefaultPollingInterval).Should(Equal("Succeeded"))

	By("getting the gRPC pod logs")
	getLogsCmd := exec.Command("kubectl", "logs", "-n", Namespace, tempPodName)
	getLogsCmdOut, err := utils.Run(getLogsCmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve gRPC pod logs")

	return getLogsCmdOut
}

// VerifyGRPCResponse verifies that the gRPC response contains the expected values
func (f *GRPCAPIFixture) VerifyGRPCResponse(response string, expectations map[string]string) {
	for path, expectedValue := range expectations {
		actualValue := gjson.Get(response, path).String()
		Expect(actualValue).To(Equal(expectedValue),
			fmt.Sprintf("path: %s, expected: %s, actual: %s", path, expectedValue, actualValue))
	}
}

// WaitForResource waits for a resource to be available via the gRPC API
func (f *GRPCAPIFixture) WaitForResource(data, method, resourcePath, resourceName string) {
	By(fmt.Sprintf("waiting for resource %s to be available", resourceName))

	verifyResourceAvailable := func(g Gomega) {
		response := f.FetchDataViaGRPC(data, method)
		resources := gjson.Get(response, resourcePath).Array()

		found := false
		for _, resource := range resources {
			if resource.Get("name").String() == resourceName {
				found = true
				break
			}
		}

		g.Expect(found).To(BeTrue(), fmt.Sprintf("Resource %s not found in response", resourceName))
	}

	Eventually(verifyResourceAvailable, DefaultTimeout, DefaultPollingInterval).Should(Succeed())
}
