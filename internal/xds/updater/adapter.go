package updater

import (
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2"
)

// buildResourcesAdapter builds Envoy resources for a VirtualService
func buildResourcesAdapter(vs *v1alpha1.VirtualService, store store.Store) (*resbuilder_v2.Resources, error) {
	return resbuilder_v2.BuildResources(vs, store)
}
