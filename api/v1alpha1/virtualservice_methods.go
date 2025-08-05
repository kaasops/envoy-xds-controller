package v1alpha1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/merge"
)

const (
	AnnotationNodeIDs  = "envoy.kaasops.io/node-id"
	AnnotationEditable = "envoy.kaasops.io/editable"
	LabelAccessGroup   = "exc-access-group"
	LabelName          = "exc-name"
)

func (vs *VirtualService) GetNodeIDs() []string {
	annotations := vs.GetAnnotations()
	nodeIDsAnnotation := annotations[AnnotationNodeIDs]
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

func (vs *VirtualService) SetNodeIDs(nodeIDs []string) {
	annotations := vs.GetAnnotations()
	if len(nodeIDs) == 0 {
		delete(annotations, AnnotationNodeIDs)
	} else {
		annotations[AnnotationNodeIDs] = strings.Join(nodeIDs, ",")
	}
	vs.SetAnnotations(annotations)
}

func (vs *VirtualService) GetLabelName() string {
	name, ok := vs.GetLabels()[LabelName]
	if !ok {
		return vs.Name
	}
	return name
}

func (vs *VirtualService) SetLabelName(name string) {
	labels := vs.GetLabels()
	if len(labels) == 0 {
		labels = make(map[string]string)
	}
	labels[LabelName] = name
	vs.SetLabels(labels)
}

func (vs *VirtualService) GetAccessGroup() string {
	accessGroup := vs.GetLabels()[LabelAccessGroup]
	if accessGroup == "" {
		return GeneralAccessGroup
	}
	return accessGroup
}

func (vs *VirtualService) SetAccessGroup(accessGroup string) {
	labels := vs.GetLabels()
	if len(labels) == 0 {
		labels = make(map[string]string)
	}
	labels[LabelAccessGroup] = accessGroup
	vs.SetLabels(labels)
}

func (vs *VirtualService) SetEditable(editable bool) {
	if len(vs.GetAnnotations()) == 0 {
		vs.SetAnnotations(make(map[string]string))
	}
	vs.Annotations[AnnotationEditable] = strconv.FormatBool(editable)
}

func (vs *VirtualService) FillFromTemplate(vst *VirtualServiceTemplate, templateOpts ...TemplateOpts) error {
	vst.NormalizeSpec()

	// Validate extraFields
	if len(vst.Spec.ExtraFields) > 0 {
		// Create a map of valid extraFields from the template
		validExtraFields := make(map[string]bool)
		for _, field := range vst.Spec.ExtraFields {
			validExtraFields[field.Name] = true

			// Validate required fields
			if field.Required {
				value, exists := vs.Spec.ExtraFields[field.Name]
				if !exists || value == "" {
					return fmt.Errorf("required extra field '%s' is missing or empty", field.Name)
				}
			}

			// Validate enum fields
			if field.Type == "enum" {
				value, exists := vs.Spec.ExtraFields[field.Name]
				if exists {
					notFound := true
					for _, enum := range field.Enum {
						if enum == value {
							notFound = false
							break
						}
					}
					if notFound {
						return fmt.Errorf("extra field '%s' has invalid value '%s'", field.Name, value)
					}
				}
			}
		}

		// Validate that only extraFields defined in the template are present
		for fieldName := range vs.Spec.ExtraFields {
			if !validExtraFields[fieldName] {
				return fmt.Errorf("extra field '%s' is not defined in the template", fieldName)
			}
		}
	}

	baseData, err := json.Marshal(vst.Spec.VirtualServiceCommonSpec)
	if err != nil {
		return err
	}
	if len(vs.Spec.ExtraFields) > 0 && len(vst.Spec.ExtraFields) > 0 {
		tmpl, err := template.New("template").Funcs(template.FuncMap{}).Parse(string(baseData))
		if err != nil {
			return fmt.Errorf("failed to parse template: %w", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, vs.Spec.ExtraFields); err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}
		baseData = buf.Bytes()
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
	mergedData := merge.JSONRawMessages(baseData, svcData, tOpts)

	err = json.Unmarshal(mergedData, &vs.Spec.VirtualServiceCommonSpec)
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
	if vs.Annotations[AnnotationNodeIDs] != other.Annotations[AnnotationNodeIDs] {
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
		Namespace: helpers.GetNamespace(vs.Spec.Listener.Namespace, vs.Namespace),
		Name:      vs.Spec.Listener.Name,
	}, nil
}

func (vs *VirtualService) IsEditable() bool {
	if vs.Annotations == nil {
		return false
	}
	editable, ok := vs.Annotations[AnnotationEditable]
	if !ok {
		return false
	}
	return editable == "true"
}

func (vs *VirtualService) GetDescription() string {
	return vs.Annotations[annotationDescription]
}

func (vs *VirtualService) SetDescription(description string) {
	if vs.Annotations == nil {
		vs.Annotations = make(map[string]string)
	}
	vs.Annotations[annotationDescription] = description
}

func (vs *VirtualService) Raw() []byte {
	if vs == nil {
		return nil
	}
	data, err := json.Marshal(vs.Spec.VirtualServiceCommonSpec)
	if err != nil {
		return nil
	}
	return data
}

func (vs *VirtualService) UpdateStatus(invalid bool, message string) {
	if vs == nil {
		return
	}
	vs.Status.Invalid = invalid
	vs.Status.Message = message
}

func (vs *VirtualService) IsStatusInvalid() bool {
	if vs == nil {
		return false
	}
	return vs.Status.Invalid
}
