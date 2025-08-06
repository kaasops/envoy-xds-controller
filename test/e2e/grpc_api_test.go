package e2e

import (
	"fmt"
	"strings"
	"time"

	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

// grpcAPIContext contains tests for the gRPC API functionality
func grpcAPIContext() {
	var fixture *fixtures.GRPCAPIFixture

	BeforeEach(func() {
		fixture = fixtures.NewGRPCAPIFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	Context("HTTP Filters API", func() {
		It("should ensure HTTP filters gRPC API is available", func() {
			By("verifying initial empty state")
			response := fixture.FetchDataViaGRPC(`{}`, "http_filter.v1.HTTPFilterStoreService.ListHTTPFilters")
			Expect(strings.TrimSpace(response)).To(Equal("{}"))

			By("applying HTTP filter manifest")
			fixture.ApplyManifests("test/testdata/e2e/grpc/http-filter.yaml")

			By("verifying HTTP filter is available via API")
			Eventually(func() bool {
				response := fixture.FetchDataViaGRPC(`{}`, "http_filter.v1.HTTPFilterStoreService.ListHTTPFilters")
				resources := gjson.Get(response, "items").Array()

				for _, resource := range resources {
					if resource.Get("name").String() == "http-filter" {
						return true
					}
				}

				return false
			}, 2*time.Minute, time.Second).Should(BeTrue(), "HTTP filter not found in response")
		})
	})

	Context("Listeners API", func() {
		It("should ensure listeners gRPC API is available", func() {
			By("verifying initial empty state")
			response := fixture.FetchDataViaGRPC(`{}`, "listener.v1.ListenerStoreService.ListListeners")
			Expect(strings.TrimSpace(response)).To(Equal("{}"))

			By("applying listener manifest")
			fixture.ApplyManifests("test/testdata/e2e/grpc/listener.yaml")

			By("verifying listener is available via API")
			Eventually(func() bool {
				response := fixture.FetchDataViaGRPC(`{}`, "listener.v1.ListenerStoreService.ListListeners")
				resources := gjson.Get(response, "items").Array()

				for _, resource := range resources {
					if resource.Get("name").String() == "test-listener" {
						return true
					}
				}

				return false
			}, 2*time.Minute, time.Second).Should(BeTrue(), "Listener not found in response")
		})
	})

	Context("Policies API", func() {
		It("should ensure policies gRPC API is available", func() {
			By("verifying initial empty state")
			response := fixture.FetchDataViaGRPC(`{}`, "policy.v1.PolicyStoreService.ListPolicies")
			Expect(strings.TrimSpace(response)).To(Equal("{}"))

			By("applying policy manifest")
			fixture.ApplyManifests("test/testdata/e2e/grpc/policy.yaml")

			By("verifying policy is available via API")
			Eventually(func() bool {
				response := fixture.FetchDataViaGRPC(`{}`, "policy.v1.PolicyStoreService.ListPolicies")
				resources := gjson.Get(response, "items").Array()

				for _, resource := range resources {
					if resource.Get("name").String() == "test-policy" {
						return true
					}
				}

				return false
			}, 2*time.Minute, time.Second).Should(BeTrue(), "Policy not found in response")
		})
	})

	Context("Routes API", func() {
		It("should ensure routes gRPC API is available", func() {
			By("verifying initial empty state")
			response := fixture.FetchDataViaGRPC(`{}`, "route.v1.RouteStoreService.ListRoutes")
			Expect(strings.TrimSpace(response)).To(Equal("{}"))

			By("applying route manifest")
			fixture.ApplyManifests("test/testdata/e2e/grpc/route.yaml")

			By("verifying route is available via API")
			Eventually(func() bool {
				response := fixture.FetchDataViaGRPC(`{}`, "route.v1.RouteStoreService.ListRoutes")
				resources := gjson.Get(response, "items").Array()

				for _, resource := range resources {
					if resource.Get("name").String() == "test-route" {
						return true
					}
				}

				return false
			}, 2*time.Minute, time.Second).Should(BeTrue(), "Route not found in response")
		})
	})

	Context("VirtualServiceTemplates API", func() {
		It("should ensure virtual service templates gRPC API is available", func() {
			By("verifying initial empty state")
			response := fixture.FetchDataViaGRPC(`{}`, "virtual_service_template.v1.VirtualServiceTemplateStoreService.ListVirtualServiceTemplates")
			Expect(strings.TrimSpace(response)).To(Equal("{}"))

			By("applying virtual service template manifest")
			fixture.ApplyManifests("test/testdata/e2e/grpc/virtual-service-template.yaml")

			By("verifying virtual service template is available via API")
			Eventually(func() bool {
				response := fixture.FetchDataViaGRPC(`{}`, "virtual_service_template.v1.VirtualServiceTemplateStoreService.ListVirtualServiceTemplates")
				resources := gjson.Get(response, "items").Array()

				for _, resource := range resources {
					if resource.Get("name").String() == "test-virtual-service-template" {
						return true
					}
				}

				return false
			}, 2*time.Minute, time.Second).Should(BeTrue(), "Virtual service template not found in response")
		})
	})

	Context("VirtualServices API", func() {
		It("should ensure virtual services gRPC API is available", func() {
			By("verifying initial empty state")
			response := fixture.FetchDataViaGRPC(`{"accessGroup": "general"}`, "virtual_service.v1.VirtualServiceStoreService.ListVirtualServices")
			Expect(strings.TrimSpace(response)).To(Equal("{}"))

			By("applying prerequisite manifests")
			fixture.ApplyManifests(
				"test/testdata/e2e/grpc/listener.yaml",
				"test/testdata/e2e/grpc/virtual-service-template.yaml",
			)

			By("getting the listener UID")
			response = fixture.FetchDataViaGRPC(`{}`, "listener.v1.ListenerStoreService.ListListeners")
			listenerUID := gjson.Get(response, "items.#(name==\"test-listener\").uid").String()
			Expect(listenerUID).NotTo(BeEmpty())

			By("getting the template UID")
			response = fixture.FetchDataViaGRPC(`{}`, "virtual_service_template.v1.VirtualServiceTemplateStoreService.ListVirtualServiceTemplates")
			templateUID := gjson.Get(response, "items.#(name==\"test-virtual-service-template\").uid").String()
			Expect(templateUID).NotTo(BeEmpty())

			By("creating virtual service via API")
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

			response = fixture.FetchDataViaGRPC(createVSRequest, "virtual_service.v1.VirtualServiceStoreService.CreateVirtualService")
			Expect(strings.TrimSpace(response)).To(Equal("{}"))

			By("verifying virtual service is available via API")
			Eventually(func() bool {
				response := fixture.FetchDataViaGRPC(`{"accessGroup": "test"}`, "virtual_service.v1.VirtualServiceStoreService.ListVirtualServices")
				resources := gjson.Get(response, "items").Array()

				for _, resource := range resources {
					if resource.Get("name").String() == "test-virtual-service" {
						return true
					}
				}

				return false
			}, 2*time.Minute, time.Second).Should(BeTrue(), "Virtual service not found in response")
		})

		It("should ensure nodes gRPC API is available", func() {
			By("verifying nodes are available")
			response := fixture.FetchDataViaGRPC(`{}`, "node.v1.NodeStoreService.ListNodes")
			Expect(gjson.Get(response, "items.#.id").String()).To(Equal(`["dev","node1","node2","node3","node4","test"]`))
		})

		It("should ensure access groups gRPC API is available", func() {
			By("verifying access groups are available")
			response := fixture.FetchDataViaGRPC(`{}`, "access_group.v1.AccessGroupStoreService.ListAccessGroups")
			Expect(gjson.Get(response, "items.#.name").String()).To(Equal(`["dev","general","group1","group2","group3","test"]`))
		})

		It("should ensure access log configs gRPC API is available", func() {
			By("verifying initial empty state")
			response := fixture.FetchDataViaGRPC(`{}`, "access_log_config.v1.AccessLogConfigStoreService.ListAccessLogConfigs")
			Expect(strings.TrimSpace(response)).To(Equal("{}"))

			By("applying access log config manifest")
			fixture.ApplyManifests("test/testdata/e2e/grpc/access-log-config.yaml")

			By("verifying access log config is available via API")
			Eventually(func() bool {
				response := fixture.FetchDataViaGRPC(`{}`, "access_log_config.v1.AccessLogConfigStoreService.ListAccessLogConfigs")
				resources := gjson.Get(response, "items").Array()

				for _, resource := range resources {
					if resource.Get("name").String() == "test-access-log-config" {
						return true
					}
				}

				return false
			}, 2*time.Minute, time.Second).Should(BeTrue(), "Access log config not found in response")
		})

		It("should update and delete virtual service via gRPC API", func() {
			By("applying prerequisite manifests")
			fixture.ApplyManifests(
				"test/testdata/e2e/grpc/listener.yaml",
				"test/testdata/e2e/grpc/virtual-service-template.yaml",
			)

			By("getting the listener UID")
			response := fixture.FetchDataViaGRPC(`{}`, "listener.v1.ListenerStoreService.ListListeners")
			listenerUID := gjson.Get(response, "items.#(name==\"test-listener\").uid").String()
			Expect(listenerUID).NotTo(BeEmpty())

			By("getting the template UID")
			response = fixture.FetchDataViaGRPC(`{}`, "virtual_service_template.v1.VirtualServiceTemplateStoreService.ListVirtualServiceTemplates")
			templateUID := gjson.Get(response, "items.#(name==\"test-virtual-service-template\").uid").String()
			Expect(templateUID).NotTo(BeEmpty())

			By("creating the virtual service")
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

			response = fixture.FetchDataViaGRPC(createVSRequest, "virtual_service.v1.VirtualServiceStoreService.CreateVirtualService")
			Expect(response).NotTo(BeEmpty())

			By("verifying the virtual service was created")
			Eventually(func() bool {
				response := fixture.FetchDataViaGRPC(`{"accessGroup": "test"}`, "virtual_service.v1.VirtualServiceStoreService.ListVirtualServices")
				_, _ = GinkgoWriter.Write([]byte(response))
				return gjson.Get(response, "items.#(name==\"update-delete-test-vs\").uid").String() != ""
			}, 2*time.Minute, time.Second).Should(BeTrue(), "Virtual service not created")

			By("getting the virtual service UID")
			response = fixture.FetchDataViaGRPC(`{"accessGroup": "test"}`, "virtual_service.v1.VirtualServiceStoreService.ListVirtualServices")
			vsUID := gjson.Get(response, "items.#(name==\"update-delete-test-vs\").uid").String()
			Expect(vsUID).NotTo(BeEmpty())

			By("updating the virtual service")
			updateVSRequest := fmt.Sprintf(`{
				"uid": "%s",
				"node_ids": ["test"],
				"template_uid": "%s",
				"listener_uid": "%s",
				"virtual_host": {
					"domains": ["updated.example.com"]
				}
			}`, vsUID, templateUID, listenerUID)

			response = fixture.FetchDataViaGRPC(updateVSRequest, "virtual_service.v1.VirtualServiceStoreService.UpdateVirtualService")
			Expect(response).NotTo(BeEmpty())

			By("verifying the virtual service was updated")
			Eventually(func() bool {
				response := fixture.FetchDataViaGRPC(fmt.Sprintf(`{"uid": "%s"}`, vsUID), "virtual_service.v1.VirtualServiceStoreService.GetVirtualService")
				_, _ = GinkgoWriter.Write([]byte(response))
				return gjson.Get(response, "virtualHost.domains.0").String() == "updated.example.com"
			}, 2*time.Minute, time.Second).Should(BeTrue(), "Virtual service not updated")

			By("deleting the virtual service")
			deleteVSRequest := fmt.Sprintf(`{"uid": "%s"}`, vsUID)
			response = fixture.FetchDataViaGRPC(deleteVSRequest, "virtual_service.v1.VirtualServiceStoreService.DeleteVirtualService")
			Expect(response).NotTo(BeEmpty())

			By("verifying the virtual service was deleted")
			response = fixture.FetchDataViaGRPC(`{"accessGroup": "test"}`, "virtual_service.v1.VirtualServiceStoreService.ListVirtualServices")
			Expect(gjson.Get(response, "items.#.name").String()).To(Equal(`["test-virtual-service"]`))
		})
	})
}
