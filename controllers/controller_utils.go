package controllers

import (
	"context"
	"fmt"

	v1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getResourceName(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func virtualServiceResourceRefMapper(obj client.Object, vs v1alpha1.VirtualService) []*v1alpha1.ResourceRef {
	var resources []*v1alpha1.ResourceRef
	switch obj.(type) {
	case *v1alpha1.AccessLogConfig:
		if vs.Spec.AccessLogConfig != nil {
			return append(resources, vs.Spec.AccessLogConfig)
		}
	case *v1alpha1.Route:
		if vs.Spec.AdditionalRoutes != nil {
			return vs.Spec.AdditionalRoutes
		}
	case *v1alpha1.HttpFilter:
		if vs.Spec.AdditionalHttpFilters != nil {
			return vs.Spec.AdditionalHttpFilters
		}
	case *v1alpha1.Policy:
		if vs.Spec.RBAC != nil {
			if vs.Spec.RBAC.AdditionalPolicies != nil {
				return vs.Spec.RBAC.AdditionalPolicies
			}
		}
	case *v1alpha1.VirtualServiceTemplate:
		if vs.Spec.Template == nil {
			return nil
		}
		return append(resources, vs.Spec.Template)
	}
	return nil
}

func refContains(resources []*v1alpha1.ResourceRef, obj client.Object) bool {
	for _, res := range resources {
		if res.Name == obj.GetName() {
			return true
		}
	}
	return false
}

func defaultNodeIDs(ctx context.Context, cl client.Client, namespace string) ([]string, error) {
	// TODO: Get all VirtualServices, Routes, Listeners that contains this object and set nodeIDs
	var nodeIDs []string
	listeners := &v1alpha1.ListenerList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
	}
	if err := cl.List(ctx, listeners, listOpts...); err != nil {
		return nil, errors.Wrap(err, errors.GetFromKubernetesMessage)
	}
	for _, l := range listeners.Items {
		for _, v := range k8s.NodeIDs(l.DeepCopy()) {
			if !contains(nodeIDs, v) {
				nodeIDs = append(nodeIDs, v)
			}
		}
	}
	return nodeIDs, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
