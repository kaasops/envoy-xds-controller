package controllers

import "errors"

const (
	VirtualServiceListenerFeild = "spec.listener.name"
)

var (
	ErrEmptySpec = errors.New("spec could not be empty")
)
