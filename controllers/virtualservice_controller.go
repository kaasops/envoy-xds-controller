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
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"github.com/kaasops/envoy-xds-controller/pkg/hash"
	"github.com/kaasops/envoy-xds-controller/pkg/tls"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
	"github.com/kaasops/envoy-xds-controller/pkg/xds/filterchain"
	"github.com/kaasops/k8s-utils"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
)

// VirtualServiceReconciler reconciles a VirtualService object
type VirtualServiceReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Cache           xdscache.Cache
	Unmarshaler     *protojson.UnmarshalOptions
	DiscoveryClient *discovery.DiscoveryClient
	Config          config.Config
}

//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=envoy.kaasops.io,resources=virtualservices/finalizers,verbs=update

func (r *VirtualServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("VirtualService", req.NamespacedName)
	log.Info("Reconciling VirtualService")

	instance := &v1alpha1.VirtualService{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if api_errors.IsNotFound(err) {
			log.Info("VirtualService not found. Ignoring since object must be deleted")
			for _, nodeID := range NodeIDs(instance, r.Cache) {
				if err := r.Cache.Delete(nodeID, resourcev3.RouteType, getResourceName(req.Namespace, req.Name)); err != nil {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if instance.Spec.VirtualHost == nil {
		log.Error(err, "VirtualHost could not be empty")
		return ctrl.Result{}, ErrEmptySpec
	}

	// Get envoy virtualhost from virtualSerive spec
	virtualHost := &routev3.VirtualHost{}
	if err := r.Unmarshaler.Unmarshal(instance.Spec.VirtualHost.Raw, virtualHost); err != nil {
		return ctrl.Result{}, err
	}

	// Generate RouteConfiguration and add to xds cache
	routeConfig, err := filterchain.MakeRouteConfig(virtualHost, getResourceName(req.Namespace, req.Name))

	if err != nil {
		return ctrl.Result{}, err
	}

	for _, nodeID := range NodeIDs(instance, r.Cache) {
		log.Info("Adding route", "name:", routeConfig.Name)
		if err := r.Cache.Update(nodeID, routeConfig); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check VirtualService hash and skip reconcile if no changes
	checkResult, err := checkHash(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	if checkResult {
		log.Info("VirtualService has no changes. Finish Reconcile")
		return ctrl.Result{}, nil
	}

	// Check if tlsConfig valid
	certsProvider := tls.New(r.Client, r.DiscoveryClient, r.Config, instance.Namespace, log)
	index, err := certsProvider.IndexCertificateSecrets(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	errorList, err := certsProvider.Validate(ctx, index, virtualHost, instance.Spec.TlsConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(errorList) > 0 {
		instance.Status.Errors = errorList
		r.Client.Status().Update(ctx, instance)
	}

	if instance.Spec.AccessLogConfig != nil {
		if instance.Spec.AccessLog != nil {
			return ctrl.Result{}, err
		}
		accessLog := &v1alpha1.AccessLogConfig{}
		err := r.Get(ctx, instance.Spec.AccessLogConfig.NamespacedName(instance.Namespace), accessLog)
		if err != nil {
			return ctrl.Result{}, err
		}
		instance.Spec.AccessLog = accessLog.Spec
	}

	// Set default listener if listener not set
	if instance.Spec.Listener == nil {
		// TODO: fix default listerner namespace
		instance.Spec.Listener = &v1alpha1.ResourceRef{Name: DefaultListenerName}
	}

	listener := &v1alpha1.Listener{}
	err = r.Get(ctx, instance.Spec.Listener.NamespacedName(instance.Namespace), listener)
	if err != nil {
		return ctrl.Result{}, err
	}

	if GetNodeIDsAnnotation(listener) != "*" {
		if !NodeIDsContains(NodeIDs(instance, r.Cache), NodeIDs(listener, r.Cache)) {
			return ctrl.Result{}, ErrNodeIDMismatch
		}
	}

	if err := controllerutil.SetControllerReference(listener, instance, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Client.Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Updating last applied hash")
	if err := setLastAppliedHash(ctx, r.Client, instance); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Triggering listener reconiliation", "Listener.name", instance.Spec.Listener.Name)

	// listenerReconciliationChannel <- event.GenericEvent{Object: listener}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add custom index to list by listerner
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.VirtualService{}, VirtualServiceListenerFeild, func(rawObject client.Object) []string {
		virtualService := rawObject.(*v1alpha1.VirtualService)
		if virtualService.Spec.Listener == nil {
			return []string{DefaultListenerName}
		}
		return []string{virtualService.Spec.Listener.Name}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.VirtualService{}).
		Complete(r)
}

func checkHash(virtualService *v1alpha1.VirtualService) (bool, error) {
	hash, err := getHash(virtualService)
	if err != nil {
		return false, err
	}

	if virtualService.Status.LastAppliedHash != nil && *hash == *virtualService.Status.LastAppliedHash {
		return true, nil
	}

	return false, nil
}

func setLastAppliedHash(ctx context.Context, client client.Client, virtualService *v1alpha1.VirtualService) error {
	hash, err := getHash(virtualService)
	if err != nil {
		return err
	}
	virtualService.Status.LastAppliedHash = hash

	return k8s.UpdateStatus(ctx, virtualService, client)
}

func getHash(virtualService *v1alpha1.VirtualService) (*uint32, error) {
	specHash, err := json.Marshal(virtualService.Spec)
	if err != nil {
		return nil, err
	}
	annotationHash, err := json.Marshal(virtualService.Annotations)
	if err != nil {
		return nil, err
	}
	hash := hash.Get(specHash) + hash.Get(annotationHash)
	return &hash, nil
}
