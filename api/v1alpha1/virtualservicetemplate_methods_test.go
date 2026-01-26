package v1alpha1

import "testing"

func TestVirtualServiceTemplate_IsEqual(t *testing.T) {
	tests := []struct {
		name     string
		vst1     *VirtualServiceTemplate
		vst2     *VirtualServiceTemplate
		expected bool
	}{
		{
			name:     "both nil",
			vst1:     nil,
			vst2:     nil,
			expected: true,
		},
		{
			name:     "first nil",
			vst1:     nil,
			vst2:     &VirtualServiceTemplate{},
			expected: false,
		},
		{
			name:     "second nil",
			vst1:     &VirtualServiceTemplate{},
			vst2:     nil,
			expected: false,
		},
		{
			name: "equal empty specs",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{},
			},
			expected: true,
		},
		{
			name: "different listener reference",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					VirtualServiceCommonSpec: VirtualServiceCommonSpec{
						Listener: &ResourceRef{Name: "listener1"},
					},
				},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					VirtualServiceCommonSpec: VirtualServiceCommonSpec{
						Listener: &ResourceRef{Name: "listener2"},
					},
				},
			},
			expected: false,
		},
		{
			name: "same ExtraFields",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "string", Required: true},
					},
				},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "string", Required: true},
					},
				},
			},
			expected: true,
		},
		{
			name: "different ExtraFields count",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "string"},
					},
				},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "string"},
						{Name: "field2", Type: "string"},
					},
				},
			},
			expected: false,
		},
		{
			name: "different ExtraFields name",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "string"},
					},
				},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field2", Type: "string"},
					},
				},
			},
			expected: false,
		},
		{
			name: "different ExtraFields type",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "string"},
					},
				},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "enum"},
					},
				},
			},
			expected: false,
		},
		{
			name: "different ExtraFields required",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "string", Required: true},
					},
				},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "string", Required: false},
					},
				},
			},
			expected: false,
		},
		{
			name: "different ExtraFields default",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "string", Default: "val1"},
					},
				},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "string", Default: "val2"},
					},
				},
			},
			expected: false,
		},
		{
			name: "same ExtraFields with enum",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "enum", Enum: []string{"a", "b", "c"}},
					},
				},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "enum", Enum: []string{"a", "b", "c"}},
					},
				},
			},
			expected: true,
		},
		{
			name: "different ExtraFields enum values",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "enum", Enum: []string{"a", "b"}},
					},
				},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "enum", Enum: []string{"a", "c"}},
					},
				},
			},
			expected: false,
		},
		{
			name: "different ExtraFields enum count",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "enum", Enum: []string{"a", "b"}},
					},
				},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "enum", Enum: []string{"a", "b", "c"}},
					},
				},
			},
			expected: false,
		},
		{
			name: "one ExtraField is nil",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{nil},
				},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{
						{Name: "field1", Type: "string"},
					},
				},
			},
			expected: false,
		},
		{
			name: "both ExtraFields nil at same index",
			vst1: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{nil},
				},
			},
			vst2: &VirtualServiceTemplate{
				Spec: VirtualServiceTemplateSpec{
					ExtraFields: []*ExtraField{nil},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.vst1.IsEqual(tt.vst2)
			if result != tt.expected {
				t.Errorf("IsEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
