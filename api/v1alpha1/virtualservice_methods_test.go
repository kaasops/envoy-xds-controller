package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVirtualService_IsEqual(t *testing.T) {
	tests := []struct {
		name     string
		vs1      *VirtualService
		vs2      *VirtualService
		expected bool
	}{
		{
			name:     "both nil",
			vs1:      nil,
			vs2:      nil,
			expected: true,
		},
		{
			name:     "first nil",
			vs1:      nil,
			vs2:      &VirtualService{},
			expected: false,
		},
		{
			name:     "second nil",
			vs1:      &VirtualService{},
			vs2:      nil,
			expected: false,
		},
		{
			name: "both have nil annotations - equal specs",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: nil},
				Spec:       VirtualServiceSpec{},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: nil},
				Spec:       VirtualServiceSpec{},
			},
			expected: true,
		},
		{
			name: "nil vs empty annotations - should be equal",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: nil},
				Spec:       VirtualServiceSpec{},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec:       VirtualServiceSpec{},
			},
			expected: true,
		},
		{
			name: "same node IDs",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{AnnotationNodeIDs: "node1,node2"},
				},
				Spec: VirtualServiceSpec{},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{AnnotationNodeIDs: "node1,node2"},
				},
				Spec: VirtualServiceSpec{},
			},
			expected: true,
		},
		{
			name: "different node IDs",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{AnnotationNodeIDs: "node1"},
				},
				Spec: VirtualServiceSpec{},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{AnnotationNodeIDs: "node2"},
				},
				Spec: VirtualServiceSpec{},
			},
			expected: false,
		},
		{
			name: "one has node IDs, other doesn't",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{AnnotationNodeIDs: "node1"},
				},
				Spec: VirtualServiceSpec{},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: nil},
				Spec:       VirtualServiceSpec{},
			},
			expected: false,
		},
		{
			name: "same template reference",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					Template: &ResourceRef{Name: "tmpl1"},
				},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					Template: &ResourceRef{Name: "tmpl1"},
				},
			},
			expected: true,
		},
		{
			name: "different template reference",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					Template: &ResourceRef{Name: "tmpl1"},
				},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					Template: &ResourceRef{Name: "tmpl2"},
				},
			},
			expected: false,
		},
		{
			name: "same template options",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					TemplateOptions: []TemplateOpts{{Field: "f1", Modifier: ModifierMerge}},
				},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					TemplateOptions: []TemplateOpts{{Field: "f1", Modifier: ModifierMerge}},
				},
			},
			expected: true,
		},
		{
			name: "different template options",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					TemplateOptions: []TemplateOpts{{Field: "f1", Modifier: ModifierMerge}},
				},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					TemplateOptions: []TemplateOpts{{Field: "f1", Modifier: ModifierReplace}},
				},
			},
			expected: false,
		},
		{
			name: "same ExtraFields",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					ExtraFields: map[string]string{"key1": "value1", "key2": "value2"},
				},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					ExtraFields: map[string]string{"key1": "value1", "key2": "value2"},
				},
			},
			expected: true,
		},
		{
			name: "different ExtraFields values",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					ExtraFields: map[string]string{"key1": "value1"},
				},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					ExtraFields: map[string]string{"key1": "value2"},
				},
			},
			expected: false,
		},
		{
			name: "different ExtraFields keys",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					ExtraFields: map[string]string{"key1": "value1"},
				},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					ExtraFields: map[string]string{"key2": "value1"},
				},
			},
			expected: false,
		},
		{
			name: "different ExtraFields count",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					ExtraFields: map[string]string{"key1": "value1"},
				},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					ExtraFields: map[string]string{"key1": "value1", "key2": "value2"},
				},
			},
			expected: false,
		},
		{
			name: "nil vs empty ExtraFields - should be equal",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					ExtraFields: nil,
				},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					ExtraFields: map[string]string{},
				},
			},
			expected: true,
		},
		{
			name: "one has ExtraFields, other doesn't",
			vs1: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					ExtraFields: map[string]string{"key1": "value1"},
				},
			},
			vs2: &VirtualService{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
				Spec: VirtualServiceSpec{
					ExtraFields: nil,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.vs1.IsEqual(tt.vs2)
			if result != tt.expected {
				t.Errorf("IsEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestVirtualService_NormalizeSpec_TracingRefNamespaceDefault(t *testing.T) {
	vs := &VirtualService{}
	vs.Namespace = "ns-app"
	vs.Spec.TracingRef = &ResourceRef{Name: "my-tracing", Namespace: nil}

	vs.NormalizeSpec()

	if vs.Spec.TracingRef == nil || vs.Spec.TracingRef.Namespace == nil {
		t.Fatalf("expected tracingRef.namespace to be set to %q, got nil", vs.Namespace)
	}
	if got := *vs.Spec.TracingRef.Namespace; got != vs.Namespace {
		t.Fatalf("expected tracingRef.namespace %q, got %q", vs.Namespace, got)
	}
}
