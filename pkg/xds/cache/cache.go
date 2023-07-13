package cache

import (
	"strings"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

type Cache struct {
	SnapshotCache cachev3.SnapshotCache
}

func New() Cache {
	return Cache{
		SnapshotCache: cachev3.NewSnapshotCache(true, cachev3.IDHash{}, nil),
	}
}

// GetResourceFromCache return
func (c *Cache) GetResourceFromCache(resourceType, nodeID string) (map[string]types.Resource, string, error) {
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
