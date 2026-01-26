package v1alpha1

import "encoding/json"

func (vst *VirtualServiceTemplate) IsEqual(other *VirtualServiceTemplate) bool {
	if vst == nil && other == nil {
		return true
	}
	if vst == nil || other == nil {
		return false
	}
	if !vst.Spec.VirtualServiceCommonSpec.IsEqual(&other.Spec.VirtualServiceCommonSpec) {
		return false
	}
	// Compare ExtraFields
	if len(vst.Spec.ExtraFields) != len(other.Spec.ExtraFields) {
		return false
	}
	for i, ef := range vst.Spec.ExtraFields {
		otherEF := other.Spec.ExtraFields[i]
		if ef == nil && otherEF == nil {
			continue
		}
		if ef == nil || otherEF == nil {
			return false
		}
		if ef.Name != otherEF.Name ||
			ef.Description != otherEF.Description ||
			ef.Type != otherEF.Type ||
			ef.Required != otherEF.Required ||
			ef.Default != otherEF.Default {
			return false
		}
		// Compare Enum slices
		if len(ef.Enum) != len(otherEF.Enum) {
			return false
		}
		for j, enumVal := range ef.Enum {
			if enumVal != otherEF.Enum[j] {
				return false
			}
		}
	}
	return true
}

func (vst *VirtualServiceTemplate) GetAccessGroup() string {
	accessGroup := vst.GetLabels()[LabelAccessGroup]
	if accessGroup == "" {
		return GeneralAccessGroup
	}
	return accessGroup
}

func (vst *VirtualServiceTemplate) GetDescription() string {
	return vst.Annotations[annotationDescription]
}

func (vst *VirtualServiceTemplate) NormalizeSpec() {
	if vst == nil {
		return
	}
	if vst.Spec.Listener != nil && vst.Spec.Listener.Namespace == nil {
		vst.Spec.Listener.Namespace = &vst.Namespace
	}
	if vst.Spec.AccessLogConfig != nil && vst.Spec.AccessLogConfig.Namespace == nil {
		vst.Spec.AccessLogConfig.Namespace = &vst.Namespace
	}
	if len(vst.Spec.AccessLogConfigs) > 0 {
		for _, accessLogConfig := range vst.Spec.AccessLogConfigs {
			if accessLogConfig.Namespace == nil {
				accessLogConfig.Namespace = &vst.Namespace
			}
		}
	}
	if len(vst.Spec.AdditionalRoutes) > 0 {
		for _, route := range vst.Spec.AdditionalRoutes {
			if route.Namespace == nil {
				route.Namespace = &vst.Namespace
			}
		}
	}
	if len(vst.Spec.AdditionalHttpFilters) > 0 {
		for _, httpFilter := range vst.Spec.AdditionalHttpFilters {
			if httpFilter.Namespace == nil {
				httpFilter.Namespace = &vst.Namespace
			}
		}
	}
	if vst.Spec.TracingRef != nil && vst.Spec.TracingRef.Namespace == nil {
		vst.Spec.TracingRef.Namespace = &vst.Namespace
	}
}

func (vst *VirtualServiceTemplate) Raw() []byte {
	if vst == nil {
		return nil
	}
	data, err := json.Marshal(vst.Spec)
	if err != nil {
		return nil
	}
	return data
}
