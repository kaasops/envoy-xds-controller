package store

import (
	"testing"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Store Suite")
}

var _ = Describe("Store", func() {
	Context("Clone method", func() {
		var originalStore *Store
		var clonedStore *Store

		BeforeEach(func() {
			// Create a new store with some test data
			originalStore = New()

			// Add a listener
			listener := &v1alpha1.Listener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-listener",
					Namespace: "default",
					UID:       "listener-uid",
				},
				Spec: &runtime.RawExtension{},
			}
			err := listener.Spec.UnmarshalJSON([]byte(`{
				"name": "test-listener",
				"address": {
					"socket_address": {
						"address": "0.0.0.0",
						"port_value": 10443
					}
				}
			}`))
			Expect(err).NotTo(HaveOccurred())
			originalStore.SetListener(listener)

			// Add a virtual service
			virtualService := &v1alpha1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vs",
					Namespace: "default",
					UID:       "vs-uid",
				},
				Spec: v1alpha1.VirtualServiceSpec{},
			}
			originalStore.SetVirtualService(virtualService)

			// Add a cluster
			cluster := &v1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
					UID:       "cluster-uid",
				},
				Spec: &runtime.RawExtension{},
			}
			err = cluster.Spec.UnmarshalJSON([]byte(`{
				"name": "test-cluster",
				"connect_timeout": "1s",
				"lb_policy": "LEAST_REQUEST",
				"type": "STATIC",
				"load_assignment": {
					"cluster_name": "test-cluster",
					"endpoints": [
						{
							"lb_endpoints": [
								{
									"endpoint": {
										"address": {
											"socket_address": {
												"address": "127.0.0.1",
												"port_value": 8765
											}
										}
									}
								}
							]
						}
					]
				}
			}`))
			Expect(err).NotTo(HaveOccurred())
			originalStore.SetCluster(cluster)

			// Add a secret
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
					Annotations: map[string]string{
						v1alpha1.AnnotationSecretDomains: "example.com",
					},
				},
				Data: map[string][]byte{
					"tls.crt": []byte("test-cert"),
					"tls.key": []byte("test-key"),
				},
			}
			originalStore.SetSecret(secret)

			// Clone the store
			clonedStore = originalStore.Clone()
		})

		It("should create a deep copy of the store", func() {
			// Verify that the cloned store has the same data as the original
			Expect(clonedStore).NotTo(BeNil())
			Expect(clonedStore).NotTo(BeIdenticalTo(originalStore))

			// Check listeners
			Expect(clonedStore.MapListeners()).To(HaveLen(len(originalStore.MapListeners())))
			Expect(clonedStore.GetListener(helpers.NamespacedName{Namespace: "default", Name: "test-listener"})).NotTo(BeNil())
			Expect(clonedStore.GetListenerByUID("listener-uid")).NotTo(BeNil())

			// Check virtual services
			Expect(clonedStore.MapVirtualServices()).To(HaveLen(len(originalStore.MapVirtualServices())))
			Expect(clonedStore.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "test-vs"})).NotTo(BeNil())
			Expect(clonedStore.GetVirtualServiceByUID("vs-uid")).NotTo(BeNil())

			// Check clusters
			Expect(clonedStore.MapClusters()).To(HaveLen(len(originalStore.MapClusters())))
			Expect(clonedStore.GetCluster(helpers.NamespacedName{Namespace: "default", Name: "test-cluster"})).NotTo(BeNil())
			Expect(clonedStore.GetSpecCluster("test-cluster")).NotTo(BeNil())

			// Check secrets
			Expect(clonedStore.MapSecrets()).To(HaveLen(len(originalStore.MapSecrets())))
			Expect(clonedStore.GetSecret(helpers.NamespacedName{Namespace: "default", Name: "test-secret"})).NotTo(BeNil())
			Expect(clonedStore.MapDomainSecrets()).To(HaveKey("example.com"))
		})

		It("should ensure changes to original store don't affect the clone", func() {
			// Modify the original store
			newListener := &v1alpha1.Listener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-listener",
					Namespace: "default",
					UID:       "new-listener-uid",
				},
				Spec: &runtime.RawExtension{},
			}
			err := newListener.Spec.UnmarshalJSON([]byte(`{
				"name": "new-listener",
				"address": {
					"socket_address": {
						"address": "0.0.0.0",
						"port_value": 10443
					}
				}
			}`))
			Expect(err).NotTo(HaveOccurred())
			originalStore.SetListener(newListener)

			// Delete a virtual service from the original store
			originalStore.DeleteVirtualService(helpers.NamespacedName{Namespace: "default", Name: "test-vs"})

			// Verify that the clone is not affected
			Expect(clonedStore.MapListeners()).To(HaveLen(1))
			Expect(clonedStore.GetListener(helpers.NamespacedName{Namespace: "default", Name: "new-listener"})).To(BeNil())
			Expect(clonedStore.GetVirtualService(helpers.NamespacedName{Namespace: "default", Name: "test-vs"})).NotTo(BeNil())
		})

		It("should ensure changes to clone don't affect the original store", func() {
			// Modify the cloned store
			newCluster := &v1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-cluster",
					Namespace: "default",
					UID:       "new-cluster-uid",
				},
				Spec: &runtime.RawExtension{},
			}
			err := newCluster.Spec.UnmarshalJSON([]byte(`{
				"name": "new-cluster",
				"connect_timeout": "1s",
				"lb_policy": "LEAST_REQUEST",
				"type": "STATIC",
				"load_assignment": {
					"cluster_name": "new-cluster",
					"endpoints": [
						{
							"lb_endpoints": [
								{
									"endpoint": {
										"address": {
											"socket_address": {
												"address": "127.0.0.1",
												"port_value": 8765
											}
										}
									}
								}
							]
						}
					]
				}
			}`))
			Expect(err).NotTo(HaveOccurred())
			clonedStore.SetCluster(newCluster)

			// Delete a secret from the cloned store
			clonedStore.DeleteSecret(helpers.NamespacedName{Namespace: "default", Name: "test-secret"})

			// Verify that the original store is not affected
			Expect(originalStore.MapClusters()).To(HaveLen(1))
			Expect(originalStore.GetCluster(helpers.NamespacedName{Namespace: "default", Name: "new-cluster"})).To(BeNil())
			Expect(originalStore.GetSecret(helpers.NamespacedName{Namespace: "default", Name: "test-secret"})).NotTo(BeNil())
		})

		It("should ensure that modifying objects in one store doesn't affect the other", func() {
			// Get objects from both stores
			originalListener := originalStore.GetListener(helpers.NamespacedName{Namespace: "default", Name: "test-listener"})
			clonedListener := clonedStore.GetListener(helpers.NamespacedName{Namespace: "default", Name: "test-listener"})

			// Modify the object in the original store
			originalListener.Labels = map[string]string{"modified": "true"}
			originalStore.SetListener(originalListener)

			// Verify that the object in the cloned store is not affected
			Expect(clonedListener.Labels).To(BeNil())
			Expect(clonedStore.GetListener(helpers.NamespacedName{Namespace: "default", Name: "test-listener"}).Labels).To(BeNil())
		})
	})
})
