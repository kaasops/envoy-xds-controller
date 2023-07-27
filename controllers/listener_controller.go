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
	"sync"

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

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/go-logr/logr"

	// "github.com/go-logr/logr"
	v1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"github.com/kaasops/envoy-xds-controller/pkg/tls"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
	"github.com/kaasops/envoy-xds-controller/pkg/xds/filterchain"
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
			log.Info("Listener instance not found. Delete object fron xDS cache")
			for _, nodeID := range NodeIDs(instance, r.Cache) {
				if err := r.Cache.Delete(nodeID, &listenerv3.Listener{}, getResourceName(req.Namespace, req.Name)); err != nil {
					return ctrl.Result{}, err
				}
			}
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
	chains, err := r.buildFilterChain(ctx, log, builder, virtualServices.Items)
	if err != nil {
		return ctrl.Result{}, err
	}

	listener.FilterChains = append(listener.FilterChains, chains...)

	for _, nodeID := range NodeIDs(instance, r.Cache) {
		if len(listener.FilterChains) == 0 {
			if err := r.Cache.Delete(nodeID, &listenerv3.Listener{}, getResourceName(req.Namespace, req.Name)); err != nil {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, nil
		}

		if err := r.Cache.Update(nodeID, listener, getResourceName(req.Namespace, req.Name)); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ListenerReconciler) buildFilterChain(ctx context.Context, log logr.Logger, b filterchain.Builder, virtualServices []v1alpha1.VirtualService) ([]*listenerv3.FilterChain, error) {
	var chains []*listenerv3.FilterChain
	for _, vs := range virtualServices {
		log.WithValues("Virtual Service", vs.Name).Info("Generate Filter Chains for Virtual Service")

		// Get envoy virtualhost from virtualSerive spec
		virtualHost := &routev3.VirtualHost{}
		if err := r.Unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, virtualHost); err != nil {
			return nil, err
		}

		// Get envoy AccessLog from virtualService spec
		var accessLog *accesslogv3.AccessLog = nil
		if vs.Spec.AccessLog != nil {
			accessLog = &accesslogv3.AccessLog{}
			if err := r.Unmarshaler.Unmarshal(vs.Spec.AccessLog.Raw, accessLog); err != nil {
				return nil, err
			}
		}

		if vs.Spec.TlsConfig == nil {
			f, err := b.WithHttpConnectionManager(virtualHost, accessLog).
				WithFilterChainMatch(virtualHost).
				Build(vs.Name)
			if err != nil {
				return nil, err
			}
			chains = append(chains, f)
			continue
		} else {
			certsProvider := tls.New(r.Client, r.DiscoveryClient, vs.Spec.TlsConfig, virtualHost, r.Config, vs.Namespace)
			certs, err := certsProvider.Provide(ctx, log)
			if err != nil {
				return nil, err
			}

			var wg sync.WaitGroup
			for certName, domains := range certs {
				wg.Add(1)

				go func(log logr.Logger,
					domains []string,
					certName string,
					virtualHost *routev3.VirtualHost,
					vs v1alpha1.VirtualService,
				) {
					defer wg.Done()
					virtualHost.Domains = domains
					f, err := b.WithDownstreamTlsContext(certName).
						WithFilterChainMatch(virtualHost).
						WithHttpConnectionManager(virtualHost, accessLog).
						Build(vs.Name)
					if err != nil {
						log.WithValues("Certificate Name", certName).Error(err, "Can't create Filter Chain")
					}
					chains = append(chains, f)
				}(log, domains, certName, virtualHost, vs)

				wg.Wait()
			}
		}
	}
	return chains, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ListenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Listener{}).
		WatchesRawSource(&source.Channel{Source: listenerReconciliationChannel}, &handler.EnqueueRequestForObject{}).
		Owns(&v1alpha1.VirtualService{}).
		Complete(r)
}
