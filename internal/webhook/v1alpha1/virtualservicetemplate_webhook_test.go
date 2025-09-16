package v1alpha1

import (
	"context"
	"testing"

	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const namespace = "default"

func makeScheme(t *testing.T) *runtime.Scheme {
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	if err := envoyv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("add envoy scheme: %v", err)
	}
	return s
}

func TestValidateTemplateTracing_XOR(t *testing.T) {
	scheme := makeScheme(t)
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()

	vst := &envoyv1alpha1.VirtualServiceTemplate{}
	vst.Namespace = namespace
	// Both inline and ref set -> error
	vst.Spec.VirtualServiceCommonSpec.Tracing = &runtime.RawExtension{Raw: []byte(`{"foo":"bar"}`)}
	vst.Spec.VirtualServiceCommonSpec.TracingRef = &envoyv1alpha1.ResourceRef{Name: "tr", Namespace: nil}

	if err := validateTemplateTracing(context.Background(), cl, vst); err == nil {
		t.Fatalf("expected XOR validation error, got nil")
	}
}

func TestValidateTemplateTracing_RefNotFound(t *testing.T) {
	scheme := makeScheme(t)
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()

	vst := &envoyv1alpha1.VirtualServiceTemplate{}
	vst.Namespace = namespace
	vst.Spec.VirtualServiceCommonSpec.TracingRef = &envoyv1alpha1.ResourceRef{Name: "missing"}

	if err := validateTemplateTracing(context.Background(), cl, vst); err == nil {
		t.Fatalf("expected not found error for tracingRef, got nil")
	}
}

func TestValidateTemplateTracing_RefExists(t *testing.T) {
	scheme := makeScheme(t)
	tr := &envoyv1alpha1.Tracing{}
	tr.Namespace = namespace
	tr.Name = "exists"
	// Minimal spec (may be nil); existence check should pass without validating Tracing content here
	objs := []ctrlclient.Object{tr}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()

	vst := &envoyv1alpha1.VirtualServiceTemplate{}
	vst.Namespace = namespace
	vst.Spec.VirtualServiceCommonSpec.TracingRef = &envoyv1alpha1.ResourceRef{Name: "exists"}

	if err := validateTemplateTracing(context.Background(), cl, vst); err != nil {
		t.Fatalf("expected no error for existing tracingRef, got: %v", err)
	}
}
