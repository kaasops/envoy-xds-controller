package e2e

import (
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	dynamicModuleSource = "test/testdata/e2e/dynamic_module/module"
	// dynamicModuleNodeDir is mounted into the Envoy pod by test/utils/envoy/deployment.yaml.
	dynamicModuleNodeDir = "/opt/envoy-modules"
	dynamicModuleFile    = "libe2e.so"
	dynamicModuleHeader  = "x-e2e-dynamic-module"
	// dynamicModuleCounter is the counter the listener filter increments for each accepted connection.
	dynamicModuleCounter = "accepted_connections"
)

var dynamicModuleCounterRe = regexp.MustCompile(`\.` + dynamicModuleCounter + `: (\d+)`)

// dynamicModuleEnvoyContext checks that a Listener and an HttpFilter load an Envoy dynamic module
// by path (dynamic_module_config.module.local.filename) without restarting Envoy.
func dynamicModuleEnvoyContext() {
	var fixture *fixtures.EnvoyFixture

	BeforeEach(func() {
		supported, err := utils.EnvoyVersionAtLeast(1, 38)
		Expect(err).NotTo(HaveOccurred())
		if !supported {
			Skip("loading a dynamic module by path needs Envoy 1.38 or newer")
		}

		By("setting up EnvoyFixture")
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	It("should load a dynamic module by path in a Listener and an HttpFilter without restarting Envoy", func() {
		By("building the test module and copying it to the kind node")
		installDynamicModule()

		podBefore := envoyPodState()

		By("applying manifests")
		fixture.ApplyManifests(
			"test/testdata/e2e/dynamic_module/listener.yaml",
			"test/testdata/e2e/dynamic_module/http-filter.yaml",
			"test/testdata/e2e/dynamic_module/virtual-service.yaml",
		)

		// Wait for this listener rather than for any config change: a deletion left over from
		// the previous context can change the config first.
		By("waiting for the module filters in the Envoy config")
		modulePath := dynamicModuleNodeDir + "/" + dynamicModuleFile
		listenerPath := `configs.2.dynamic_listeners.#(name=="default/http-dynamic-module").active_state.listener`
		listenerFilterPath := listenerPath +
			`.listener_filters.#(name=="envoy.filters.listener.dynamic_modules").typed_config`
		httpFilterPath := listenerPath + `.filter_chains.0.filters.0.typed_config` +
			`.http_filters.#(name=="envoy.filters.http.dynamic_modules").typed_config`
		fixture.WaitEnvoyConfigMatches(map[string]string{
			listenerFilterPath + ".dynamic_module_config.module.local.filename": modulePath,
			listenerFilterPath + ".filter_name":                                 "count_connections",
			httpFilterPath + ".dynamic_module_config.module.local.filename":     modulePath,
			httpFilterPath + ".filter_name":                                     "set_response_header",
		})

		By("sending a request through the module")
		Eventually(func(g Gomega) {
			status, headers, body := fixture.FetchResponseFromEnvoy("http://dynamic-module.kaasops.io:8090/")
			g.Expect(status).To(Equal(http.StatusOK), body)
			g.Expect(headers.Get(dynamicModuleHeader)).To(Equal("loaded-by-path"))
		}, time.Minute, 5*time.Second).Should(Succeed())

		By("verifying the listener filter counted the connection")
		Eventually(func(g Gomega) {
			stats := fixture.GetEnvoyAdmin("stats?filter=" + dynamicModuleCounter)
			m := dynamicModuleCounterRe.FindStringSubmatch(stats)
			g.Expect(m).NotTo(BeNil(), "counter %s not found in stats: %s", dynamicModuleCounter, stats)
			count, err := strconv.Atoi(m[1])
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(count).To(BeNumerically(">", 0))
		}, time.Minute, 5*time.Second).Should(Succeed())

		By("verifying Envoy was not restarted")
		Expect(envoyPodState()).To(Equal(podBefore))
	})
}

// installDynamicModule builds the test module for the architecture of the kind node
// and copies it to the directory the Envoy pod mounts.
func installDynamicModule() {
	projectDir, err := utils.GetProjectDir()
	Expect(err).NotTo(HaveOccurred())

	// Docker exports the build result itself, so no directory has to be shared with the Docker host.
	outDir := GinkgoT().TempDir()
	cmd := exec.Command("docker", "buildx", "build", "--output", "type=local,dest="+outDir,
		filepath.Join(projectDir, dynamicModuleSource))
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to build the dynamic module")

	node := utils.KindClusterName() + "-control-plane"
	cmd = exec.Command("docker", "exec", node, "mkdir", "-p", dynamicModuleNodeDir)
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to create the module directory on the kind node")

	cmd = exec.Command("docker", "cp", filepath.Join(outDir, dynamicModuleFile),
		node+":"+dynamicModuleNodeDir+"/"+dynamicModuleFile)
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to copy the dynamic module to the kind node")
}

// envoyPodState returns the Envoy pod name and the restart count of its container.
func envoyPodState() string {
	cmd := exec.Command("kubectl", "get", "pods", "-l", "app.kubernetes.io/name=envoy", "-o",
		"jsonpath={.items[0].metadata.name}/{.items[0].status.containerStatuses[0].restartCount}")
	output, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to get the Envoy pod")
	return strings.TrimSpace(output)
}
