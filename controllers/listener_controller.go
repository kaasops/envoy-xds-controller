/*
Copyright 2023.

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

package controllers

import (
	"context"

	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	v1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/xds"
)

var listenerReconciliationChannel = make(chan event.GenericEvent)

// ListenerReconciler reconciles a Listener object
type ListenerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Cache  cachev3.SnapshotCache
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=listeners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=listeners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=listeners/finalizers,verbs=update

func (r *ListenerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	log := log.FromContext(ctx).WithValues("Envoy Listener", req.NamespacedName)

	log.Info("Start process Envoy Listener")
	listenerCR, err := r.findListenerCustomResourceInstance(ctx, req)
	if err != nil {
		log.Error(err, "Failed to get Envoy Listener CR")
		return ctrl.Result{}, err
	}
	if listenerCR == nil {
		log.Info("Envoy Listener CR not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	}
	if listenerCR.Spec == nil {
		log.Info("Envoy Listener CR spec not found. Ignoring since object")
		return ctrl.Result{}, nil
	}

	if err := xds.Ensure(ctx, r.Cache, listenerCR); err != nil {
		return ctrl.Result{}, err
	}

	// if virtualServiceCR.Spec.Listener != nil {
	// 	listenerSpec := &listenerv3.Listener{}
	// 	if err := r.Unmarshaler.Unmarshal(listenerCR.Spec.Raw, listenerSpec); err != nil {
	// 		return ctrl.Result{}, err
	// 	}
	// }

	// virtualHostSpec := &routev3.VirtualHost{}

	// if err := r.Unmarshaler.Unmarshal(virtualServiceCR.Spec.VirtualHost.Raw, virtualHostSpec); err != nil {
	// 	return ctrl.Result{}, err
	// }

	// var keypair tls.KeyPair

	// if !virtualServiceCR.Spec.TlsConfig.UseCertManager {
	// 	certificateGetter := tls.NewSecretCertificateGetter(ctx, r.Client, *virtualServiceCR.Spec.TlsConfig.SecretRef)
	// 	keypair, err = certificateGetter.GetKeyPair()
	// 	if err != nil {
	// 		return ctrl.Result{}, err
	// 	}
	// }

	// envoysecret := xds.EnvoySecret(virtualServiceCR.Name, keypair)
	// filterChainBuilder := xds.NewFilterChainBuilder()
	// filterChain, err := filterChainBuilder.WithFilters(*virtualHostSpec).WithTlsTransportSocket(virtualServiceCR.Name).Build()
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	// fmt.Printf("=============Debug run: %v, %v", envoysecret, filterChain)

	return ctrl.Result{}, nil
}

func (r *ListenerReconciler) findListenerCustomResourceInstance(ctx context.Context, req ctrl.Request) (*v1alpha1.Listener, error) {
	cr := &v1alpha1.Listener{}
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return cr, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ListenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Listener{}).
		WatchesRawSource(&source.Channel{Source: listenerReconciliationChannel}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
