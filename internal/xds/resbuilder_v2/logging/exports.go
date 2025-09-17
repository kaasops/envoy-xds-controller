package logging

import (
	"github.com/kaasops/envoy-xds-controller/internal/store"
)

// NewAccessLogBuilder creates a new instance of the AccessLogBuilder interface
func NewAccessLogBuilder(store *store.Store) *Builder {
	return NewBuilder(store)
}