package utils

import (
	"testing"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

func TestStringSlicePool(t *testing.T) {
	// Get a slice from the pool
	s1 := GetStringSlice()
	if s1 == nil {
		t.Fatal("GetStringSlice returned nil")
	}

	// Verify slice is empty but has capacity
	if len(*s1) != 0 {
		t.Errorf("New slice should be empty, got length %d", len(*s1))
	}
	if cap(*s1) < 8 {
		t.Errorf("New slice should have capacity of at least 8, got %d", cap(*s1))
	}

	// Add some data
	*s1 = append(*s1, "test1", "test2", "test3")

	// Return to the pool
	PutStringSlice(s1)

	// Get another slice
	s2 := GetStringSlice()
	if s2 == nil {
		t.Fatal("GetStringSlice returned nil on second call")
	}

	// Verify the slice is empty (reused and cleared)
	if len(*s2) != 0 {
		t.Errorf("Reused slice should be empty, got length %d", len(*s2))
	}

	// Add some different data
	*s2 = append(*s2, "other1", "other2")

	// Return nil should not panic
	PutStringSlice(nil)

	// Return to the pool
	PutStringSlice(s2)
}

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

func TestSafeCopyStringSlice(t *testing.T) {
	// Test with empty source
	src1 := []string{}
	dst1 := SafeCopyStringSlice(src1)
	if dst1 == nil {
		t.Fatal("SafeCopyStringSlice returned nil for empty source")
	}
	if len(*dst1) != 0 {
		t.Errorf("Copy of empty slice should be empty, got length %d", len(*dst1))
	}
	PutStringSlice(dst1)

	// Test with non-empty source
	src2 := []string{"test1", "test2", "test3"}
	dst2 := SafeCopyStringSlice(src2)
	if dst2 == nil {
		t.Fatal("SafeCopyStringSlice returned nil for non-empty source")
	}
	if len(*dst2) != len(src2) {
		t.Errorf("Copy length mismatch: expected %d, got %d", len(src2), len(*dst2))
	}
	for i, v := range src2 {
		if (*dst2)[i] != v {
			t.Errorf("Copy value mismatch at index %d: expected %s, got %s", i, v, (*dst2)[i])
		}
	}
	PutStringSlice(dst2)
}

func TestSafeCopyClusterSlice(t *testing.T) {
	// Test with empty source
	src1 := []*cluster.Cluster{}
	dst1 := SafeCopyClusterSlice(src1)
	if dst1 == nil {
		t.Fatal("SafeCopyClusterSlice returned nil for empty source")
	}
	if len(*dst1) != 0 {
		t.Errorf("Copy of empty slice should be empty, got length %d", len(*dst1))
	}
	PutClusterSlice(dst1)

	// Test with non-empty source
	src2 := []*cluster.Cluster{
		{Name: "cluster1"},
		{Name: "cluster2"},
		nil, // Test handling of nil entries
	}
	dst2 := SafeCopyClusterSlice(src2)
	if dst2 == nil {
		t.Fatal("SafeCopyClusterSlice returned nil for non-empty source")
	}
	// Should skip nil entries
	expectedLen := len(src2) - 1 // -1 for the nil entry
	if len(*dst2) != expectedLen {
		t.Errorf("Copy length mismatch: expected %d, got %d", expectedLen, len(*dst2))
	}
	for i := 0; i < expectedLen; i++ {
		if (*dst2)[i].Name != src2[i].Name {
			t.Errorf("Copy value mismatch at index %d: expected %s, got %s", i, src2[i].Name, (*dst2)[i].Name)
		}
	}
	PutClusterSlice(dst2)
}

func TestSafeCopyHTTPFilterSlice(t *testing.T) {
	// Test with empty source
	src1 := []*hcmv3.HttpFilter{}
	dst1 := SafeCopyHTTPFilterSlice(src1)
	if dst1 == nil {
		t.Fatal("SafeCopyHTTPFilterSlice returned nil for empty source")
	}
	if len(*dst1) != 0 {
		t.Errorf("Copy of empty slice should be empty, got length %d", len(*dst1))
	}
	PutHTTPFilterSlice(dst1)

	// Test with non-empty source
	src2 := []*hcmv3.HttpFilter{
		{Name: "filter1"},
		{Name: "filter2"},
		nil, // Test handling of nil entries
	}
	dst2 := SafeCopyHTTPFilterSlice(src2)
	if dst2 == nil {
		t.Fatal("SafeCopyHTTPFilterSlice returned nil for non-empty source")
	}
	// Should skip nil entries
	expectedLen := len(src2) - 1 // -1 for the nil entry
	if len(*dst2) != expectedLen {
		t.Errorf("Copy length mismatch: expected %d, got %d", expectedLen, len(*dst2))
	}
	for i := 0; i < expectedLen; i++ {
		if (*dst2)[i].Name != src2[i].Name {
			t.Errorf("Copy value mismatch at index %d: expected %s, got %s", i, src2[i].Name, (*dst2)[i].Name)
		}
	}
	PutHTTPFilterSlice(dst2)
}

func BenchmarkStringSlicePoolVsNew(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s := GetStringSlice()
			*s = append(*s, "test1", "test2", "test3")
			PutStringSlice(s)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s := make([]string, 0, 8)
			s = append(s, "test1", "test2", "test3")
			_ = s
		}
	})
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
