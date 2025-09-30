package updater

import (
	"fmt"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// buildResourcesAdapter routes resource building to either the old or new implementation
// based on feature flags configuration
func buildResourcesAdapter(vs *v1alpha1.VirtualService, store store.Store) (*resbuilder.Resources, error) {
	logger := log.Log.WithValues("virtualservice", vs.Name, "namespace", vs.Namespace)

	// Get feature flags configuration
	flags := config.GetFeatureFlags()

	// Create namespaced name for consistent hashing
	namespacedName := fmt.Sprintf("%s/%s", vs.Namespace, vs.Name)

	// Use the existing rollout logic with consistent hashing
	useMainBuilder := config.ShouldUseMainBuilder(flags, namespacedName)

	if useMainBuilder {
		logger.V(2).Info("Using MainBuilder implementation",
			"EnableMainBuilder", flags.EnableMainBuilder,
			"MainBuilderPercentage", flags.MainBuilderPercentage)
		return buildResourcesWithMainBuilder(vs, store)
	}

	// Fall back to the original implementation
	logger.V(2).Info("Using legacy resbuilder implementation")
	return resbuilder.BuildResources(vs, store)
}

// buildResourcesWithMainBuilder builds resources using the new MainBuilder implementation
// but returns them in the format expected by the old implementation
func buildResourcesWithMainBuilder(vs *v1alpha1.VirtualService, store store.Store) (*resbuilder.Resources, error) {
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
