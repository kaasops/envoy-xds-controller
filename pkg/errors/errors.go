package errors

import (
	"errors"
	"fmt"
)

type customError struct {
	originalError    error
	message          string
	updateKubeStatus bool

	// options map[string]string
}

func (error customError) Error() string {
	if error.message != "" {
		return fmt.Sprintf("%s: %v", error.message, error.originalError)
	}
	return error.originalError.Error()
}

func New(msg string) error {
	return customError{
		originalError: errors.New(msg),
	}
}

func Newf(msg string, args ...interface{}) error {
	return customError{
		originalError: fmt.Errorf(msg, args...),
	}
}

func NewUKS(msg string) error {
	return customError{
		originalError:    errors.New(msg),
		updateKubeStatus: true,
	}
}

func Wrap(err error, msg string) error {
	if customErr, ok := err.(customError); ok {
		return customError{
			originalError:    err,
			message:          msg,
			updateKubeStatus: customErr.updateKubeStatus,
		}
	}

	return customError{
		originalError: err,
		message:       msg,
	}
}

func WrapUKS(err error, msg string) error {
	if _, ok := err.(customError); ok {
		return customError{
			originalError:    err,
			message:          msg,
			updateKubeStatus: true,
		}
	}

	return customError{
		originalError:    err,
		message:          msg,
		updateKubeStatus: true,
	}
}

func NeedStatusUpdate(err error) bool {
	if customErr, ok := err.(customError); ok {
		return customErr.updateKubeStatus
	}
	return false
}
