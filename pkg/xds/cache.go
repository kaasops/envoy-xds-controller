package xds

import (
	"context"
	"strings"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

type Cache interface {
	GetSnapshotCache() cachev3.SnapshotCache
	GetResource(ResourceType string, nodeID string) (map[string]types.Resource, error)
	SetSnaphot(ctx context.Context, nodeID string, snap cachev3.ResourceSnapshot)
}

type cache struct {
	xDSCache cachev3.SnapshotCache
}

func NewCache() Cache {
	return &cache{
		xDSCache: cachev3.NewSnapshotCache(false, cachev3.IDHash{}, nil),
	}
}

func (c *cache) GetSnapshotCache() cachev3.SnapshotCache {
	return c.xDSCache
}

func (c *cache) GetResource(ResourceType string, nodeID string) (map[string]types.Resource, error) {
	snap, err := c.xDSCache.GetSnapshot(nodeID)
	if err == nil {
		return snap.GetResources(ResourceType), nil
	}
	if strings.Contains(err.Error(), "no snapshot found for node") {
		return map[string]types.Resource{}, nil
	}
	return nil, err
}

func (c *cache) SetSnaphot(ctx context.Context, nodeID string, snap cachev3.ResourceSnapshot) {
	c.xDSCache.SetSnapshot(ctx, nodeID, snap)
}
