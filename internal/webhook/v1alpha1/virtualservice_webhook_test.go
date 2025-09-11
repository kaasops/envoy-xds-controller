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

const namespace = "ns1"

func makeSchemeVS(t *testing.T) *runtime.Scheme {
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	if err := envoyv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("add envoy scheme: %v", err)
	}
	return s
}

func TestValidateVSTracing_XOR(t *testing.T) {
	scheme := makeSchemeVS(t)
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()

	vs := &envoyv1alpha1.VirtualService{}
	vs.Namespace = namespace
	// Both inline and ref set -> error
	vs.Spec.VirtualServiceCommonSpec.Tracing = &runtime.RawExtension{Raw: []byte(`{"foo":"bar"}`)}
	vs.Spec.VirtualServiceCommonSpec.TracingRef = &envoyv1alpha1.ResourceRef{Name: "tr", Namespace: nil}

	if err := validateVSTracing(context.Background(), cl, vs); err == nil {
		t.Fatalf("expected XOR validation error, got nil")
	}
}

func TestValidateVSTracing_RefNotFound(t *testing.T) {
	scheme := makeSchemeVS(t)
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()

	vs := &envoyv1alpha1.VirtualService{}
	vs.Namespace = namespace
	vs.Spec.VirtualServiceCommonSpec.TracingRef = &envoyv1alpha1.ResourceRef{Name: "missing"}

	if err := validateVSTracing(context.Background(), cl, vs); err == nil {
		t.Fatalf("expected not found error for tracingRef, got nil")
	}
}

func TestValidateVSTracing_RefExists(t *testing.T) {
	scheme := makeSchemeVS(t)
	tr := &envoyv1alpha1.Tracing{}
	tr.Namespace = namespace
	tr.Name = "exists"
	objs := []ctrlclient.Object{tr}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()

	vs := &envoyv1alpha1.VirtualService{}
	vs.Namespace = namespace
	vs.Spec.VirtualServiceCommonSpec.TracingRef = &envoyv1alpha1.ResourceRef{Name: "exists"}

	if err := validateVSTracing(context.Background(), cl, vs); err != nil {
		t.Fatalf("expected no error for existing tracingRef, got: %v", err)
	}
}
