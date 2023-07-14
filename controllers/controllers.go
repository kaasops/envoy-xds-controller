package controllers

import "errors"

const (
	DefaultListenerName         = "default-https"
	VirtualServiceListenerFeild = "spec.listener.name"
)

var (
	ErrEmptySpec      = errors.New("spec could not be empty")
	ErrInvalidSpec    = errors.New("invalid config component spec")
	ErrNodeIDMismatch = errors.New("NodeID mismatch")
)
