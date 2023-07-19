package cache

import (
	"context"
	"errors"
	"strconv"
	"strings"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	// https://github.com/envoyproxy/go-control-plane/issues/390
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/tls_inspector/v3"
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
}

func New() Cache {
	return Cache{
		SnapshotCache: cachev3.NewSnapshotCache(true, cachev3.IDHash{}, nil),
	}
}

func (c *Cache) Update(nodeID string, resource types.Resource, resourceName string) error {
	if resourceName == "" {
		resourceName = cachev3.GetResourceName(resource)
	}

	if resourceName == "" {
		return ErrEmptyResourceName
	}

	resourceType := getResourceType(resource)

	if resourceType == "" {
		return ErrUnknownResourceType
	}

	// Get all nodeID resources indexed by type
	resources, version, err := c.getAll(nodeID)

	if err != nil {
		return nil
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

	return nil
}

func (c *Cache) Delete(nodeID string, resource types.Resource, resourceName string) error {
	if resourceName == "" {
		resourceName = cachev3.GetResourceName(resource)
	}

	if resourceName == "" {
		return ErrEmptyResourceName
	}

	resourceType := getResourceType(resource)

	if resourceType == "" {
		return ErrUnknownResourceType
	}

	// Get all nodeID resources indexed by type
	resources, version, err := c.getAll(nodeID)

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

func (c *Cache) getAll(nodeID string) (map[resourcev3.Type][]types.Resource, int, error) {
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

	if err := snapshot.Consistent(); err != nil {
		return err
	}

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
	case *routev3.Route:
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
