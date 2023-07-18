package controllers

import (
	"errors"

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

func NodeID(obj client.Object) string {
	annotations := obj.GetAnnotations()

	nodeID, ok := annotations[nodeIDAnnotation]
	if !ok {
		return defaultNodeID
	}

	return nodeID
}
