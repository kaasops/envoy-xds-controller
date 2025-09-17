# MainBuilder Developer Usage Guide

This document provides practical examples and guidance for using the MainBuilder component in the Envoy XDS Controller project.

## Table of Contents

1. [Basic Usage](#basic-usage)
2. [Feature Flag Control](#feature-flag-control)
3. [Advanced Configuration](#advanced-configuration)
4. [Common Patterns](#common-patterns)
5. [Error Handling](#error-handling)
6. [Performance Considerations](#performance-considerations)
7. [Monitoring](#monitoring)

## Basic Usage

### Creating a ResourceBuilder with MainBuilder Enabled

```go
// Create a ResourceBuilder with MainBuilder enabled by default
import (
    "github.com/kaasops/envoy-xds-controller/internal/store"
    "github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder_v2"
)

func createResourceBuilder(s *store.Store) *resbuilder_v2.ResourceBuilder {
    // Create a ResourceBuilder instance
    rb := resbuilder_v2.NewResourceBuilder(s)
    
    // Enable MainBuilder
    rb.EnableMainBuilder(true)
    
    return rb
}
```

### Building Resources for a VirtualService

```go
// Build resources for a VirtualService
func buildResources(rb *resbuilder_v2.ResourceBuilder, vs *v1alpha1.VirtualService) (*resbuilder_v2.Resources, error) {
    // The BuildResources method automatically uses MainBuilder if enabled
    resources, err := rb.BuildResources(vs)
    if err != nil {
        return nil, fmt.Errorf("failed to build resources: %w", err)
    }
    
    return resources, nil
}
```

## Feature Flag Control

### Environment Variable Configuration

You can control the MainBuilder through environment variables:

```bash
# Enable MainBuilder for all requests
export ENABLE_MAIN_BUILDER=true

# Use MainBuilder for 50% of requests (gradual rollout)
export MAIN_BUILDER_PERCENTAGE=50

# Disable MainBuilder completely
export ENABLE_MAIN_BUILDER=false
```

### Programmatic Control

```go
// Enable or disable MainBuilder programmatically
func configureResourceBuilder(rb *resbuilder_v2.ResourceBuilder, useMainBuilder bool) {
    rb.EnableMainBuilder(useMainBuilder)
}

// Update feature flags from environment variables
func refreshFeatureFlags(rb *resbuilder_v2.ResourceBuilder) {
    rb.UpdateFeatureFlags()
}
```

### A/B Testing

```go
// Create two builders for A/B testing
func setupABTest(store *store.Store) (original, new *resbuilder_v2.ResourceBuilder) {
    // Original implementation
    original = resbuilder_v2.NewResourceBuilder(store)
    original.EnableMainBuilder(false)
    
    // New implementation with MainBuilder
    new = resbuilder_v2.NewResourceBuilder(store)
    new.EnableMainBuilder(true)
    
    return original, new
}
```

## Advanced Configuration

### Custom Components

You can create and configure custom components:

```go
// Create a ResourceBuilder with custom components
func createCustomBuilder(store *store.Store) *resbuilder_v2.ResourceBuilder {
    // Create builder with default components
    rb := resbuilder_v2.NewResourceBuilder(store)
    
    // Replace specific components with custom implementations
    customHTTPFilterBuilder := filters.NewCustomHTTPFilterBuilder(store)
    customTLSBuilder := tls.NewCustomTLSBuilder(store)
    
    // Update ResourceBuilder
    resbuilder_v2.UpdateResourceBuilderComponents(rb, map[string]interface{}{
        "httpFilterBuilder": customHTTPFilterBuilder,
        "tlsBuilder": customTLSBuilder,
    })
    
    return rb
}
```

### Cache Configuration

```go
// Configure the MainBuilder cache (requires access to internal implementation)
func configureCacheSettings(cache *resbuilder_v2.ResourcesCache) {
    // Set Time-To-Live
    cache.SetTTL(10 * time.Minute)
    
    // Set maximum cache size
    cache.SetMaxSize(200)
}
```

## Common Patterns

### Building Resources from Multiple Virtual Services

```go
// Build resources for multiple VirtualServices
func buildMultipleResources(rb *resbuilder_v2.ResourceBuilder, virtualServices []*v1alpha1.VirtualService) map[string]*resbuilder_v2.Resources {
    results := make(map[string]*resbuilder_v2.Resources)
    
    for _, vs := range virtualServices {
        key := fmt.Sprintf("%s/%s", vs.Namespace, vs.Name)
        
        resources, err := rb.BuildResources(vs)
        if err != nil {
            log.Printf("Error building resources for %s: %v", key, err)
            continue
        }
        
        results[key] = resources
    }
    
    return results
}
```

### Handling Updates

```go
// Handle updates to a VirtualService
func handleVirtualServiceUpdate(rb *resbuilder_v2.ResourceBuilder, oldVS, newVS *v1alpha1.VirtualService) (*resbuilder_v2.Resources, error) {
    // Build resources for the updated VirtualService
    resources, err := rb.BuildResources(newVS)
    if err != nil {
        return nil, fmt.Errorf("failed to build resources for updated VirtualService: %w", err)
    }
    
    return resources, nil
}
```

## Error Handling

### Handling Common Errors

```go
// Handle common errors when building resources
func buildResourcesWithErrorHandling(rb *resbuilder_v2.ResourceBuilder, vs *v1alpha1.VirtualService) (*resbuilder_v2.Resources, error) {
    resources, err := rb.BuildResources(vs)
    
    if err != nil {
        // Check for specific error types
        if strings.Contains(err.Error(), "listener not found") {
            return nil, fmt.Errorf("listener not found for VirtualService %s/%s: %w", vs.Namespace, vs.Name, err)
        }
        
        if strings.Contains(err.Error(), "cluster not found") {
            return nil, fmt.Errorf("cluster not found for VirtualService %s/%s: %w", vs.Namespace, vs.Name, err)
        }
        
        // Generic error
        return nil, fmt.Errorf("failed to build resources for VirtualService %s/%s: %w", vs.Namespace, vs.Name, err)
    }
    
    return resources, nil
}
```

### Graceful Degradation

```go
// Attempt to build resources with graceful degradation
func buildResourcesWithFallback(rb *resbuilder_v2.ResourceBuilder, vs *v1alpha1.VirtualService) (*resbuilder_v2.Resources, error) {
    // Try with MainBuilder first
    rb.EnableMainBuilder(true)
    resources, err := rb.BuildResources(vs)
    
    if err != nil {
        // Log the error
        log.Printf("Error with MainBuilder for VirtualService %s/%s: %v", vs.Namespace, vs.Name, err)
        
        // Fall back to original implementation
        rb.EnableMainBuilder(false)
        resources, err = rb.BuildResources(vs)
        if err != nil {
            return nil, fmt.Errorf("both implementations failed for VirtualService %s/%s: %w", vs.Namespace, vs.Name, err)
        }
        
        log.Printf("Used fallback implementation for VirtualService %s/%s", vs.Namespace, vs.Name)
    }
    
    return resources, nil
}
```

## Performance Considerations

### Optimizing for High Throughput

```go
// Configure ResourceBuilder for high throughput
func configureForHighThroughput(rb *resbuilder_v2.ResourceBuilder) {
    // Enable MainBuilder for performance benefits
    rb.EnableMainBuilder(true)
    
    // Configure cache size (if exposed)
    if cache := rb.GetCache(); cache != nil {
        cache.SetMaxSize(500) // Larger cache for high throughput
        cache.SetTTL(30 * time.Minute) // Longer TTL for stable configs
    }
}
```

### Handling Many VirtualServices

```go
// Process multiple VirtualServices in parallel
func processVirtualServicesParallel(rb *resbuilder_v2.ResourceBuilder, virtualServices []*v1alpha1.VirtualService) map[string]*resbuilder_v2.Resources {
    results := make(map[string]*resbuilder_v2.Resources)
    resultMu := sync.Mutex{}
    
    // Create a work pool
    workers := 4
    workChan := make(chan *v1alpha1.VirtualService, len(virtualServices))
    
    // Send all VirtualServices to the work channel
    for _, vs := range virtualServices {
        workChan <- vs
    }
    close(workChan)
    
    // Create worker goroutines
    var wg sync.WaitGroup
    wg.Add(workers)
    
    for w := 0; w < workers; w++ {
        go func() {
            defer wg.Done()
            
            for vs := range workChan {
                key := fmt.Sprintf("%s/%s", vs.Namespace, vs.Name)
                
                resources, err := rb.BuildResources(vs)
                if err != nil {
                    log.Printf("Error building resources for %s: %v", key, err)
                    continue
                }
                
                resultMu.Lock()
                results[key] = resources
                resultMu.Unlock()
            }
        }()
    }
    
    wg.Wait()
    return results
}
```

## Monitoring

### Recording Custom Metrics

```go
// Record custom metrics when building resources
func buildResourcesWithMetrics(rb *resbuilder_v2.ResourceBuilder, vs *v1alpha1.VirtualService) (*resbuilder_v2.Resources, error) {
    startTime := time.Now()
    
    // Build resources
    resources, err := rb.BuildResources(vs)
    
    // Record metrics
    duration := time.Since(startTime).Seconds()
    utils.RecordBuildDuration("custom_component", "build_resources", duration)
    
    if err != nil {
        // Record error metrics
        metrics.ResourceBuildErrors.WithLabelValues(vs.Namespace, vs.Name).Inc()
        return nil, err
    }
    
    // Record success metrics
    metrics.ResourcesBuilt.WithLabelValues(vs.Namespace, vs.Name).Inc()
    
    // Record resource counts
    if resources != nil {
        metrics.FilterChainsCreated.Add(float64(len(resources.FilterChain)))
        metrics.ClustersCreated.Add(float64(len(resources.Clusters)))
    }
    
    return resources, nil
}
```

### Prometheus Integration

```go
// Register custom metrics for MainBuilder
func registerMainBuilderMetrics() {
    // Register custom metrics
    prometheus.MustRegister(metrics.ResourceBuildErrors)
    prometheus.MustRegister(metrics.ResourcesBuilt)
    prometheus.MustRegister(metrics.FilterChainsCreated)
    prometheus.MustRegister(metrics.ClustersCreated)
}
```