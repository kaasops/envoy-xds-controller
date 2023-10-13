package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	v1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/hash"
	"github.com/kaasops/envoy-xds-controller/pkg/util/k8s"
	k8s_utils "github.com/kaasops/k8s-utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrEmptySpec               = errors.New("spec could not be empty")
	ErrInvalidSpec             = errors.New("invalid config component spec")
	ErrNodeIDMismatch          = errors.New("nodeID mismatch")
	ErrMultipleAccessLogConfig = errors.New("only one access log config is allowed")
)

func getResourceName(namespace, name string) string {
	return fmt.Sprintf("%s-%s", namespace, name)
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

	return k8s_utils.UpdateStatus(ctx, virtualService, client)
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

func virtualServiceResourceRefMapper(obj client.Object, vs v1alpha1.VirtualService) []*v1alpha1.ResourceRef {
	var resources []*v1alpha1.ResourceRef
	switch obj.(type) {
	case *v1alpha1.AccessLogConfig:
		if vs.Spec.AccessLogConfig == nil {
			return nil
		}
		return append(resources, vs.Spec.AccessLogConfig)
	case *v1alpha1.Route:
		return vs.Spec.AdditionalRoutes
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

func defaultNodeIDs(ctx context.Context, cli client.Client, namespace string) ([]string, error) {
	// TODO: Get all VirtualServices, Routes, Listeners that contains this object and set nodeIDs
	var nodeIDs []string
	listeners := &v1alpha1.ListenerList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
	}
	if err := cli.List(ctx, listeners, listOpts...); err != nil {
		return nil, err
	}
	for _, l := range listeners.Items {
		nodeIDs = append(nodeIDs, k8s.NodeIDs(l.DeepCopy())...)
	}
	return nodeIDs, nil
}
