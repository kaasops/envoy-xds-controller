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
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/go-logr/logr"

	// "github.com/go-logr/logr"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	v1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/controllers/factory/virtualservice"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"github.com/kaasops/envoy-xds-controller/pkg/tls"
	"github.com/kaasops/envoy-xds-controller/pkg/util/k8s"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
	"github.com/kaasops/envoy-xds-controller/pkg/xds/filterchain"
)

// ListenerReconciler reconciles a Listener object
type ListenerReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Cache           xdscache.Cache
	Unmarshaler     protojson.UnmarshalOptions
	DiscoveryClient *discovery.DiscoveryClient
	Config          *config.Config

	log logr.Logger
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=listeners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=listeners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=listeners/finalizers,verbs=update

func (r *ListenerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.FromContext(ctx).WithValues("Envoy Listener", req.NamespacedName)
	r.log.Info("Reconciling listener")

	// Get listener instance
	instance := &v1alpha1.Listener{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if api_errors.IsNotFound(err) {
			r.log.Info("Listener instance not found. Delete object fron xDS cache")
			for _, nodeID := range k8s.NodeIDs(instance) {
				if err := r.Cache.Delete(nodeID, resourcev3.ListenerType, getResourceName(req.Namespace, req.Name)); err != nil {
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
		client.MatchingFields{options.VirtualServiceListenerFeild: req.Name},
	}

	if err = r.List(ctx, virtualServices, listOpts...); err != nil {
		return ctrl.Result{}, err
	}

	chains, rtConfigs, err := r.configComponents(ctx, virtualServices.Items, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	listener.FilterChains = append(listener.FilterChains, chains...)
	listener.Name = getResourceName(req.Namespace, req.Name)

	// Add routeConfigs to xds cache
	for _, rtConfig := range rtConfigs {
		for _, nodeID := range k8s.NodeIDs(instance) {
			r.log.Info("Adding route", "name:", rtConfig.Name)
			if err := r.Cache.Update(nodeID, rtConfig); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Add listener to xds cache
	for _, nodeID := range k8s.NodeIDs(instance) {
		if len(listener.FilterChains) == 0 {
			r.log.WithValues("NodeID", nodeID).Info("Listener FilterChain is empty, deleting")
			if err := r.Cache.Delete(nodeID, resourcev3.ListenerType, getResourceName(req.Namespace, req.Name)); err != nil {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, nil
		}

		if err := r.Cache.Update(nodeID, listener); err != nil {
			return ctrl.Result{}, err
		}
	}

	r.log.Info("Listener reconcilation finished")

	return ctrl.Result{}, nil
}

func (r *ListenerReconciler) configComponents(ctx context.Context, virtualServices []v1alpha1.VirtualService, listener *v1alpha1.Listener) ([]*listenerv3.FilterChain, []*routev3.RouteConfiguration, error) {
	var chains []*listenerv3.FilterChain
	var routeConfigs []*routev3.RouteConfiguration
	certsProvider := tls.New(r.Client, r.DiscoveryClient, r.Config, listener.Namespace)
	index, err := certsProvider.IndexCertificateSecrets(ctx)

	if err != nil {
		return nil, nil, err
	}

	for _, vs := range virtualServices {

		factory := virtualservice.NewVirtualServiceFactory(r.Client, r.Unmarshaler, &vs, listener)

		virtSvc, err := factory.Create(ctx, getResourceName(vs.Namespace, vs.Name))

		if err != nil {
			r.log.Error(err, "Skip")
			continue
		}

		// If VirtualService nodeIDs is not empty and listener does not contains all of them - skip. TODO: Add to status
		if !k8s.NodeIDsContains(virtSvc.NodeIDs, k8s.NodeIDs(listener)) {
			r.log.Info("NodeID mismatch", "VirtualService", vs.Name)
			continue
		}

		// Get envoy virtualhost from virtualSerive spec
		virtualHost := virtSvc.VirtualHost

		routeConfigs = append(routeConfigs, virtSvc.RouteConfig)

		b := filterchain.NewBuilder()

		// Build filterchain without tls
		if vs.Spec.TlsConfig == nil {
			r.log.Info("Generate Filter Chains for Virtual Service", "name:", vs.Name)
			f, err := b.WithHttpConnectionManager(
				virtSvc.AccessLog,
				virtSvc.HttpFilters,
				getResourceName(vs.Namespace, vs.Name),
			).
				WithFilterChainMatch(virtualHost.Domains).
				Build(vs.Name)
			if err != nil {
				return nil, nil, err
			}
			chains = append(chains, f)
			continue
		}

		// Validate tls config
		errorList, err := certsProvider.Validate(ctx, index, virtualHost, vs.Spec.TlsConfig)
		if err != nil {
			return nil, nil, err
		}

		if len(errorList) > 0 {
			vs.Status.Errors = errorList
			r.Client.Status().Update(ctx, vs.DeepCopy())
		}

		// Get certs
		certs, err := certsProvider.Provide(ctx, index, virtualHost, vs.Spec.TlsConfig)
		if err != nil {
			return nil, nil, err
		}

		if len(certs) == 0 {
			r.log.Info("Failed to get secrets for VirtualService", "VirtualService", vs.Name)
			continue
		}

		// Build filterchain with tls
		r.log.Info("Generate Filter Chains for Virtual Service", "name:", vs.Name)

		for certName, domains := range certs {
			virtualHost.Domains = domains
			f, err := b.WithDownstreamTlsContext(certName).
				WithFilterChainMatch(domains).
				WithHttpConnectionManager(virtSvc.AccessLog,
					virtSvc.HttpFilters,
					getResourceName(vs.Namespace, vs.Name),
				).
				Build(fmt.Sprintf("%s-%s", vs.Name, certName))
			if err != nil {
				r.log.WithValues("Certificate Name", certName).Error(err, "Can't create Filter Chain")
			}
			chains = append(chains, f)
		}
	}
	return chains, routeConfigs, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ListenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add listener name to index
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.VirtualService{}, options.VirtualServiceListenerFeild, func(rawObject client.Object) []string {
		virtualService := rawObject.(*v1alpha1.VirtualService)
		// if listener feild is empty use default listener name as index
		if virtualService.Spec.Listener == nil {
			return []string{options.DefaultListenerName}
		}
		return []string{virtualService.Spec.Listener.Name}
	}); err != nil {
		return err
	}

	// EnqueueRequestsFromMapFunc
	// List all VirtualServies and finds listener ref
	listenerRequestMapper := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		var virtualServiceList v1alpha1.VirtualServiceList
		var reconcileRequests []reconcile.Request
		uniq := make(map[v1alpha1.ResourceRef]struct{})
		if err := mgr.GetCache().List(ctx, &virtualServiceList); err != nil {
			r.log.Error(err, "failed to list VirtualService resources")
			return nil
		}
		for _, vs := range virtualServiceList.Items {

			if refContains(virtualServiceResourceRefMapper(obj, vs), obj) {
				name := vs.Spec.Listener.Name
				namespace := obj.GetNamespace()
				resourceRef := v1alpha1.ResourceRef{Name: name}
				_, ok := uniq[resourceRef]
				if ok {
					continue
				}
				reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				}})
				uniq[resourceRef] = struct{}{}
			}
		}
		return reconcileRequests
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Listener{}).
		Watches(&v1alpha1.VirtualService{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			v := o.(*v1alpha1.VirtualService)
			reconcileRequest := []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Name:      v.GetListener(),
					Namespace: v.GetNamespace(),
				}},
			}
			checkResult, err := checkHash(v)
			if err != nil {
				r.log.Error(err, "failed to get virtualService hash")
				return reconcileRequest
			}

			if checkResult {
				r.log.V(1).Info("VirtualService has no changes. Skip Reconcile")
				return nil
			}

			r.log.V(1).Info("Updating last applied hash")
			if err := setLastAppliedHash(ctx, r.Client, v); err != nil {
				r.log.Error(err, "Failed to update last applied hash")
				return reconcileRequest
			}

			return reconcileRequest
		})).
		Watches(&v1alpha1.AccessLogConfig{}, listenerRequestMapper).
		Watches(&v1alpha1.Route{}, listenerRequestMapper).
		Complete(r)
}
