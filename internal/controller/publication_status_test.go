package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	api "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/updater"
)

var _ = It("updates and clears publication status when only the certificate domains change", func() {
	runCtx, stop := context.WithCancel(ctx)
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Metrics: metricsserver.Options{BindAddress: "0"}, HealthProbeBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())
	inputs, err := testStore()
	Expect(err).NotTo(HaveOccurred())
	inputs.SetListener(&api.Listener{
		ObjectMeta: metav1.ObjectMeta{Name: "https", Namespace: "default"},
		Spec: &runtime.RawExtension{Raw: []byte(`{
			"name":"https",
			"address":{"socket_address":{"address":"127.0.0.1","port_value":443}},
			"listener_filters":[{
				"name":"envoy.filters.listener.tls_inspector",
				"typed_config":{"@type":"type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector"}
			}]
		}`)},
	})
	snapshots := cache.NewSnapshotCache()
	builder := updater.NewCacheUpdater(snapshots, inputs)
	vsController := &VirtualServiceReconciler{Client: mgr.GetClient(), Updater: builder,
		CacheReadyChan: cacheReadyChan, VSReconcileChan: make(chan event.GenericEvent)}
	Expect(vsController.SetupWithManager(mgr)).To(Succeed())
	secretController := &SecretReconciler{Client: mgr.GetClient(), Updater: builder, CacheReadyChan: cacheReadyChan}
	Expect(secretController.SetupWithManager(mgr)).To(Succeed())
	done := make(chan error, 1)
	go func() { done <- mgr.Start(runCtx) }()
	DeferCleanup(func() { stop(); Eventually(done, 15*time.Second).Should(Receive(BeNil())) })

	const domain = "publication.example"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "publication-cert", Namespace: "default",
			Labels:      map[string]string{"envoy.kaasops.io/secret-type": "tls"},
			Annotations: map[string]string{api.AnnotationSecretDomains: domain}},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{"tls.crt": []byte("test certificate"), "tls.key": []byte("test key")},
	}
	Expect(k8sClient.Create(ctx, secret)).To(Succeed())
	DeferCleanup(func() { Expect(k8sClient.Delete(ctx, secret)).To(Succeed()) })
	autoDiscovery := true
	vs := &api.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name: "publication", Namespace: "default",
			Annotations: map[string]string{api.AnnotationNodeIDs: "local"},
		},
		Spec: api.VirtualServiceSpec{VirtualServiceCommonSpec: api.VirtualServiceCommonSpec{
			Listener:  &api.ResourceRef{Name: "https"},
			TlsConfig: &api.TlsConfig{AutoDiscovery: &autoDiscovery},
			HTTPFilters: []*runtime.RawExtension{{Raw: []byte(`{
				"name": "envoy.filters.http.router",
				"typed_config":{"@type":"type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"}
			}`)}},
			VirtualHost: &runtime.RawExtension{Raw: []byte(`{
				"domains":["publication.example"],
				"routes":[{"match":{"prefix":"/"},"direct_response":{"status":200}}]
			}`)},
		}},
	}
	Expect(k8sClient.Create(ctx, vs)).To(Succeed())
	DeferCleanup(func() { Expect(k8sClient.Delete(ctx, vs)).To(Succeed()) })
	Eventually(func() int {
		listeners, _ := snapshots.GetListeners("local")
		return len(listeners)
	}, 15*time.Second).Should(Equal(1))

	key := client.ObjectKeyFromObject(vs)
	status := func() api.VirtualServiceStatus {
		var result api.VirtualService
		Expect(k8sClient.Get(ctx, key, &result)).To(Succeed())
		return result.Status
	}
	Eventually(status, 8*time.Second).Should(HaveField("ObservedGeneration", vs.Generation))
	secret.Annotations[api.AnnotationSecretDomains] = "other.example"
	Expect(k8sClient.Update(ctx, secret)).To(Succeed())
	Eventually(status, 8*time.Second).Should(HaveField("Invalid", true))
	Expect(status().Message).To(ContainSubstring(domain))
	Expect(status().Message).To(ContainSubstring("local="))
	Expect(status().ObservedGeneration).To(Equal(vs.Generation))
	secret.Annotations[api.AnnotationSecretDomains] = domain
	Expect(k8sClient.Update(ctx, secret)).To(Succeed())
	Eventually(status, 8*time.Second).Should(HaveField("Invalid", false))
	Expect(status().Message).To(BeEmpty())
})
