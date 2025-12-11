package clusters

import (
	"testing"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tcpProxyv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// createClusterInStore is a helper to add a cluster to the mock store
func createClusterInStore(mockStore store.Store, name string) {
	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: &runtime.RawExtension{
			Raw: []byte(`{
				"name": "` + name + `",
				"connect_timeout": "30s",
				"type": "LOGICAL_DNS",
				"load_assignment": {
					"cluster_name": "` + name + `",
					"endpoints": [{
						"lb_endpoints": [{
							"endpoint": {
								"address": {
									"socket_address": {
										"address": "localhost",
										"port_value": 8080
									}
								}
							}
						}]
					}]
				}
			}`),
		},
	}
	mockStore.SetCluster(cluster)
}

// createTCPProxyFilterChain creates a filter chain with TCP proxy
func createTCPProxyFilterChain(t *testing.T, clusterName string) *listenerv3.FilterChain {
	tcpProxy := &tcpProxyv3.TcpProxy{
		StatPrefix: "test",
		ClusterSpecifier: &tcpProxyv3.TcpProxy_Cluster{
			Cluster: clusterName,
		},
	}
	tcpProxyAny, err := anypb.New(tcpProxy)
	require.NoError(t, err)

	return &listenerv3.FilterChain{
		Filters: []*listenerv3.Filter{
			{
				Name: "envoy.filters.network.tcp_proxy",
				ConfigType: &listenerv3.Filter_TypedConfig{
					TypedConfig: tcpProxyAny,
				},
			},
		},
	}
}

func TestExtractClustersFromFilterChains_EmptyChains(t *testing.T) {
	mockStore := store.New()

	result, err := ExtractClustersFromFilterChains(nil, mockStore)
	assert.NoError(t, err)
	assert.Empty(t, result)

	result, err = ExtractClustersFromFilterChains([]*listenerv3.FilterChain{}, mockStore)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestExtractClustersFromFilterChains_SingleTCPProxy(t *testing.T) {
	mockStore := store.New()
	createClusterInStore(mockStore, "test-cluster")

	filterChains := []*listenerv3.FilterChain{
		createTCPProxyFilterChain(t, "test-cluster"),
	}

	result, err := ExtractClustersFromFilterChains(filterChains, mockStore)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "test-cluster", result[0].Name)
}

func TestExtractClustersFromFilterChains_MultipleTCPProxy(t *testing.T) {
	mockStore := store.New()
	createClusterInStore(mockStore, "cluster-1")
	createClusterInStore(mockStore, "cluster-2")
	createClusterInStore(mockStore, "cluster-3")

	filterChains := []*listenerv3.FilterChain{
		createTCPProxyFilterChain(t, "cluster-1"),
		createTCPProxyFilterChain(t, "cluster-2"),
		createTCPProxyFilterChain(t, "cluster-3"),
	}

	result, err := ExtractClustersFromFilterChains(filterChains, mockStore)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	names := make([]string, len(result))
	for i, c := range result {
		names[i] = c.Name
	}
	assert.ElementsMatch(t, []string{"cluster-1", "cluster-2", "cluster-3"}, names)
}

func TestExtractClustersFromFilterChains_DuplicateClusters(t *testing.T) {
	mockStore := store.New()
	createClusterInStore(mockStore, "shared-cluster")

	// Two filter chains referencing the same cluster
	filterChains := []*listenerv3.FilterChain{
		createTCPProxyFilterChain(t, "shared-cluster"),
		createTCPProxyFilterChain(t, "shared-cluster"),
	}

	result, err := ExtractClustersFromFilterChains(filterChains, mockStore)
	assert.NoError(t, err)
	// Duplicates should be removed
	assert.Len(t, result, 1)
	assert.Equal(t, "shared-cluster", result[0].Name)
}

func TestExtractClustersFromFilterChains_ClusterNotFound(t *testing.T) {
	mockStore := store.New()
	// Don't add cluster to store

	filterChains := []*listenerv3.FilterChain{
		createTCPProxyFilterChain(t, "non-existent-cluster"),
	}

	result, err := ExtractClustersFromFilterChains(filterChains, mockStore)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Nil(t, result)
}

func TestExtractClustersFromFilterChains_WeightedClusters(t *testing.T) {
	mockStore := store.New()
	createClusterInStore(mockStore, "cluster-a")
	createClusterInStore(mockStore, "cluster-b")

	tcpProxy := &tcpProxyv3.TcpProxy{
		StatPrefix: "weighted",
		ClusterSpecifier: &tcpProxyv3.TcpProxy_WeightedClusters{
			WeightedClusters: &tcpProxyv3.TcpProxy_WeightedCluster{
				Clusters: []*tcpProxyv3.TcpProxy_WeightedCluster_ClusterWeight{
					{Name: "cluster-a", Weight: 80},
					{Name: "cluster-b", Weight: 20},
				},
			},
		},
	}
	tcpProxyAny, err := anypb.New(tcpProxy)
	require.NoError(t, err)

	filterChains := []*listenerv3.FilterChain{{
		Filters: []*listenerv3.Filter{
			{
				Name:       "envoy.filters.network.tcp_proxy",
				ConfigType: &listenerv3.Filter_TypedConfig{TypedConfig: tcpProxyAny},
			},
		},
	}}

	result, err := ExtractClustersFromFilterChains(filterChains, mockStore)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	names := make([]string, len(result))
	for i, c := range result {
		names[i] = c.Name
	}
	assert.ElementsMatch(t, []string{"cluster-a", "cluster-b"}, names)
}

func TestExtractClustersFromFilterChains_KafkaBrokerWithTCPProxy(t *testing.T) {
	mockStore := store.New()
	createClusterInStore(mockStore, "kafka-cluster")

	// Kafka broker filter (statistics only, no cluster reference)
	kafkaBrokerConfig := &anypb.Any{
		TypeUrl: "type.googleapis.com/envoy.extensions.filters.network.kafka_broker.v3.KafkaBroker",
		Value:   []byte{}, // Empty/minimal config
	}

	// TCP proxy filter (terminal, contains cluster reference)
	tcpProxy := &tcpProxyv3.TcpProxy{
		StatPrefix: "tcp",
		ClusterSpecifier: &tcpProxyv3.TcpProxy_Cluster{
			Cluster: "kafka-cluster",
		},
	}
	tcpProxyConfig, err := anypb.New(tcpProxy)
	require.NoError(t, err)

	filterChains := []*listenerv3.FilterChain{{
		Filters: []*listenerv3.Filter{
			{
				Name:       "envoy.filters.network.kafka_broker",
				ConfigType: &listenerv3.Filter_TypedConfig{TypedConfig: kafkaBrokerConfig},
			},
			{
				Name:       "envoy.filters.network.tcp_proxy",
				ConfigType: &listenerv3.Filter_TypedConfig{TypedConfig: tcpProxyConfig},
			},
		},
	}}

	result, err := ExtractClustersFromFilterChains(filterChains, mockStore)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "kafka-cluster", result[0].Name)
}

func TestExtractClustersFromFilterChains_EmptyFilters(t *testing.T) {
	mockStore := store.New()

	filterChains := []*listenerv3.FilterChain{{
		Filters: []*listenerv3.Filter{}, // empty filters list
	}}

	result, err := ExtractClustersFromFilterChains(filterChains, mockStore)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestExtractClustersFromFilterChains_FilterWithoutTypedConfig(t *testing.T) {
	mockStore := store.New()

	filterChains := []*listenerv3.FilterChain{{
		Filters: []*listenerv3.Filter{
			{
				Name:       "envoy.filters.network.tcp_proxy",
				ConfigType: nil, // no TypedConfig
			},
		},
	}}

	result, err := ExtractClustersFromFilterChains(filterChains, mockStore)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestExtractClustersFromTCPProxyFilter_EmptyCluster(t *testing.T) {
	tcpProxy := &tcpProxyv3.TcpProxy{
		StatPrefix: "test",
		ClusterSpecifier: &tcpProxyv3.TcpProxy_Cluster{
			Cluster: "", // empty cluster name
		},
	}
	tcpProxyAny, err := anypb.New(tcpProxy)
	require.NoError(t, err)

	filter := &listenerv3.Filter{
		Name:       "envoy.filters.network.tcp_proxy",
		ConfigType: &listenerv3.Filter_TypedConfig{TypedConfig: tcpProxyAny},
	}

	result, err := extractClustersFromTCPProxyFilter(filter)
	assert.NoError(t, err)
	assert.Empty(t, result, "Empty cluster name should not be included")
}

func TestExtractClustersFromTCPProxyFilter_NilTypedConfig(t *testing.T) {
	filter := &listenerv3.Filter{
		Name:       "envoy.filters.network.tcp_proxy",
		ConfigType: nil,
	}

	result, err := extractClustersFromTCPProxyFilter(filter)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestRemoveDuplicateStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all duplicates",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
		{
			name:     "empty strings filtered",
			input:    []string{"a", "", "b", ""},
			expected: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeDuplicateStrings(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
