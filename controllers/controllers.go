package controllers

import "errors"

var (
	ErrEmptySpec = errors.New("spec could not be empty")
)
