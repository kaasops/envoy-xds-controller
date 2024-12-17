package cache

import (
	"context"
	"sync"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"golang.org/x/exp/maps"
)

type SnapshotCache struct {
	cache.SnapshotCache
	mu      sync.RWMutex
	nodeIDs map[string]struct{}
}

func NewSnapshotCache() *SnapshotCache {
	return &SnapshotCache{
		SnapshotCache: cache.NewSnapshotCache(false, cache.IDHash{}, nil),
		nodeIDs:       make(map[string]struct{}),
	}
}

func (c *SnapshotCache) SetSnapshot(ctx context.Context, nodeID string, snapshot cache.ResourceSnapshot) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nodeIDs[nodeID] = struct{}{}
	return c.SnapshotCache.SetSnapshot(ctx, nodeID, snapshot)
}

func (c *SnapshotCache) GetSnapshot(nodeID string) (cache.ResourceSnapshot, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SnapshotCache.GetSnapshot(nodeID)
}

func (c *SnapshotCache) ClearSnapshot(nodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.nodeIDs, nodeID)
	c.SnapshotCache.ClearSnapshot(nodeID)
}

func (c *SnapshotCache) GetNodeIDsAsMap() map[string]struct{} {
	res := make(map[string]struct{}, len(c.nodeIDs))
	c.mu.RLock()
	maps.Copy(res, c.nodeIDs)
	c.mu.RUnlock()
	return res
}

func (c *SnapshotCache) GetNodeIDs() []string {
	res := make([]string, 0, len(c.nodeIDs))
	c.mu.RLock()
	defer c.mu.RUnlock()
	for nodeID := range c.nodeIDs {
		res = append(res, nodeID)
	}
	return res
}

func (c *SnapshotCache) GetClusters(nodeID string) ([]*clusterv3.Cluster, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot, err := c.SnapshotCache.GetSnapshot(nodeID)
	if err != nil {
		return nil, err
	}
	data := snapshot.GetResources(resourcev3.ClusterType)
	clusters := make([]*clusterv3.Cluster, 0, len(data))
	for _, cluster := range data {
		clusters = append(clusters, cluster.(*clusterv3.Cluster))
	}
	return clusters, nil
}

func (c *SnapshotCache) GetSecrets(nodeID string) ([]*tlsv3.Secret, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot, err := c.SnapshotCache.GetSnapshot(nodeID)
	if err != nil {
		return nil, err
	}
	data := snapshot.GetResources(resourcev3.SecretType)
	secrets := make([]*tlsv3.Secret, 0, len(data))
	for _, secret := range data {
		secrets = append(secrets, secret.(*tlsv3.Secret))
	}
	return secrets, nil
}

func (c *SnapshotCache) GetRouteConfigurations(nodeID string) ([]*routev3.RouteConfiguration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot, err := c.SnapshotCache.GetSnapshot(nodeID)
	if err != nil {
		return nil, err
	}
	data := snapshot.GetResources(resourcev3.RouteType)
	rConfigs := make([]*routev3.RouteConfiguration, 0, len(data))
	for _, rc := range data {
		rConfigs = append(rConfigs, rc.(*routev3.RouteConfiguration))
	}
	return rConfigs, nil
}

func (c *SnapshotCache) GetListeners(nodeID string) ([]*listenerv3.Listener, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot, err := c.SnapshotCache.GetSnapshot(nodeID)
	if err != nil {
		return nil, err
	}
	data := snapshot.GetResources(resourcev3.ListenerType)
	listeners := make([]*listenerv3.Listener, 0, len(data))
	for _, listener := range data {
		listeners = append(listeners, listener.(*listenerv3.Listener))
	}
	return listeners, nil
}
