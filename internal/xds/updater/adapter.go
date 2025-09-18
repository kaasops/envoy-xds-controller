package updater

import (
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2"
)

// buildResourcesWithMainBuilder builds resources using the new MainBuilder implementation
// but returns them in the format expected by the old implementation
func buildResourcesWithMainBuilder(vs *v1alpha1.VirtualService, store *store.Store) (*resbuilder.Resources, error) {
	// Create a ResourceBuilder with MainBuilder enabled
	rb := resbuilder_v2.NewResourceBuilder(store)

	// Enable MainBuilder - note that this will be a no-op if already enabled by feature flags
	rb.EnableMainBuilder(true)

	// Build resources using the new implementation
	v2Resources, err := rb.BuildResources(vs)
	if err != nil {
		return nil, err
	}

	// Convert v2Resources to the format expected by the old implementation
	resources := &resbuilder.Resources{
		Listener:    v2Resources.Listener,
		FilterChain: v2Resources.FilterChain,
		RouteConfig: v2Resources.RouteConfig,
		Clusters:    v2Resources.Clusters,
		Secrets:     v2Resources.Secrets,
		UsedSecrets: v2Resources.UsedSecrets,
		Domains:     v2Resources.Domains,
	}

	return resources, nil
}
