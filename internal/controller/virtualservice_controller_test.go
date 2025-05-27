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

var _ = Describe("VirtualService Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}

		virtualHost := &runtime.RawExtension{}
		err := virtualHost.UnmarshalJSON([]byte(`{
  "name": "test",
  "domains": [
    "*"
  ],
  "routes": [
    {
      "match": {
        "prefix": "/common"
      },
      "direct_response": {
        "status": 200,
        "body": {
          "inline_string": "{\"answer\":\"common\"}"
        }
      }
    }
  ]
}`))
		Expect(err).NotTo(HaveOccurred())

		virtualservice := &envoyv1alpha1.VirtualService{}
		virtualservice.Annotations = map[string]string{
			envoyv1alpha1.AnnotationNodeIDs: "test",
		}
		virtualservice.Name = resourceName
		virtualservice.Namespace = "default"
		virtualservice.Spec = envoyv1alpha1.VirtualServiceSpec{
			VirtualServiceCommonSpec: envoyv1alpha1.VirtualServiceCommonSpec{
				VirtualHost: virtualHost,
				Listener: &envoyv1alpha1.ResourceRef{
					Name: "http",
				},
			},
		}

		BeforeEach(func() {
			By("creating the custom resource for the Kind VirtualService")
			err := k8sClient.Get(ctx, typeNamespacedName, virtualservice)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, virtualservice)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &envoyv1alpha1.VirtualService{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance VirtualService")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &VirtualServiceReconciler{
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
