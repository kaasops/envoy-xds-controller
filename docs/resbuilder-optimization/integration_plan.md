# ResourceBuilder and MainBuilder Integration Plan

## Overview

This document outlines the plan for integrating the new modular MainBuilder with the existing ResourceBuilder in the Envoy XDS Controller's resbuilder_v2 package. The integration involves addressing several architectural challenges, particularly around cyclic dependencies, and ensuring a smooth transition.

## Current State

1. **ResourceBuilder (resbuilder_v2/builder.go)**:
   - Large file (~1000 lines) with multiple responsibilities
   - Contains methods for building various Envoy resources
   - Uses various builders (clusters, filters, routes, secrets)

2. **MainBuilder (resbuilder_v2/main_builder)**:
   - Implements a modular approach to resource building
   - Defines interfaces for each component
   - Uses dependency injection for components
   - Returns a generic interface{} from BuildResources

3. **Component Implementations**:
   - http_filters: Implements HTTP filter building
   - filter_chains: Implements filter chain building
   - routing: Implements route configuration
   - logging: Implements access logging
   - tls: Implements TLS configuration

4. **Adapter Components (resbuilder_v2/adapters)**:
   - Wrap existing builders to implement the required interfaces
   - Provide missing functionality where necessary
   - Enable gradual migration to the new architecture

## Challenges

1. **Cyclic Dependencies**:
   - Adapters import resbuilder_v2 for interfaces
   - resbuilder_v2 would need to import adapters for integration
   - This creates an import cycle that Go doesn't allow

2. **Interface Compatibility**:
   - Some existing builders don't fully implement required interfaces
   - Method signatures don't always match

3. **Resource Type Conversion**:
   - MainBuilder returns interface{} that needs conversion to Resources

## Integration Plan

### 1. Restructure Package Organization

To resolve the cyclic dependency issue, we need to restructure the package organization:

```
internal/xds/resbuilder_v2/
├── interfaces/           # Move interfaces here
│   └── interfaces.go     # All component interfaces
├── adapters/             # Adapter implementations
├── builder.go            # Main ResourceBuilder
├── main_builder/         # MainBuilder implementation
├── [component packages]  # http_filters, filter_chains, etc.
```

#### Steps:
1. Create interfaces package
2. Move all interfaces from resbuilder_v2/interfaces.go to interfaces/interfaces.go
3. Update imports in all files that use these interfaces
4. Update adapter implementations to import from interfaces package

### 2. Update ResourceBuilder

After restructuring, update the ResourceBuilder to use MainBuilder:

```go
// In builder.go
type ResourceBuilder struct {
    store           *store.Store
    clustersBuilder *clusters.Builder
    filtersBuilder  *filters.Builder
    routesBuilder   *routes.Builder
    secretsBuilder  *secrets.Builder
    mainBuilder     interfaces.MainBuilder
    useMainBuilder  bool  // Flag to control which implementation to use
}

// Update constructor
func NewResourceBuilder(store *store.Store) *ResourceBuilder {
    rb := &ResourceBuilder{
        store:           store,
        clustersBuilder: clusters.NewBuilder(store),
        filtersBuilder:  filters.NewBuilder(store),
        routesBuilder:   routes.NewBuilder(store),
        secretsBuilder:  secrets.NewBuilder(store),
        useMainBuilder:  false,
    }
    
    // Initialize adapters and MainBuilder
    httpFilterAdapter := adapters.NewHTTPFilterAdapter(rb.filtersBuilder)
    filterChainAdapter := adapters.NewFilterChainAdapter(filter_chains.NewBuilder(store), store)
    routingAdapter := adapters.NewRoutingAdapter(rb.routesBuilder)
    accessLogAdapter := rb.filtersBuilder // Already implements AccessLogBuilder
    tlsAdapter := adapters.NewTLSAdapter(store)
    clusterExtractorAdapter := adapters.NewClusterExtractorAdapter(rb.clustersBuilder, store)
    
    // Create and configure MainBuilder
    mainBuilder := main_builder.NewMainBuilder(store)
    mainBuilder.SetComponents(
        httpFilterAdapter,
        filterChainAdapter,
        routingAdapter,
        accessLogAdapter,
        tlsAdapter,
        clusterExtractorAdapter,
    )
    
    rb.mainBuilder = mainBuilder
    
    return rb
}

// Update BuildResources to support both implementations
func (rb *ResourceBuilder) BuildResources(vs *v1alpha1.VirtualService) (*Resources, error) {
    if rb.useMainBuilder {
        return rb.buildResourcesWithMainBuilder(vs)
    }
    
    // Original implementation
    // ...existing code...
}

// Add method to use MainBuilder
func (rb *ResourceBuilder) buildResourcesWithMainBuilder(vs *v1alpha1.VirtualService) (*Resources, error) {
    result, err := rb.mainBuilder.BuildResources(vs)
    if err != nil {
        return nil, err
    }
    
    // Type assertion and conversion
    mainResources, ok := result.(*main_builder.Resources)
    if !ok {
        return nil, fmt.Errorf("unexpected result type from MainBuilder")
    }
    
    // Convert to Resources
    resources := &Resources{
        Listener:    mainResources.Listener,
        FilterChain: mainResources.FilterChain,
        RouteConfig: mainResources.RouteConfig,
        Clusters:    mainResources.Clusters,
        Secrets:     mainResources.Secrets,
        UsedSecrets: mainResources.UsedSecrets,
        Domains:     mainResources.Domains,
    }
    
    return resources, nil
}

// Add method to enable/disable MainBuilder
func (rb *ResourceBuilder) EnableMainBuilder(enable bool) {
    rb.useMainBuilder = enable
}
```

### 3. Complete Adapter Implementations

Ensure all adapter implementations are complete and handle all edge cases:

1. **HTTPFilterAdapter**:
   - Implement BuildRBACFilter to match original functionality
   - Ensure proper error handling

2. **FilterChainAdapter**:
   - Complete BuildFilterChainParams implementation
   - Handle all parameters correctly

3. **RoutingAdapter**:
   - Ensure TLS fallback functionality works correctly
   - Verify route ordering and validation

4. **TLSAdapter**:
   - Complete all TLS configuration methods
   - Handle both auto-discovery and secretRef types

5. **ClusterExtractorAdapter**:
   - Complete ExtractClustersFromFilterChains implementation
   - Ensure proper TCP proxy filter handling

### 4. Add Testing Infrastructure

Create comprehensive tests to verify integration:

1. **Unit Tests**:
   - Test each adapter individually
   - Verify interface compliance

2. **Integration Tests**:
   - Compare results from original and MainBuilder implementations
   - Ensure they produce identical resources

3. **Benchmarks**:
   - Measure performance of both implementations
   - Verify performance improvements

### 5. Implement Feature Flags

Add feature flags to control migration:

1. **Environment Variable**:
   - `ENABLE_MAIN_BUILDER=true/false`
   - Control which implementation is used

2. **Gradual Rollout**:
   - Start with a small percentage of traffic
   - Monitor for errors or performance issues
   - Gradually increase percentage

### 6. Documentation and Cleanup

1. **Update Documentation**:
   - Document new architecture
   - Explain migration process
   - Update handoff.md

2. **Code Cleanup**:
   - Mark deprecated methods
   - Add TODO comments for future removal
   - Ensure consistent error handling

3. **Final Review**:
   - Verify no cyclic dependencies
   - Check for memory leaks or performance issues
   - Ensure backward compatibility

## Migration Strategy

1. **Phase 1: Infrastructure Ready** (Current Phase)
   - Complete adapter implementations
   - Restructure packages to avoid cycles
   - Add feature flags
   - No change to production behavior

2. **Phase 2: Testing in Development**
   - Enable MainBuilder in development/testing
   - Run comparison tests
   - Verify resources are identical
   - Measure performance impact

3. **Phase 3: Gradual Production Rollout**
   - Start with 1% of traffic using MainBuilder
   - Monitor errors and performance
   - Gradually increase percentage
   - Revert if issues are found

4. **Phase 4: Complete Migration**
   - 100% of traffic using MainBuilder
   - Remove old implementation
   - Clean up deprecated code
   - Update documentation

## Conclusion

This integration plan provides a path to safely migrate from the current ResourceBuilder implementation to the new modular MainBuilder architecture. By restructuring packages, implementing adapters, and providing feature flags, we can ensure a smooth transition with minimal risk to production systems.

The new architecture will provide better modularity, testability, and maintainability, while preserving or improving performance. The gradual migration strategy allows for safe rollback if issues are encountered during the process.