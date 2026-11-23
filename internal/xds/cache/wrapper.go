package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

type SnapshotCache struct {
	cache.SnapshotCache
	mu      sync.RWMutex
	nodeIDs map[string]time.Time
	// served, history and offered are kept per node for freezing and rollback, see freeze.go
	served  map[string]Published
	history map[string][]Published
	offered map[string]cache.ResourceSnapshot
	frozen  map[string]Freeze
	// historyLimit is how many previously served snapshots each node keeps
	historyLimit int
	// versionFloor is the highest version a freeze or rollback has served, see NextVersion
	versionFloor int64
	// now is the clock of NextVersion
	now func() time.Time
}

func NewSnapshotCache() *SnapshotCache {
	return &SnapshotCache{
		// ADS mode sends the resource types of a snapshot update in order, CDS before LDS:
		// otherwise Envoy may get a listener before the cluster its filter needs and reject it
		SnapshotCache: cache.NewSnapshotCache(true, cache.IDHash{}, nil),
		nodeIDs:       make(map[string]time.Time),
		served:        make(map[string]Published),
		history:       make(map[string][]Published),
		offered:       make(map[string]cache.ResourceSnapshot),
		frozen:        make(map[string]Freeze),
		now:           time.Now,
	}
}

func (c *SnapshotCache) SetSnapshot(ctx context.Context, nodeID string, snapshot cache.ResourceSnapshot) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.nodeIDs[nodeID]; !exists {
		c.nodeIDs[nodeID] = time.Time{}
	}
	c.offered[nodeID] = snapshot
	if err := c.validateCache(nodeID, snapshot); err != nil {
		return fmt.Errorf("snapshot is invalid: %w", err)
	}
	if _, frozen := c.frozen[nodeID]; frozen {
		return nil
	}
	if c.historyLimit > 0 && c.unchanged(nodeID, snapshot) {
		// a node without services gets the same empty snapshot on every build
		return nil
	}
	return c.publish(ctx, nodeID, snapshot)
}

// unchanged reports whether the node is served this content already, whatever the versions: the same
// content again needs no update. The caller holds c.mu.
func (c *SnapshotCache) unchanged(nodeID string, snapshot cache.ResourceSnapshot) bool {
	served, ok := c.served[nodeID]
	return ok && served.ID() == SnapshotID(snapshot)
}

// LastPublication returns the last successful snapshot write for this node in this process.
func (c *SnapshotCache) LastPublication(nodeID string) time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.nodeIDs[nodeID]
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
	delete(c.served, nodeID)
	delete(c.history, nodeID)
	delete(c.offered, nodeID)
	c.SnapshotCache.ClearSnapshot(nodeID)
}

func (c *SnapshotCache) GetNodeIDsAsMap() map[string]struct{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make(map[string]struct{}, len(c.nodeIDs))
	for nodeID := range c.nodeIDs {
		res[nodeID] = struct{}{}
	}
	return res
}

func (c *SnapshotCache) GetNodeIDs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]string, 0, len(c.nodeIDs))
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
		tlsSecret := secret.(*tlsv3.Secret)
		copySecret := proto.Clone(tlsSecret).(*tlsv3.Secret)
		secrets = append(secrets, copySecret)
	}
	return secrets, nil
}

func (c *SnapshotCache) GetVersions(nodeID string) (map[string]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot, err := c.SnapshotCache.GetSnapshot(nodeID)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, 5)
	for typ, typeURL := range map[string]resourcev3.Type{
		"clusters":  resourcev3.ClusterType,
		"routes":    resourcev3.RouteType,
		"listeners": resourcev3.ListenerType,
		"secrets":   resourcev3.SecretType,
	} {
		m[typ] = snapshot.GetVersion(typeURL)
	}
	return m, nil
}

func (c *SnapshotCache) GetRouteConfigurations(nodeID string) ([]*routev3.RouteConfiguration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot, err := c.SnapshotCache.GetSnapshot(nodeID)
	if err != nil {
		return nil, err
	}
	return getRouteConfigurationsFromSnapshot(snapshot), nil
}

func (c *SnapshotCache) GetListeners(nodeID string) ([]*listenerv3.Listener, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot, err := c.SnapshotCache.GetSnapshot(nodeID)
	if err != nil {
		return nil, err
	}
	return getListenersFromSnapshot(snapshot), nil
}

// DuplicateListenerAddress reports two listeners of one node bound to the same socket.
type DuplicateListenerAddress struct {
	nodeID, listener, previous, address string
}

// NodeID returns the node whose snapshot was rejected.
func (e *DuplicateListenerAddress) NodeID() string { return e.nodeID }

// Address returns the shared socket address as host:port.
func (e *DuplicateListenerAddress) Address() string { return e.address }

// Listeners returns the rejected listener and the one already bound to the address.
func (e *DuplicateListenerAddress) Listeners() (listener, previous string) {
	return e.listener, e.previous
}

func (e *DuplicateListenerAddress) Error() string {
	return fmt.Sprintf("'%s' has duplicate address '%s' as existing listener '%s'", e.listener, e.address, e.previous)
}

// ListenersIn reports whether both listeners are in the namespace, so their names may be shown there.
func (e *DuplicateListenerAddress) ListenersIn(namespace string) bool {
	prefix := namespace + "/"
	return namespace != "" && strings.HasPrefix(e.listener, prefix) && strings.HasPrefix(e.previous, prefix)
}

func (e *DuplicateListenerAddress) DetailForNamespace(namespace string) string {
	prefix := namespace + "/"
	if namespace != "" && strings.HasPrefix(e.listener, prefix) && strings.HasPrefix(e.previous, prefix) {
		return e.Error()
	}
	return "duplicate address"
}

func (c *SnapshotCache) validateCache(nodeID string, snapshot cache.ResourceSnapshot) error {
	listeners := getListenersFromSnapshot(snapshot)
	sort.Slice(listeners, func(i, j int) bool { return listeners[i].GetName() < listeners[j].GetName() })
	addressListener := make(map[string]string, len(listeners))
	for _, listener := range listeners {
		host := listener.GetAddress().GetSocketAddress().GetAddress()
		port := listener.GetAddress().GetSocketAddress().GetPortValue()
		hostPort := fmt.Sprintf("%s:%d", host, port)
		if existListener, ok := addressListener[hostPort]; ok {
			return &DuplicateListenerAddress{
				nodeID: nodeID, listener: listener.GetName(), previous: existListener, address: hostPort,
			}
		}
		addressListener[hostPort] = listener.GetName()
	}
	return nil
}

// GetRouteConfigurationsAndListeners returns both resource types read from a single
// snapshot. Calling GetRouteConfigurations and GetListeners separately would take two
// independent read locks, so a SetSnapshot landing in between could leave the caller
// correlating route configurations from one snapshot with listeners from the next.
func (c *SnapshotCache) GetRouteConfigurationsAndListeners(
	nodeID string,
) ([]*routev3.RouteConfiguration, []*listenerv3.Listener, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot, err := c.SnapshotCache.GetSnapshot(nodeID)
	if err != nil {
		return nil, nil, err
	}
	return getRouteConfigurationsFromSnapshot(snapshot), getListenersFromSnapshot(snapshot), nil
}

func getRouteConfigurationsFromSnapshot(snapshot cache.ResourceSnapshot) []*routev3.RouteConfiguration {
	data := snapshot.GetResources(resourcev3.RouteType)
	rConfigs := make([]*routev3.RouteConfiguration, 0, len(data))
	for _, rc := range data {
		rConfigs = append(rConfigs, rc.(*routev3.RouteConfiguration))
	}
	return rConfigs
}

func getListenersFromSnapshot(snapshot cache.ResourceSnapshot) []*listenerv3.Listener {
	data := snapshot.GetResources(resourcev3.ListenerType)
	listeners := make([]*listenerv3.Listener, 0, len(data))
	for _, listener := range data {
		listeners = append(listeners, listener.(*listenerv3.Listener))
	}
	return listeners
}

// ResourceVersion contains version info for a single resource
type ResourceVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ResourceVersions contains per-resource versions for all resource types
type ResourceVersions struct {
	Clusters  []ResourceVersion `json:"clusters"`
	Listeners []ResourceVersion `json:"listeners"`
	Routes    []ResourceVersion `json:"routes"`
	Secrets   []ResourceVersion `json:"secrets"`
}

// GetResourceVersions returns hash-based versions for each individual resource in the snapshot.
// This can be used to compare with Envoy's config_dump to detect unsynchronized resources.
func (c *SnapshotCache) GetResourceVersions(nodeID string) (*ResourceVersions, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot, err := c.SnapshotCache.GetSnapshot(nodeID)
	if err != nil {
		return nil, err
	}

	result := &ResourceVersions{
		Clusters:  computeResourceVersions(snapshot.GetResources(resourcev3.ClusterType)),
		Listeners: computeResourceVersions(snapshot.GetResources(resourcev3.ListenerType)),
		Routes:    computeResourceVersions(snapshot.GetResources(resourcev3.RouteType)),
		Secrets:   computeResourceVersions(snapshot.GetResources(resourcev3.SecretType)),
	}

	return result, nil
}

// computeResourceVersions computes a hash for each resource in the map
func computeResourceVersions(resources map[string]types.Resource) []ResourceVersion {
	if len(resources) == 0 {
		return nil
	}

	versions := make([]ResourceVersion, 0, len(resources))
	for name, res := range resources {
		versions = append(versions, ResourceVersion{
			Name:    name,
			Version: computeResourceHash(res),
		})
	}
	return versions
}

// computeResourceHash computes a SHA256 hash of a protobuf message
func computeResourceHash(msg types.Resource) string {
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
	if err != nil {
		return "error"
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
