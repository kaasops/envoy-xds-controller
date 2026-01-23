package updater

import (
	"testing"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

const testNodeID = "test-node"

// TestSortFilterChains verifies that sortFilterChains sorts by name.
func TestSortFilterChains(t *testing.T) {
	fcs := []*listenerv3.FilterChain{
		{Name: "fc-c"},
		{Name: "fc-a"},
		{Name: "fc-b"},
	}
	sortFilterChains(fcs)

	expected := []string{"fc-a", "fc-b", "fc-c"}
	for i, fc := range fcs {
		if fc.Name != expected[i] {
			t.Errorf("index %d: expected %s, got %s", i, expected[i], fc.Name)
		}
	}
}

// TestSortResources verifies that sortResources sorts by name.
func TestSortResources(t *testing.T) {
	resources := []types.Resource{
		&clusterv3.Cluster{Name: "cluster-z"},
		&clusterv3.Cluster{Name: "cluster-a"},
		&clusterv3.Cluster{Name: "cluster-m"},
	}
	sortResources(resources)

	expected := []string{"cluster-a", "cluster-m", "cluster-z"}
	for i, res := range resources {
		if name := cachev3.GetResourceName(res); name != expected[i] {
			t.Errorf("index %d: expected %s, got %s", i, expected[i], name)
		}
	}
}

// TestMixerDeterministicOutput verifies that Mixer produces the same output
// regardless of the order in which resources are added.
func TestMixerDeterministicOutput(t *testing.T) {
	clusters := []*clusterv3.Cluster{
		{Name: "cluster-a", ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_STATIC}},
		{Name: "cluster-b", ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_STATIC}},
		{Name: "cluster-c", ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_STATIC}},
	}

	// Add in different orders
	orders := [][]int{
		{0, 1, 2}, // a, b, c
		{2, 0, 1}, // c, a, b
		{1, 2, 0}, // b, c, a
	}

	results := make([][]types.Resource, 0, len(orders))
	for _, order := range orders {
		mixer := NewMixer()
		for _, idx := range order {
			mixer.Add(testNodeID, resource.ClusterType, clusters[idx])
		}
		result, _ := mixer.Mix(nil)
		results = append(results, result[testNodeID][resource.ClusterType])
	}

	// All results should have the same order
	for i := 1; i < len(results); i++ {
		for j := 0; j < 3; j++ {
			name0 := cachev3.GetResourceName(results[0][j])
			nameI := cachev3.GetResourceName(results[i][j])
			if name0 != nameI {
				t.Errorf("order %d, position %d: expected %s, got %s", i, j, name0, nameI)
			}
		}
	}
}

// TestMixerVersionStability verifies that snapshot versions remain stable
// when the same resources are processed in different orders.
func TestMixerVersionStability(t *testing.T) {
	clusters := []*clusterv3.Cluster{
		{Name: "cluster-a", ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_STATIC}},
		{Name: "cluster-b", ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_STATIC}},
		{Name: "cluster-c", ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_STATIC}},
	}

	// Create initial snapshot
	mixer := NewMixer()
	for _, c := range clusters {
		mixer.Add(testNodeID, resource.ClusterType, c)
	}
	result, _ := mixer.Mix(nil)
	snapshot, _ := cachev3.NewSnapshot("1", result[testNodeID])

	// Reconcile with different add orders
	addOrders := [][]int{{2, 0, 1}, {1, 2, 0}, {0, 2, 1}}
	prevSnapshot := snapshot
	versionIncrements := 0

	for _, order := range addOrders {
		m := NewMixer()
		for _, idx := range order {
			m.Add(testNodeID, resource.ClusterType, clusters[idx])
		}
		res, _ := m.Mix(nil)

		newSnapshot, hasChanges, _ := updateSnapshot(prevSnapshot, res[testNodeID])
		if hasChanges {
			versionIncrements++
		}
		prevSnapshot = newSnapshot
	}

	if versionIncrements > 0 {
		t.Errorf("expected 0 version increments, got %d", versionIncrements)
	}
}

// BenchmarkSortResources measures sorting performance.
func BenchmarkSortResources(b *testing.B) {
	clusters := make([]types.Resource, 100)
	for i := range clusters {
		clusters[i] = &clusterv3.Cluster{
			Name:                 "cluster-" + string(rune('a'+i%26)) + string(rune('a'+i/26)),
			ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_STATIC},
		}
	}

	b.ResetTimer()
	for range b.N {
		toSort := make([]types.Resource, len(clusters))
		copy(toSort, clusters)
		sortResources(toSort)
	}
}

// BenchmarkSortFilterChains measures filter chain sorting performance.
func BenchmarkSortFilterChains(b *testing.B) {
	fcs := make([]*listenerv3.FilterChain, 50)
	for i := range fcs {
		fcs[i] = &listenerv3.FilterChain{
			Name: "fc-" + string(rune('a'+i%26)) + string(rune('a'+i/26)),
		}
	}

	b.ResetTimer()
	for range b.N {
		toSort := make([]*listenerv3.FilterChain, len(fcs))
		copy(toSort, fcs)
		sortFilterChains(toSort)
	}
}
