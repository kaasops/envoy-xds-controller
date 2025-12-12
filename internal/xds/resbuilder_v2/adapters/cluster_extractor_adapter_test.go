package adapters

import (
	"testing"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tcpProxyv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/clusters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestClusterExtractorAdapter_DelegatesToClustersPackage(t *testing.T) {
	// Helper function to create a cluster in store
	createClusterInStore := func(mockStore store.Store, name string) {
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

	t.Run("Adapter correctly delegates to clusters.ExtractClustersFromFilterChains", func(t *testing.T) {
		mockStore := store.New()
		createClusterInStore(mockStore, "kafka-cluster")

		adapter := NewClusterExtractorAdapter(clusters.NewBuilder(mockStore), mockStore)

		// Create filter chain with kafka_broker + tcp_proxy
		kafkaBrokerConfig := &anypb.Any{
			TypeUrl: "type.googleapis.com/envoy.extensions.filters.network.kafka_broker.v3.KafkaBroker",
			Value:   []byte(`{"stat_prefix": "kafka"}`),
		}

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

		result, err := adapter.ExtractClustersFromFilterChains(filterChains)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "kafka-cluster", result[0].Name)
	})

	t.Run("Adapter handles TCP proxy only", func(t *testing.T) {
		mockStore := store.New()
		createClusterInStore(mockStore, "test-cluster")

		adapter := NewClusterExtractorAdapter(clusters.NewBuilder(mockStore), mockStore)

		tcpProxy := &tcpProxyv3.TcpProxy{
			StatPrefix: "tcp",
			ClusterSpecifier: &tcpProxyv3.TcpProxy_Cluster{
				Cluster: "test-cluster",
			},
		}
		tcpProxyConfig, err := anypb.New(tcpProxy)
		require.NoError(t, err)

		filterChains := []*listenerv3.FilterChain{{
			Filters: []*listenerv3.Filter{
				{
					Name:       "envoy.filters.network.tcp_proxy",
					ConfigType: &listenerv3.Filter_TypedConfig{TypedConfig: tcpProxyConfig},
				},
			},
		}}

		result, err := adapter.ExtractClustersFromFilterChains(filterChains)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "test-cluster", result[0].Name)
	})
}
