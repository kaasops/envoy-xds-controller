package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/kaasops/envoy-xds-controller/test/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func grpcAPIContext() {
	It("should ensure http filters grpc api available", func() {
		response := fetchDataViaGRPC(`{}`, "http_filter.v1.HTTPFilterStoreService.ListHTTPFilters")
		Expect(strings.TrimSpace(response)).To(Equal("{}"))

		err := utils.ApplyManifests("test/testdata/e2e/grpc/http-filter.yaml")
		Expect(err).NotTo(HaveOccurred())

		verifyConfigUpdated := func(g Gomega) {
			response = fetchDataViaGRPC(`{}`, "http_filter.v1.HTTPFilterStoreService.ListHTTPFilters")
			g.Expect(gjson.Get(response, "items.#.name").String()).To(Equal(`["http-filter"]`))
		}
		Eventually(verifyConfigUpdated).Should(Succeed())
	})

	It("should ensure listeners grpc api available", func() {
		response := fetchDataViaGRPC(`{}`, "listener.v1.ListenerStoreService.ListListeners")
		Expect(strings.TrimSpace(response)).To(Equal("{}"))

		err := utils.ApplyManifests("test/testdata/e2e/grpc/listener.yaml")
		Expect(err).NotTo(HaveOccurred())

		verifyConfigUpdated := func(g Gomega) {
			response = fetchDataViaGRPC(`{}`, "listener.v1.ListenerStoreService.ListListeners")
			g.Expect(gjson.Get(response, "items.#.name").String()).To(Equal(`["test-listener"]`))
		}
		Eventually(verifyConfigUpdated).Should(Succeed())
	})
	It("should ensure policies grpc api available", func() {
		response := fetchDataViaGRPC(`{}`, "policy.v1.PolicyStoreService.ListPolicies")
		Expect(strings.TrimSpace(response)).To(Equal("{}"))

		err := utils.ApplyManifests("test/testdata/e2e/grpc/policy.yaml")
		Expect(err).NotTo(HaveOccurred())

		verifyConfigUpdated := func(g Gomega) {
			response = fetchDataViaGRPC(`{}`, "policy.v1.PolicyStoreService.ListPolicies")
			g.Expect(gjson.Get(response, "items.#.name").String()).To(Equal(`["test-policy"]`))
		}
		Eventually(verifyConfigUpdated).Should(Succeed())
	})
	It("should ensure routes grpc api available", func() {
		response := fetchDataViaGRPC(`{}`, "route.v1.RouteStoreService.ListRoutes")
		Expect(strings.TrimSpace(response)).To(Equal("{}"))

		err := utils.ApplyManifests("test/testdata/e2e/grpc/route.yaml")
		Expect(err).NotTo(HaveOccurred())

		verifyConfigUpdated := func(g Gomega) {
			response = fetchDataViaGRPC(`{}`, "route.v1.RouteStoreService.ListRoutes")
			g.Expect(gjson.Get(response, "items.#.name").String()).To(Equal(`["test-route"]`))
		}
		Eventually(verifyConfigUpdated).Should(Succeed())
	})
	It("should ensure virtual service templates grpc api available", func() {
		response := fetchDataViaGRPC(`{}`, "virtual_service_template.v1.VirtualServiceTemplateStoreService.ListVirtualServiceTemplates")
		Expect(strings.TrimSpace(response)).To(Equal("{}"))

		err := utils.ApplyManifests("test/testdata/e2e/grpc/virtual-service-template.yaml")
		Expect(err).NotTo(HaveOccurred())

		verifyConfigUpdated := func(g Gomega) {
			response = fetchDataViaGRPC(`{}`, "virtual_service_template.v1.VirtualServiceTemplateStoreService.ListVirtualServiceTemplates")
			g.Expect(gjson.Get(response, "items.#.name").String()).To(Equal(`["test-virtual-service-template"]`))
		}
		Eventually(verifyConfigUpdated).Should(Succeed())
	})
	It("should ensure virtual services grpc api available", func() {
		response := fetchDataViaGRPC(`{"accessGroup": "general"}`, "virtual_service.v1.VirtualServiceStoreService.ListVirtualServices")
		Expect(strings.TrimSpace(response)).To(Equal("{}"))

		// Apply listener and template manifests first to ensure they exist
		err := utils.ApplyManifests("test/testdata/e2e/grpc/listener.yaml")
		Expect(err).NotTo(HaveOccurred())

		err = utils.ApplyManifests("test/testdata/e2e/grpc/virtual-service-template.yaml")
		Expect(err).NotTo(HaveOccurred())

		// Get the listener UID
		response = fetchDataViaGRPC(`{}`, "listener.v1.ListenerStoreService.ListListeners")
		listenerUID := gjson.Get(response, "items.#(name==\"test-listener\").uid").String()
		Expect(listenerUID).NotTo(BeEmpty())

		// Get the template UID
		response = fetchDataViaGRPC(`{}`, "virtual_service_template.v1.VirtualServiceTemplateStoreService.ListVirtualServiceTemplates")
		templateUID := gjson.Get(response, "items.#(name==\"test-virtual-service-template\").uid").String()
		Expect(templateUID).NotTo(BeEmpty())

		// Create virtual service via API
		createVSRequest := fmt.Sprintf(`{
			"name": "test-virtual-service",
			"node_ids": ["test"],
			"access_group": "test",
			"template_uid": "%s",
			"listener_uid": "%s",
			"virtual_host": {
				"domains": ["test.example.com"]
			}
		}`, templateUID, listenerUID)

		response = fetchDataViaGRPC(createVSRequest, "virtual_service.v1.VirtualServiceStoreService.CreateVirtualService")
		Expect(response).NotTo(BeEmpty())

		verifyConfigUpdated := func(g Gomega) {
			response = fetchDataViaGRPC(`{"accessGroup": "test"}`, "virtual_service.v1.VirtualServiceStoreService.ListVirtualServices")
			g.Expect(gjson.Get(response, "items.#.name").String()).To(Equal(`["test-virtual-service"]`))
		}
		Eventually(verifyConfigUpdated).Should(Succeed())
	})
	It("should ensure nodes grpc api available", func() {
		response := fetchDataViaGRPC(`{}`, "node.v1.NodeStoreService.ListNodes")
		Expect(gjson.Get(response, "items.#.id").String()).To(Equal(`["dev","node1","node2","node3","node4","test"]`))
	})
	It("should ensure access groups grpc api available", func() {
		response := fetchDataViaGRPC(`{}`, "access_group.v1.AccessGroupStoreService.ListAccessGroups")
		Expect(gjson.Get(response, "items.#.name").String()).To(Equal(`["dev","general","group1","group2","group3","test"]`))
	})

	It("should ensure access log configs grpc api available", func() {
		response := fetchDataViaGRPC(`{}`, "access_log_config.v1.AccessLogConfigStoreService.ListAccessLogConfigs")
		Expect(strings.TrimSpace(response)).To(Equal("{}"))

		err := utils.ApplyManifests("test/testdata/e2e/grpc/access-log-config.yaml")
		Expect(err).NotTo(HaveOccurred())

		verifyConfigUpdated := func(g Gomega) {
			response = fetchDataViaGRPC(`{}`, "access_log_config.v1.AccessLogConfigStoreService.ListAccessLogConfigs")
			g.Expect(gjson.Get(response, "items.#.name").String()).To(Equal(`["test-access-log-config"]`))
		}
		Eventually(verifyConfigUpdated).Should(Succeed())
	})

	It("should update and delete virtual service via grpc api", func() {
		// Apply listener and template manifests first to ensure they exist
		err := utils.ApplyManifests("test/testdata/e2e/grpc/listener.yaml")
		Expect(err).NotTo(HaveOccurred())

		err = utils.ApplyManifests("test/testdata/e2e/grpc/virtual-service-template.yaml")
		Expect(err).NotTo(HaveOccurred())

		// Get the listener UID
		response := fetchDataViaGRPC(`{}`, "listener.v1.ListenerStoreService.ListListeners")
		listenerUID := gjson.Get(response, "items.#(name==\"test-listener\").uid").String()
		Expect(listenerUID).NotTo(BeEmpty())

		// Get the template UID
		response = fetchDataViaGRPC(`{}`, "virtual_service_template.v1.VirtualServiceTemplateStoreService.ListVirtualServiceTemplates")
		templateUID := gjson.Get(response, "items.#(name==\"test-virtual-service-template\").uid").String()
		Expect(templateUID).NotTo(BeEmpty())

		// Create virtual service via API
		createVSRequest := fmt.Sprintf(`{
			"name": "update-delete-test-vs",
			"node_ids": ["test"],
			"access_group": "test",
			"template_uid": "%s",
			"listener_uid": "%s",
			"virtual_host": {
				"domains": ["test2.example.com"]
			}
		}`, templateUID, listenerUID)

		By("creating the virtual service")
		response = fetchDataViaGRPC(createVSRequest, "virtual_service.v1.VirtualServiceStoreService.CreateVirtualService")
		Expect(response).NotTo(BeEmpty())

		// Verify the virtual service was created
		By("verifying the virtual service was created")
		verifyVSCreated := func(g Gomega) {
			response = fetchDataViaGRPC(`{"accessGroup": "test"}`, "virtual_service.v1.VirtualServiceStoreService.ListVirtualServices")
			_, _ = GinkgoWriter.Write([]byte(response))
			g.Expect(gjson.Get(response, "items.#(name==\"update-delete-test-vs\").uid").String()).NotTo(BeEmpty())
		}
		Eventually(verifyVSCreated).Should(Succeed())

		By("verifying the virtual service was created")
		// Get the virtual service UID
		response = fetchDataViaGRPC(`{"accessGroup": "test"}`, "virtual_service.v1.VirtualServiceStoreService.ListVirtualServices")
		vsUID := gjson.Get(response, "items.#(name==\"update-delete-test-vs\").uid").String()
		Expect(vsUID).NotTo(BeEmpty())

		By("updating the virtual service")
		// Update the virtual service
		updateVSRequest := fmt.Sprintf(`{
			"uid": "%s",
			"node_ids": ["test"],
			"template_uid": "%s",
			"listener_uid": "%s",
			"virtual_host": {
				"domains": ["updated.example.com"]
			}
		}`, vsUID, templateUID, listenerUID)

		response = fetchDataViaGRPC(updateVSRequest, "virtual_service.v1.VirtualServiceStoreService.UpdateVirtualService")
		Expect(response).NotTo(BeEmpty())

		By("verifying the virtual service was updated")
		// Verify the virtual service was updated
		verifyVSUpdated := func(g Gomega) {
			response = fetchDataViaGRPC(fmt.Sprintf(`{"uid": "%s"}`, vsUID), "virtual_service.v1.VirtualServiceStoreService.GetVirtualService")
			_, _ = GinkgoWriter.Write([]byte(response))
			g.Expect(gjson.Get(response, "virtualHost.domains.0").String()).To(Equal("updated.example.com"))
		}
		Eventually(verifyVSUpdated).Should(Succeed())

		By("deleting the virtual service")
		// Delete the virtual service
		deleteVSRequest := fmt.Sprintf(`{"uid": "%s"}`, vsUID)
		response = fetchDataViaGRPC(deleteVSRequest, "virtual_service.v1.VirtualServiceStoreService.DeleteVirtualService")
		Expect(response).NotTo(BeEmpty())

		By("verifying the virtual service was deleted")
		// Verify the virtual service was deleted
		response = fetchDataViaGRPC(`{"accessGroup": "test"}`, "virtual_service.v1.VirtualServiceStoreService.ListVirtualServices")
		Expect(gjson.Get(response, "items.#.name").String()).To(Equal(`["test-virtual-service"]`))
	})
}

func fetchDataViaGRPC(params string, endpoint string) string {
	podName := "grpcurl"

	By("creating the grpcurl pod to fetch or send data")
	cmd := exec.Command("kubectl", "run", podName, "-n", namespace, "--restart=Never",
		"--image=fullstorydev/grpcurl:v1.9.3-alpine",
		"--", "-plaintext", "-d", params,
		"exc-e2e-envoy-xds-controller-grpc-api:10000",
		endpoint)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to create grpcurl pod")

	checkReady := func(g Gomega) {
		cmd := exec.Command("kubectl", "-n", namespace, "get", "pods", podName, "-o", "jsonpath={.status.phase}")
		out, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out).To(Equal("Succeeded"))
	}
	Eventually(checkReady, time.Minute).Should(Succeed())

	By("reading response")
	cmd = exec.Command("kubectl", "logs", podName, "-n", namespace)
	response, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve output from grpcurl pod")

	By("removing the grpcurl pod")
	cmd = exec.Command("kubectl", "-n", namespace, "delete", "pod", podName)
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete grpcurl pod")

	return response
}
