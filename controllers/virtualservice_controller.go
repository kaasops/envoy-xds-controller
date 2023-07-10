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
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/tls"
	"github.com/kaasops/envoy-xds-controller/pkg/xds"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
)

// VirtualServiceReconciler reconciles a VirtualService object
type VirtualServiceReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Unmarshaler *protojson.UnmarshalOptions
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservices/finalizers,verbs=update

func (r *VirtualServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	_ = log.FromContext(ctx)
	log := log.FromContext(ctx).WithValues("Envoy Cluster", req.NamespacedName)

	log.Info("Start process Envoy Cluster")
	virtualServiceCR, err := r.findVirtualServiceCustomResourceInstance(ctx, req)
	if err != nil {
		log.Error(err, "Failed to get Envoy Cluster CR")
		return ctrl.Result{}, err
	}
	if virtualServiceCR == nil {
		log.Info("Envoy Cluster CR not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	}

	virtualHostSpec := &routev3.VirtualHost{}

	if err := r.Unmarshaler.Unmarshal(virtualServiceCR.Spec.VirtualHost.Raw, virtualHostSpec); err != nil {
		return ctrl.Result{}, err
	}

	if virtualServiceCR.Spec.Listener == nil {
		virtualServiceCR.Spec.Listener = &xds.DefaultListener
	}

	listenerCR := &v1alpha1.Listener{}
	err = r.Get(ctx, virtualServiceCR.Spec.Listener.NamespacedName(), listenerCR)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if virtualServiceCR.Spec.Listener != nil {
		listenerSpec := &listenerv3.Listener{}
		if err := r.Unmarshaler.Unmarshal(listenerCR.Spec.Raw, listenerSpec); err != nil {
			return ctrl.Result{}, err
		}
	}

	var keypair tls.KeyPair

	if !virtualServiceCR.Spec.TlsConfig.UseCertManager {
		certificateGetter := tls.NewSecretCertificateGetter(ctx, r.Client, *virtualServiceCR.Spec.TlsConfig.SecretRef)
		keypair, err = certificateGetter.GetKeyPair()
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	envoysecret := xds.EnvoySecret(virtualServiceCR.Name, keypair)
	filterChainBuilder := xds.NewFilterChainBuilder()
	filterChain, err := filterChainBuilder.WithFilters(*virtualHostSpec).WithTlsTransportSocket(virtualServiceCR.Name).Build()
	if err != nil {
		return ctrl.Result{}, err
	}

	fmt.Printf("=============Debug run: %v, %v", envoysecret, filterChain)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.VirtualService{}).
		Complete(r)
}

func (r *VirtualServiceReconciler) findVirtualServiceCustomResourceInstance(ctx context.Context, req ctrl.Request) (*v1alpha1.VirtualService, error) {
	cr := &v1alpha1.VirtualService{}
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return cr, nil
}
