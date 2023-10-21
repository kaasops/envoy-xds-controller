package cache

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"k8s.io/utils/strings/slices"
)

var (
	resourceTypes = []resourcev3.Type{
		resourcev3.EndpointType,
		resourcev3.ClusterType,
		resourcev3.RouteType,
		resourcev3.ScopedRouteType,
		resourcev3.VirtualHostType,
		resourcev3.ListenerType,
		resourcev3.SecretType,
		resourcev3.ExtensionConfigType,
	}
	ErrUnknownResourceType = errors.New("unknown resource type")
	ErrEmptyResourceName   = errors.New("empty resource name")
)

type Cache struct {
	SnapshotCache cachev3.SnapshotCache

	// List of node ID in cache. Temporary, wait PR - https://github.com/envoyproxy/go-control-plane/pull/769
	nodeIDs []string

	mu sync.Mutex
}

func New() *Cache {
	return &Cache{
		SnapshotCache: cachev3.NewSnapshotCache(true, cachev3.IDHash{}, nil),
	}
}

func (c *Cache) Update(nodeID string, resource types.Resource) error {
	resourceName := cachev3.GetResourceName(resource)

	if resourceName == "" {
		return ErrEmptyResourceName
	}

	resourceType := getResourceType(resource)

	if resourceType == "" {
		return ErrUnknownResourceType
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Get all nodeID resources indexed by type
	resources, version, err := c.GetResources(nodeID)
	if err != nil {
		return err
	}

	// Get resources by type indexed by resource name
	updated, _, err := c.getByType(resourceType, nodeID)
	if err != nil {
		return err
	}

	updated[resourceName] = resource

	resources[resourceType] = toSlice(updated)

	version++

	if err := c.createSnapshot(nodeID, resources, version); err != nil {
		return err
	}

	// Add information about node ID to list
	if !slices.Contains(c.nodeIDs, nodeID) {
		c.nodeIDs = append(c.nodeIDs, nodeID)
	}

	return nil
}

func (c *Cache) Delete(nodeID string, resourceType resourcev3.Type, resourceName string) error {

	if resourceName == "" {
		return ErrEmptyResourceName
	}

	if resourceType == "" {
		return ErrUnknownResourceType
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Get all nodeID resources indexed by type
	resources, version, err := c.GetResources(nodeID)

	if err != nil {
		return nil
	}

	// Get resources by type indexed by resource name
	updated, _, err := c.getByType(resourceType, nodeID)

	if err != nil {
		return err
	}

	delete(updated, resourceName)

	resources[resourceType] = toSlice(updated)

	version++

	if err := c.createSnapshot(nodeID, resources, version); err != nil {
		return err
	}

	return nil
}

func (c *Cache) GetCache() cachev3.SnapshotCache {
	return c.SnapshotCache
}

func (c *Cache) GetResources(nodeID string) (map[resourcev3.Type][]types.Resource, int, error) {
	version := 0
	resources := make(map[resourcev3.Type][]types.Resource, 0)
	for _, t := range resourceTypes {
		resourceCache, rVersionStr, err := c.getByType(t, nodeID)
		if err != nil {
			return nil, 0, err
		}

		// Get max version from resources
		if rVersionStr != "" {
			rVersion, err := strconv.Atoi(rVersionStr)
			if err != nil {
				return nil, 0, err
			}
			if rVersion > version {
				version = rVersion
			}
		}

		res := make([]types.Resource, 0)

		for _, r := range resourceCache {
			res = append(res, r)
		}

		resources[t] = res
	}
	return resources, version, nil
}

func (c *Cache) GetNodeIDs() []string {
	return c.nodeIDs
}

// Wait blocks if:
// 1. There is no Listener for any NodeID.
// 2. The number of resources in the cache has changed within 10 seconds.
func (c *Cache) Wait() error {
	resourceCount, err := c.getResourceCount()
	if err != nil {
		return err
	}

	for {
		time.Sleep(10 * time.Second)
		resourceCountNew, err := c.getResourceCount()
		if err != nil {
			return err
		}
		if resourceCountNew == 0 {
			continue
		}
		if resourceCountNew == resourceCount {
			break
		}
		resourceCount = resourceCountNew
	}

	return nil
}

func (c *Cache) getResourceCount() (int, error) {
	resourceCount := 0

	nodeIDs := c.GetNodeIDs()

	for _, nodeID := range nodeIDs {
		resources, _, err := c.GetResources(nodeID)
		if err != nil {
			return resourceCount, err
		}
		if len(resources[resourcev3.ListenerType]) == 0 {
			continue
		}

		for _, resourceType := range resourceTypes {
			resourceCount += len(resources[resourceType])
		}
	}
	return resourceCount, nil
}

// GetResourceFromCache return
func (c *Cache) getByType(resourceType resourcev3.Type, nodeID string) (map[string]types.Resource, string, error) {
	resSnap, err := c.SnapshotCache.GetSnapshot(nodeID)
	if err == nil {
		if resSnap.GetResources(resourceType) == nil {
			return make(map[string]types.Resource), resSnap.GetVersion(resourceType), nil
		}
		return resSnap.GetResources(resourceType), resSnap.GetVersion(resourceType), nil
	}
	if strings.Contains(err.Error(), "no snapshot found for node") {
		return map[string]types.Resource{}, "", nil
	}
	return nil, "", err
}

func (c *Cache) createSnapshot(nodeID string, resources map[resourcev3.Type][]types.Resource, version int) error {

	snapshot, err := cachev3.NewSnapshot(strconv.Itoa(version), resources)

	if err != nil {
		return err
	}

	// if err := snapshot.Consistent(); err != nil {
	// 	return err
	// }

	if err := c.SnapshotCache.SetSnapshot(context.Background(), nodeID, snapshot); err != nil {
		return err
	}
	return nil
}

// GetResourceName returns the resource name for a valid xDS response type.
func getResourceType(res types.Resource) resourcev3.Type {
	switch res.(type) {
	case *clusterv3.Cluster:
		return resourcev3.ClusterType
	case *routev3.RouteConfiguration:
		return resourcev3.RouteType
	case *routev3.ScopedRouteConfiguration:
		return resourcev3.ScopedRouteType
	case *routev3.VirtualHost:
		return resourcev3.VirtualHostType
	case *listenerv3.Listener:
		return resourcev3.ListenerType
	case *endpointv3.Endpoint:
		return resourcev3.EndpointType
	case *tlsv3.Secret:
		return resourcev3.SecretType
	case *corev3.TypedExtensionConfig:
		return resourcev3.ExtensionConfigType
	default:
		return ""
	}
}

func toSlice(resources map[string]types.Resource) []types.Resource {
	res := make([]types.Resource, 0)
	for _, r := range resources {
		res = append(res, r)
	}
	return res
}

// func (c *Cache) CheckSnapshotCache(nodeID string) error {
// 	snap, err := c.SnapshotCache.GetSnapshot(nodeID)
// 	if err != nil {
// 		return err
// 	}

// 	for _, t := range resourceTypes {
// 		// if t == resourcev3.SecretType {
// 		// 	continue
// 		// }
// 		snapRes := snap.GetResources(t)
// 		fmt.Printf("TYPE: %s\n, Len: %+v", t, len(snapRes))
// 		// for k, v := range snapRes {
// 		// 	fmt.Printf("Name: %s, Resource: %+v", k, v)
// 		// }
// 		fmt.Println()
// 		fmt.Println()
// 	}

// 	return nil
// }
