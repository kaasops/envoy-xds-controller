package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	v1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/hash"
	"github.com/kaasops/k8s-utils"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultListenerName         = "https"
	VirtualServiceListenerFeild = "spec.listener.name"
	nodeIDAnnotation            = "envoy.kaasops.io/node-id"
	defaultNodeID               = "default"
)

var (
	ErrEmptySpec               = errors.New("spec could not be empty")
	ErrInvalidSpec             = errors.New("invalid config component spec")
	ErrNodeIDMismatch          = errors.New("nodeID mismatch")
	ErrMultipleAccessLogConfig = errors.New("only one access log config is allowed")
)

func NodeIDs(obj client.Object) []string {
	annotation := NodeIDsAnnotation(obj)
	if annotation == "" {
		return nil
	}
	return strings.Split(NodeIDsAnnotation(obj), ",")
}

func NodeIDsAnnotation(obj client.Object) string {
	annotations := obj.GetAnnotations()

	annotation, ok := annotations[nodeIDAnnotation]
	if !ok {
		return ""
	}

	return annotation
}

func getResourceName(namespace, name string) string {
	return fmt.Sprintf("%s-%s", namespace, name)
}

func NodeIDsContains(s1, s2 []string) bool {

	if len(s1) > len(s2) {
		return false
	}

	for _, e := range s1 {
		if !slices.Contains(s2, e) {
			return false
		}
	}

	return true
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

func virtualServiceResourceRefMapper(obj client.Object, vs v1alpha1.VirtualService) []*v1alpha1.ResourceRef {
	var resources []*v1alpha1.ResourceRef
	switch obj.(type) {
	case *v1alpha1.AccessLogConfig:
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
