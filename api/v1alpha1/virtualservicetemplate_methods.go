package v1alpha1

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
