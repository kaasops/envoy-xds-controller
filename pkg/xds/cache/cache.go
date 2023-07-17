package cache

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

var (
	resourceTypes = []string{
		resourcev3.EndpointType,
		resourcev3.ClusterType,
		resourcev3.RouteType,
		resourcev3.ScopedRouteType,
		resourcev3.VirtualHostType,
		resourcev3.ListenerType,
		resourcev3.SecretType,
		resourcev3.ExtensionConfigType,
		resourcev3.RouteType,
	}
	ErrNotSupported = errors.New("not supported type for create or update kubernetes resource")
)

type Cache struct {
	SnapshotCache cachev3.SnapshotCache
}

func New() Cache {
	return Cache{
		SnapshotCache: cachev3.NewSnapshotCache(true, cachev3.IDHash{}, nil),
	}
}

func (c *Cache) Update(nodeID string, resource types.Resource, resourceName string, resourceType string) error {
	version := 0

	// Create map for new snapshot
	resources := make(map[string][]types.Resource, 0)
	for _, t := range resourceTypes {
		// resourceCache := snap.GetResources(t)
		resourceCache, rVersionStr, err := c.getResourceFromCache(t, nodeID)
		if err != nil {
			return err
		}

		// Get max version from resources
		if rVersionStr != "" {
			rVersion, err := strconv.Atoi(rVersionStr)
			if err != nil {
				return err
			}
			if rVersion > version {
				version = rVersion
			}
		}

		// if our resource - update
		if t == resourceType {
			resourceCache[resourceName] = resource
		}

		res := make([]types.Resource, 0)

		for _, r := range resourceCache {
			res = append(res, r)
		}

		resources[t] = res
	}

	// Increment version
	version++
	snapshot, err := cachev3.NewSnapshot(strconv.Itoa(version), resources)

	if err != nil {
		return err
	}

	if err := c.SnapshotCache.SetSnapshot(context.Background(), nodeID, snapshot); err != nil {
		return err
	}

	return nil
}

// GetResourceFromCache return
func (c *Cache) getResourceFromCache(resourceType, nodeID string) (map[string]types.Resource, string, error) {
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
