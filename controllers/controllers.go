package controllers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultListenerName         = "default-https"
	VirtualServiceListenerFeild = "spec.listener.name"
	nodeIDAnnotation            = "envoy.kaasops.io/node-id"
	defaultNodeID               = "main"
)

var (
	ErrEmptySpec      = errors.New("spec could not be empty")
	ErrInvalidSpec    = errors.New("invalid config component spec")
	ErrNodeIDMismatch = errors.New("NodeID mismatch")
)

func GetNodeIDsAnnotation(obj client.Object) string {
	annotations := obj.GetAnnotations()

	annotation, ok := annotations[nodeIDAnnotation]
	if !ok {
		return ""
	}

	return annotation
}

func NodeIDs(obj client.Object, cache cache.Cache) []string {
	nodeIDStr := GetNodeIDsAnnotation(obj)

	if nodeIDStr == "" {
		return []string{defaultNodeID}
	}

	if nodeIDStr == "*" {
		return cache.GetAllNodeIDs()
	}

	nodeIDs := strings.Split(nodeIDStr, ",")

	return nodeIDs
}

func getResourceName(namespace, name string) string {
	return fmt.Sprintf("%s-%s", namespace, name)
}

func NodeIDsContains(s1, s2 []string) bool {
	if len(s1) > len(s2) {
		return false
	}

	for _, e := range s1 {
		if !contains(s2, e) {
			return false
		}
	}

	return true
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
