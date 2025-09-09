package v1alpha1

import (
	"bytes"

	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
)

func (t *Tracing) UnmarshalV3AndValidate() (*hcmv3.HttpConnectionManager_Tracing, error) {
	tracing, err := t.unmarshalV3()
	if err != nil {
		return nil, err
	}
	return tracing, tracing.ValidateAll()
}

func (t *Tracing) UnmarshalV3() (*hcmv3.HttpConnectionManager_Tracing, error) {
	return t.unmarshalV3()
}

func (t *Tracing) unmarshalV3() (*hcmv3.HttpConnectionManager_Tracing, error) {
	if t.Spec == nil {
		return nil, ErrSpecNil
	}
	var tracing hcmv3.HttpConnectionManager_Tracing
	if err := protoutil.Unmarshaler.Unmarshal(t.Spec.Raw, &tracing); err != nil {
		return nil, err
	}
	return &tracing, nil
}

func (t *Tracing) IsEqual(other *Tracing) bool {
	if t == nil && other == nil {
		return true
	}
	if t == nil || other == nil || t.Spec == nil || other.Spec == nil || t.Spec.Raw == nil || other.Spec.Raw == nil {
		return false
	}
	return bytes.Equal(t.Spec.Raw, other.Spec.Raw)
}
