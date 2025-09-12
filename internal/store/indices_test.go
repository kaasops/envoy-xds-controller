package store

import (
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	api "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
)

func makeListenerCR(ns, name string, l *listenerv3.Listener) *api.Listener {
	b, _ := protoutil.Marshaler.Marshal(l)
	return &api.Listener{
		TypeMeta:   metav1.TypeMeta{APIVersion: "envoy.kaasops.io/v1alpha1", Kind: "Listener"},
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec:       &runtime.RawExtension{Raw: b},
	}
}

func TestGetListenerAddressDuplicate_NoConflict(t *testing.T) {
	s := New()

	l1 := &listenerv3.Listener{
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{Address: "0.0.0.0", PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: 8080}}}},
	}
	l2 := &listenerv3.Listener{
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{Address: "0.0.0.0", PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: 8081}}}},
	}

	s.SetListener(makeListenerCR("default", "l1", l1))
	s.SetListener(makeListenerCR("default", "l2", l2))

	if addr, first, second, ok := s.GetListenerAddressDuplicate(); ok {
		t.Fatalf("unexpected duplicate reported: addr=%s first=%s second=%s", addr, first, second)
	}
}

func TestGetListenerAddressDuplicate_Conflict(t *testing.T) {
	s := New()

	l1 := &listenerv3.Listener{
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{Address: "127.0.0.1", PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: 8443}}}},
	}
	l2 := &listenerv3.Listener{
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{Address: "127.0.0.1", PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: 8443}}}},
	}

	// Insert both listeners with the same host:port
	s.SetListener(makeListenerCR("ns", "first", l1))
	s.SetListener(makeListenerCR("ns", "second", l2))

	addr, first, second, ok := s.GetListenerAddressDuplicate()
	if !ok {
		t.Fatalf("expected duplicate, got none")
	}
	if addr != "127.0.0.1:8443" {
		t.Fatalf("unexpected addr: %s", addr)
	}
	if first == "" || second == "" {
		t.Fatalf("expected both listener names to be set, got first='%s' second='%s'", first, second)
	}
	if first == second {
		t.Fatalf("expected different listeners to be reported, got same '%s'", first)
	}
}

func TestGetListenerAddressDuplicate_IncompleteAddressIgnored(t *testing.T) {
	s := New()

	// Listener without Address should be ignored by indexer
	incomplete := &listenerv3.Listener{}
	valid := &listenerv3.Listener{
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{Address: "0.0.0.0", PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: 9090}}}},
	}

	s.SetListener(makeListenerCR("ns", "incomplete", incomplete))
	s.SetListener(makeListenerCR("ns", "valid", valid))

	if _, _, _, ok := s.GetListenerAddressDuplicate(); ok {
		t.Fatalf("unexpected duplicate due to incomplete address")
	}
}

// Ensure updateListenerAddressIndex is refreshed on Set/Delete and map cloning works
func TestListenerIndex_RefreshOnDelete(t *testing.T) {
	s := New()
	l1 := &listenerv3.Listener{
		Address: &corev3.Address{Address: &corev3.Address_SocketAddress{SocketAddress: &corev3.SocketAddress{Address: "0.0.0.0", PortSpecifier: &corev3.SocketAddress_PortValue{PortValue: 10000}}}},
	}
	nn1 := helpers.NamespacedName{Namespace: "ns", Name: "l1"}
	s.SetListener(makeListenerCR(nn1.Namespace, nn1.Name, l1))

	idx := s.GetListenerAddressIndex()
	if len(idx) != 1 {
		t.Fatalf("expected 1 entry in index, got %d", len(idx))
	}

	s.DeleteListener(nn1)

	idx2 := s.GetListenerAddressIndex()
	if len(idx2) != 0 {
		t.Fatalf("expected index to be empty after delete, got %d", len(idx2))
	}
}
