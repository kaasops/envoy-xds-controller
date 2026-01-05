package utils

import (
	"testing"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

func TestClusterSlicePool(t *testing.T) {
	// Get a slice from the pool
	s1 := GetClusterSlice()
	if s1 == nil {
		t.Fatal("GetClusterSlice returned nil")
	}

	// Verify slice is empty but has capacity
	if len(*s1) != 0 {
		t.Errorf("New slice should be empty, got length %d", len(*s1))
	}
	if cap(*s1) < 16 {
		t.Errorf("New slice should have capacity of at least 16, got %d", cap(*s1))
	}

	// Add some data
	*s1 = append(*s1, &cluster.Cluster{Name: "test-cluster"})

	// Return to the pool
	PutClusterSlice(s1)

	// Get another slice
	s2 := GetClusterSlice()
	if s2 == nil {
		t.Fatal("GetClusterSlice returned nil on second call")
	}

	// Verify the slice is empty (reused and cleared)
	if len(*s2) != 0 {
		t.Errorf("Reused slice should be empty, got length %d", len(*s2))
	}

	// Return nil should not panic
	PutClusterSlice(nil)

	// Return to the pool
	PutClusterSlice(s2)
}

func TestHTTPFilterSlicePool(t *testing.T) {
	// Get a slice from the pool
	s1 := GetHTTPFilterSlice()
	if s1 == nil {
		t.Fatal("GetHTTPFilterSlice returned nil")
	}

	// Verify slice is empty but has capacity
	if len(*s1) != 0 {
		t.Errorf("New slice should be empty, got length %d", len(*s1))
	}
	if cap(*s1) < 8 {
		t.Errorf("New slice should have capacity of at least 8, got %d", cap(*s1))
	}

	// Add some data
	*s1 = append(*s1, &hcmv3.HttpFilter{Name: "test-filter"})

	// Return to the pool
	PutHTTPFilterSlice(s1)

	// Get another slice
	s2 := GetHTTPFilterSlice()
	if s2 == nil {
		t.Fatal("GetHTTPFilterSlice returned nil on second call")
	}

	// Verify the slice is empty (reused and cleared)
	if len(*s2) != 0 {
		t.Errorf("Reused slice should be empty, got length %d", len(*s2))
	}

	// Return nil should not panic
	PutHTTPFilterSlice(nil)

	// Return to the pool
	PutHTTPFilterSlice(s2)
}

func BenchmarkClusterSlicePoolVsNew(b *testing.B) {
	cluster1 := &cluster.Cluster{Name: "cluster1"}
	cluster2 := &cluster.Cluster{Name: "cluster2"}

	b.Run("WithPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s := GetClusterSlice()
			*s = append(*s, cluster1, cluster2)
			PutClusterSlice(s)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s := make([]*cluster.Cluster, 0, 16)
			s = append(s, cluster1, cluster2)
			_ = s
		}
	})
}

func BenchmarkHTTPFilterSlicePoolVsNew(b *testing.B) {
	filter1 := &hcmv3.HttpFilter{Name: "filter1"}
	filter2 := &hcmv3.HttpFilter{Name: "filter2"}

	b.Run("WithPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s := GetHTTPFilterSlice()
			*s = append(*s, filter1, filter2)
			PutHTTPFilterSlice(s)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s := make([]*hcmv3.HttpFilter, 0, 8)
			s = append(s, filter1, filter2)
			_ = s
		}
	})
}
