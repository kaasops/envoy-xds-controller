package updater

import (
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
)

// buildResourcesAdapter builds Envoy resources for a VirtualService
func buildResourcesAdapter(vs *v1alpha1.VirtualService, store store.Store) (*resbuilder.Resources, error) {
	return resbuilder.BuildResources(vs, store)
}
