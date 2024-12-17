package v1alpha1

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	filev3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	annotationAutogenFilename = "envoy.kaasops.io/auto-generated-filename"
)

var (
	ErrSpecNil                               = errors.New("spec is nil")
	ErrInvalidAnnotationAutogenFilenameValue = errors.New("envoy.kaasops.io/auto-generated-filename annotation value must be true or false")
	ErrInvalidAccessLogConfigType            = errors.New("access log config type must be of type file")
)

func WithAccessLogFileName(name string) func(*opts) {
	return func(o *opts) {
		o.filename = name
	}
}

type opts struct {
	filename string
}

func (a *AccessLogConfig) UnmarshalAndValidateV3(options ...func(*opts)) (*accesslogv3.AccessLog, error) {
	accessLog, err := a.unmarshalV3(options...)
	if err != nil {
		return nil, err
	}
	if err := accessLog.ValidateAll(); err != nil {
		return nil, err
	}
	return accessLog, nil
}

func (a *AccessLogConfig) unmarshalV3(options ...func(*opts)) (*accesslogv3.AccessLog, error) {
	if a.Spec == nil {
		return nil, ErrSpecNil
	}

	accessLog := &accesslogv3.AccessLog{}
	if err := protoutil.Unmarshaler.Unmarshal(a.Spec.Raw, accessLog); err != nil {
		return nil, err
	}

	// apply options

	unmarshalOpts := &opts{
		filename: "access",
	}
	for _, o := range options {
		o(unmarshalOpts)
	}

	val, ok := a.Annotations[annotationAutogenFilename]
	if ok {
		enabled, err := strconv.ParseBool(val)
		if err != nil {
			return nil, ErrInvalidAnnotationAutogenFilenameValue
		}

		if enabled {
			fileConfig := &filev3.FileAccessLog{}
			configType, ok := accessLog.GetConfigType().(*accesslogv3.AccessLog_TypedConfig)
			if !ok {
				return nil, ErrInvalidAccessLogConfigType
			}

			if err := configType.TypedConfig.UnmarshalTo(fileConfig); err != nil {
				return nil, ErrInvalidAccessLogConfigType
			}

			fileConfig.Path += fmt.Sprintf("/%s.log", unmarshalOpts.filename)

			fileConfigAny, err := anypb.New(fileConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal fileConfig to anypb: %w", err)
			}

			accessLog.ConfigType = &accesslogv3.AccessLog_TypedConfig{
				TypedConfig: fileConfigAny,
			}
		}
	}
	return accessLog, nil
}

func (a *AccessLogConfig) IsEqual(other *AccessLogConfig) bool {
	if a == nil && other == nil {
		return true
	}
	if a == nil || other == nil {
		return false
	}

	valA, okA := a.Annotations[annotationAutogenFilename]
	valB, okB := other.Annotations[annotationAutogenFilename]
	if okA != okB || valA != valB {
		return false
	}

	if a.Spec == nil && other.Spec == nil {
		return true
	}
	if a.Spec == nil || other.Spec == nil {
		return false
	}
	if !bytes.Equal(a.Spec.Raw, other.Spec.Raw) {
		return false
	}

	return true
}
