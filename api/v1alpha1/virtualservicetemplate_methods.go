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
