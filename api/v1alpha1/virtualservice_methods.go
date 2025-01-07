package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/merge"
)

const (
	AnnotationKeyEnvoyKaaSopsIoNodeID = "envoy.kaasops.io/node-id"
)

func (vs *VirtualService) GetNodeIDs() []string {
	annotations := vs.GetAnnotations()
	nodeIDsAnnotation := annotations[AnnotationKeyEnvoyKaaSopsIoNodeID]
	if nodeIDsAnnotation == "" {
		return nil
	}
	keys := make(map[string]struct{})
	var list []string
	for _, entry := range strings.Split(nodeIDsAnnotation, ",") {
		entry = strings.TrimSpace(entry)
		if _, value := keys[entry]; !value {
			keys[entry] = struct{}{}
			list = append(list, entry)
		}
	}
	return list
}

func (vs *VirtualService) FillFromTemplate(vst *VirtualServiceTemplate, templateOpts ...TemplateOpts) error {
	baseData, err := json.Marshal(vst.Spec.VirtualServiceCommonSpec)
	if err != nil {
		return err
	}
	svcData, err := json.Marshal(vs.Spec.VirtualServiceCommonSpec)
	if err != nil {
		return err
	}
	var tOpts []merge.Opt
	if len(templateOpts) > 0 {
		tOpts = make([]merge.Opt, 0, len(templateOpts))
		for _, opt := range templateOpts {
			if opt.Field == "" {
				return fmt.Errorf("template option field is empty")
			}
			var op merge.OperationType
			switch opt.Modifier {
			case ModifierMerge:
				op = merge.OperationMerge
			case ModifierReplace:
				op = merge.OperationReplace
			case ModifierDelete:
				op = merge.OperationDelete
			default:
				return fmt.Errorf("template option modifier is invalid")
			}
			tOpts = append(tOpts, merge.Opt{
				Path:      opt.Field,
				Operation: op,
			})
		}
	}
	mergedDate := merge.JSONRawMessages(baseData, svcData, tOpts)
	err = json.Unmarshal(mergedDate, &vs.Spec.VirtualServiceCommonSpec)
	if err != nil {
		return err
	}
	return nil
}

func (vs *VirtualService) IsEqual(other *VirtualService) bool {
	if vs == nil && other == nil {
		return true
	}
	if vs == nil || other == nil {
		return false
	}
	if vs.Annotations == nil || other.Annotations == nil {
		return false
	}
	if vs.Annotations[AnnotationKeyEnvoyKaaSopsIoNodeID] != other.Annotations[AnnotationKeyEnvoyKaaSopsIoNodeID] {
		return false
	}
	if !vs.Spec.VirtualServiceCommonSpec.IsEqual(&other.Spec.VirtualServiceCommonSpec) {
		return false
	}
	if (vs.Spec.Template == nil) != (other.Spec.Template == nil) {
		return false
	}
	if vs.Spec.Template != nil && other.Spec.Template != nil {
		if vs.Spec.Template.Name != other.Spec.Template.Name ||
			vs.Spec.Template.Namespace != other.Spec.Template.Namespace {
			return false
		}
	}
	if len(vs.Spec.TemplateOptions) != len(other.Spec.TemplateOptions) {
		return false
	}
	for i := range vs.Spec.TemplateOptions {
		if vs.Spec.TemplateOptions[i].Field != other.Spec.TemplateOptions[i].Field ||
			vs.Spec.TemplateOptions[i].Modifier != other.Spec.TemplateOptions[i].Modifier {
			return false
		}
	}
	return true
}

func (vs *VirtualService) GetListenerNamespacedName() (helpers.NamespacedName, error) {
	if vs.Spec.Listener == nil {
		return helpers.NamespacedName{}, fmt.Errorf("listener is nil")
	}
	return helpers.NamespacedName{
		Namespace: vs.Namespace,
		Name:      vs.Spec.Listener.Name,
	}, nil
}
