package logging

import (
	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/filters"
)

// Builder handles the construction of access log configurations
type Builder struct {
	filtersBuilder *filters.Builder
}

// NewBuilder creates a new access log builder
func NewBuilder(store *store.Store) *Builder {
	return &Builder{
		filtersBuilder: filters.NewBuilder(store),
	}
}

// BuildAccessLogConfigs builds access log configurations from VirtualService
// This implementation delegates to the existing filters.Builder implementation
func (b *Builder) BuildAccessLogConfigs(vs *v1alpha1.VirtualService) ([]*accesslogv3.AccessLog, error) {
	return b.filtersBuilder.BuildAccessLogConfigs(vs)
}
