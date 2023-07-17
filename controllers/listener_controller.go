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

	"google.golang.org/protobuf/encoding/protojson"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	v1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"github.com/kaasops/envoy-xds-controller/pkg/filterchain"
	"github.com/kaasops/envoy-xds-controller/pkg/tls"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
)

var listenerReconciliationChannel = make(chan event.GenericEvent)

// ListenerReconciler reconciles a Listener object
type ListenerReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Cache           xdscache.Cache
	Unmarshaler     *protojson.UnmarshalOptions
	DiscoveryClient *discovery.DiscoveryClient
	Config          config.Config
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=listeners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=listeners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=listeners/finalizers,verbs=update

func (r *ListenerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("Envoy Listener", req.NamespacedName)
	log.Info("Reconciling listener")

	// Get listener instance
	instance := &v1alpha1.Listener{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if api_errors.IsNotFound(err) {
			log.Info("Listener instance not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if instance.Spec == nil {
		return ctrl.Result{}, ErrEmptySpec
	}

	// get envoy listener from listener instance spec
	listener := &listenerv3.Listener{}
	if err := r.Unmarshaler.Unmarshal(instance.Spec.Raw, listener); err != nil {
		return ctrl.Result{}, err
	}

	// Get VirtualService objects with matching listener
	virtualServices := &v1alpha1.VirtualServiceList{}
	listOpts := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingFields{VirtualServiceListenerFeild: req.Name},
	}

	if err = r.List(ctx, virtualServices, listOpts...); err != nil {
		return ctrl.Result{}, err
	}

	builder := filterchain.NewBuilder()
	chain, err := r.buildFilterChain(ctx, builder, virtualServices.Items)
	if err != nil {
		return ctrl.Result{}, err
	}

	listener.FilterChains = chain

	if err := r.Cache.Update(NodeID(instance), listener, instance.Name, resourcev3.ListenerType); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ListenerReconciler) buildFilterChain(ctx context.Context, b filterchain.Builder, virtualServices []v1alpha1.VirtualService) ([]*listenerv3.FilterChain, error) {
	var chain []*listenerv3.FilterChain
	for _, vs := range virtualServices {
		// Get envoy virtualhost from virtualSerive spec
		virtualHost := &routev3.VirtualHost{}
		if err := r.Unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, virtualHost); err != nil {
			return nil, err
		}

		nodeIDs := []string{"default", "main"}

		if vs.Spec.TlsConfig == nil {
			f, err := b.WithHttpConnectionManager(virtualHost).Build()
			if err != nil {
				return nil, err
			}
			chain = append(chain, f)
			return chain, nil
		}

		certsProvider := tls.New(r.Client, r.DiscoveryClient, vs.Spec.TlsConfig, virtualHost, nodeIDs, r.Config, vs.Namespace)
		certs, err := certsProvider.Provide(ctx)
		if err != nil {
			return nil, err
		}

		for certName, domain := range certs {
			virtualHost.Domains = domain
			f, err := b.WithDownstreamTlsContext(certName).WithHttpConnectionManager(virtualHost).Build()
			if err != nil {
				return nil, err
			}
			chain = append(chain, f)
		}
	}
	return chain, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ListenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Listener{}).
		WatchesRawSource(&source.Channel{Source: listenerReconciliationChannel}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
