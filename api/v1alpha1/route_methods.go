package v1alpha1

import (
	"bytes"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
)

func (r *Route) UnmarshalV3() ([]*routev3.Route, error) {
	return r.unmarshalV3()
}

func (r *Route) UnmarshalV3AndValidate() ([]*routev3.Route, error) {
	routesV3, err := r.unmarshalV3()
	if err != nil {
		return nil, err
	}
	for _, routeV3 := range routesV3 {
		if err := routeV3.ValidateAll(); err != nil {
			return nil, err
		}
	}
	return routesV3, nil
}

func (r *Route) unmarshalV3() ([]*routev3.Route, error) {
	if len(r.Spec) == 0 {
		return nil, ErrSpecNil
	}

	routesV3 := make([]*routev3.Route, 0, len(r.Spec))
	for _, routeSpec := range r.Spec {
		var routeV3 routev3.Route
		if err := protoutil.Unmarshaler.Unmarshal(routeSpec.Raw, &routeV3); err != nil {
			return nil, err
		}
		routesV3 = append(routesV3, &routeV3)
	}
	return routesV3, nil
}

func (r *Route) IsEqual(other *Route) bool {
	if r == nil && other == nil {
		return true
	}
	if r == nil || other == nil {
		return false
	}
	if len(r.Spec) != len(other.Spec) {
		return false
	}
	for i, route := range r.Spec {
		if !bytes.Equal(other.Spec[i].Raw, route.Raw) {
			return false
		}
	}
	return true
}

func (r *Route) GetAccessGroup() string {
	accessGroup := r.GetLabels()[LabelAccessGroup]
	if accessGroup == "" {
		return GeneralAccessGroup
	}
	return accessGroup
}

func (r *Route) GetDescription() string {
	return r.Annotations[annotationDescription]
}
