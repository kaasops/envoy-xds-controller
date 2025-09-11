package v1alpha1

import "testing"

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
