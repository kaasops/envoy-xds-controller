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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
)

var _ = Describe("HttpFilter Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: defaultNamespace,
		}
		httpfilter := &envoyv1alpha1.HttpFilter{}
		httpfilter.Name = resourceName
		httpfilter.Namespace = defaultNamespace

		spec := runtime.RawExtension{}
		err := spec.UnmarshalJSON([]byte(`{
    "name": "envoy.filters.http.router",
    "typed_config": {
      "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
    }
  }`))
		Expect(err).NotTo(HaveOccurred())
		httpfilter.Spec = []*runtime.RawExtension{&spec}

		BeforeEach(func() {
			By("creating the custom resource for the Kind HttpFilter")
			err := k8sClient.Get(ctx, typeNamespacedName, httpfilter)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, httpfilter)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &envoyv1alpha1.HttpFilter{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance HttpFilter")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &HttpFilterReconciler{
				Client:  k8sClient,
				Scheme:  k8sClient.Scheme(),
				Updater: cacheUpdater,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
