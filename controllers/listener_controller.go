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
	"slices"
	"sort"
	"strings"

	"github.com/go-logr/logr"

	"google.golang.org/protobuf/proto"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	v1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/factory/virtualservice"
	"github.com/kaasops/envoy-xds-controller/pkg/factory/virtualservice/tls"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"

	corev1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ListenerReconciler reconciles a Listener object
type ListenerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Cache  *xdscache.Cache
	Config *config.Config

	log logr.Logger
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=listeners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=listeners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=listeners/finalizers,verbs=update

func (r *ListenerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.FromContext(ctx).WithValues("Envoy Listener", req.NamespacedName)
	r.log.Info("Reconciling listener")

	var instanceMessage v1alpha1.Message

	// Get listener instance
	instance := &v1alpha1.Listener{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		// if listener not found, delete him from cache
		if api_errors.IsNotFound(err) {
			r.log.V(1).Info("Listener instance not found. Delete object from xDS cache")
			nodeIDs, err := r.Cache.GetNodeIDsForResource(resourcev3.ListenerType, getResourceName(req.Namespace, req.Name))
			if err != nil {
				return ctrl.Result{}, errors.Wrap(err, errors.GetNodeIDForResource)
			}
			for _, nodeID := range nodeIDs {
				if err := r.Cache.Delete(nodeID, resourcev3.ListenerType, getResourceName(req.Namespace, req.Name)); err != nil {
					return ctrl.Result{}, errors.Wrap(err, errors.CannotDeleteFromCacheMessage)
				}
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrap(err, errors.GetFromKubernetesMessage)
	}

	// Validate Listener
	if err := instance.Validate(ctx); err != nil {
		instanceMessage.Add(err.Error())
		if err := instance.SetError(ctx, r.Client, instanceMessage); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Get listener NodeIDs
	// TODO: If nodeid not set - use default nodeID
	nodeIDs := k8s.NodeIDs(instance)
	if len(nodeIDs) == 0 {
		return ctrl.Result{}, errors.New(errors.NodeIDsEmpty)
	}

	// Get Envoy Listener from listener instance spec
	listener := &listenerv3.Listener{}
	options.Unmarshaler.Unmarshal(instance.Spec.Raw, listener)

	// Get VirtualService objects with matching listener
	// TODO: add functcionality to merge listener and virtual service in different namespaces
	virtualServices := &v1alpha1.VirtualServiceList{}
	listOpts := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingFields{options.VirtualServiceListenerNameFeild: req.Name},
	}
	if err = r.List(ctx, virtualServices, listOpts...); err != nil {
		return ctrl.Result{}, errors.Wrap(err, errors.GetFromKubernetesMessage)
	}

	listener.Name = getResourceName(req.Namespace, req.Name)

	// TODO: if listener doesnt work with TLS - don't get certificate index

	// Create HashMap for fast searching of certificates
	index, err := k8s.IndexCertificateSecrets(ctx, r.Client, r.Config.GetWatchNamespaces())
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "cannot generate TLS certificates index from Kubernetes secrets")
	}

	// Sort for find which VirtualService was created first
	sort.Slice(virtualServices.Items, func(i, j int) bool {
		return virtualServices.Items[i].CreationTimestamp.Before(&virtualServices.Items[j].CreationTimestamp)
	})

	// Save listener FilterChains
	listenerFilterChains := listener.FilterChains

	// Collect all VirtualServices for listener in each NodeID
	for _, nodeID := range nodeIDs {
		// errs for collect all errors with processing VirtualServices
		var errs []error

		// routeConfigs for collect Routes for listener
		routeConfigs := make([]*routev3.RouteConfiguration, 0)

		// chains for collect Filter Chains for listener
		chains := make([]*listenerv3.FilterChain, 0)

		for _, vs := range virtualServices.Items {
			var vsMessage v1alpha1.Message

			if err := v1alpha1.FillFromTemplateIfNeeded(ctx, r.Client, &vs); err != nil {
				if errors.NeedStatusUpdate(err) {
					vsMessage.Add(errors.Wrap(err, "cannot get Virtual Service struct").Error())
					if err := vs.SetError(ctx, r.Client, vsMessage); err != nil {
						errs = append(errs, err)
					}
					continue
				}
				errs = append(errs, err)
				continue
			}

			// If Virtual Service has nodeID or Virtual Service don't have any nondeID (or all NodeID)
			if slices.Contains(k8s.NodeIDs(&vs), nodeID) || k8s.NodeIDs(&vs) == nil {
				// Create Factory for TLS
				tlsFactory := tls.NewTlsFactory(
					ctx,
					vs.Spec.TlsConfig,
					r.Client,
					r.Config.GetDefaultIssuer(),
					instance.Namespace,
					index,
				)

				// Create Factory for VirtualService
				vsFactory := virtualservice.NewVirtualServiceFactory(
					r.Client,
					&vs,
					instance,
					*tlsFactory,
				)

				// Create VirtualService
				virtSvc, err := vsFactory.Create(ctx, getResourceName(vs.Namespace, vs.Name))
				if err != nil {
					if errors.NeedStatusUpdate(err) {
						vsMessage.Add(errors.Wrap(err, "cannot get Virtual Service struct").Error())
						if err := vs.SetError(ctx, r.Client, vsMessage); err != nil {
							errs = append(errs, err)
						}
						continue
					}
					errs = append(errs, err)
					continue
				}

				// Collect routes
				routeConfigs = append(routeConfigs, virtSvc.RouteConfig)

				// Get and collect Filter Chains
				filterChains, err := virtualservice.FilterChains(&virtSvc)
				if err != nil {
					if errors.NeedStatusUpdate(err) {
						vsMessage.Add(errors.Wrap(err, "cannot get Filter Chains").Error())
						if err := vs.SetError(ctx, r.Client, vsMessage); err != nil {
							errs = append(errs, err)
						}
						continue
					}
					errs = append(errs, err)
					continue
				}

				chains = append(chains, filterChains...)

				// If for this vs used some secrets (with certificates), add secrets to cache
				if virtSvc.CertificatesWithDomains != nil {
					for nn := range virtSvc.CertificatesWithDomains {
						if err := r.makeEnvoySecret(ctx, nn, nodeID); err != nil {
							return ctrl.Result{}, err
						}
					}

					// Add information aboute used secrets to VirtualService
					i := 0
					keys := make([]string, len(virtSvc.CertificatesWithDomains))
					for k := range virtSvc.CertificatesWithDomains {
						keys[i] = k
						i++
					}
					if err := vs.SetValidWithUsedSecrets(ctx, r.Client, keys, vsMessage); err != nil {
						errs = append(errs, err)
					}
					continue
				}

				if err := vs.SetValid(ctx, r.Client, vsMessage); err != nil {
					errs = append(errs, err)
				}
			}
		}

		// Check errors
		if len(errs) != 0 {
			for _, e := range errs {
				r.log.Error(e, "FilterChain build errors")
			}

			// Stop working with this NodeID
			continue
		}

		// Add builded FilterChains to Listener
		listener.FilterChains = append(listenerFilterChains, chains...)
		applyListener := proto.Clone(listener).(*listenerv3.Listener)

		// Clear Listener, if don't have FilterChains
		if len(listener.FilterChains) == 0 {
			r.log.WithValues("NodeID", nodeID).Info("Listener FilterChain is empty, deleting")

			instanceMessage.Add(fmt.Sprintf("NodeID: %s, %s", nodeID, "Listener FilterChain is empty"))

			if err := r.Cache.Delete(nodeID, resourcev3.ListenerType, getResourceName(req.Namespace, req.Name)); err != nil {
				return ctrl.Result{}, errors.Wrap(err, errors.CannotDeleteFromCacheMessage)
			}
			continue
		}

		// Validate Listener
		if err := listener.ValidateAll(); err != nil {
			instanceMessage.Add(fmt.Sprintf("NodeID: %s, %s", nodeID, errors.CannotValidateCacheResourceMessage))
			if err := instance.SetError(ctx, r.Client, instanceMessage); err != nil {
				return ctrl.Result{}, err
			}
			return reconcile.Result{}, errors.WrapUKS(err, errors.CannotValidateCacheResourceMessage)
		}

		// Update listener in xDS cache
		r.log.V(1).WithValues("NodeID", nodeID).Info("Update listener", "name:", listener.Name)
		if err := r.Cache.Update(nodeID, applyListener); err != nil {
			return ctrl.Result{}, errors.Wrap(err, errors.CannotUpdateCacheMessage)
		}

		// Update routes in xDS cache
		for _, rtConfig := range routeConfigs {
			// Validate RouteConfig
			if err := rtConfig.ValidateAll(); err != nil {
				return reconcile.Result{}, errors.WrapUKS(err, errors.CannotValidateCacheResourceMessage)
			}

			r.log.V(1).WithValues("NodeID", nodeID).Info("Update route", "name:", rtConfig.Name)
			if err := r.Cache.Update(nodeID, rtConfig); err != nil {
				return ctrl.Result{}, errors.Wrap(err, errors.CannotUpdateCacheMessage)
			}
		}

	}

	if err := instance.SetValid(ctx, r.Client, instanceMessage); err != nil {
		return ctrl.Result{}, err
	}

	r.log.Info("Listener reconcilation finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ListenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add listener name to index
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.VirtualService{}, options.VirtualServiceListenerNameFeild, func(rawObject client.Object) []string {
		virtualService := rawObject.(*v1alpha1.VirtualService)
		// if listener feild is empty use default listener name as index
		if virtualService.Spec.Listener == nil {
			return []string{options.DefaultListenerName}
		}
		return []string{virtualService.Spec.Listener.Name}
	}); err != nil {
		return errors.Wrap(err, "cannot add Listener names to Listener Reconcile Index")
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
		Watches(&v1alpha1.VirtualService{}, &virtualservice.EnqueueRequestForVirtualService{}, builder.WithPredicates(virtualservice.GenerationOrMetadataChangedPredicate{})).
		Watches(&v1alpha1.AccessLogConfig{}, listenerRequestMapper).
		Watches(&v1alpha1.HttpFilter{}, listenerRequestMapper).
		Watches(&v1alpha1.Route{}, listenerRequestMapper).
		Watches(&v1alpha1.Policy{}, listenerRequestMapper).
		Complete(r)
}

func (r *ListenerReconciler) makeEnvoySecret(ctx context.Context, nn string, nodeID string) error {
	kubeSecret := &corev1.Secret{}
	err := r.Get(ctx, getNamespaceNameFromSecretName(nn), kubeSecret)
	if err != nil {
		return errors.New("Secret with certificate not found")
	}

	if kubeSecret.Type != corev1.SecretTypeTLS && kubeSecret.Type != corev1.SecretTypeOpaque {
		return errors.New("Kuberentes Secret is not a type TLS or Opaque")
	}

	envoySecrets, err := xdscache.MakeEnvoySecretFromKubernetesSecret(kubeSecret)
	if err != nil {
		return err
	}

	for _, envoySecret := range envoySecrets {
		if err := r.Cache.Update(nodeID, envoySecret); err != nil {
			return err
		}
	}

	return nil
}

func getNamespaceNameFromSecretName(nn string) types.NamespacedName {
	nnSplit := strings.Split(nn, "/")

	return types.NamespacedName{
		Namespace: nnSplit[0],
		Name:      nnSplit[1],
	}
}
