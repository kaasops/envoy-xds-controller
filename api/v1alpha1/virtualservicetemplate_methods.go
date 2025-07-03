package v1alpha1

import "encoding/json"

func (vst *VirtualServiceTemplate) IsEqual(other *VirtualServiceTemplate) bool {
	if vst == nil && other == nil {
		return true
	}
	if vst == nil || other == nil {
		return false
	}
	return vst.Spec.VirtualServiceCommonSpec.IsEqual(&other.Spec.VirtualServiceCommonSpec)
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
