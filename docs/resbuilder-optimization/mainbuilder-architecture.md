# MainBuilder Architecture Documentation

## Overview

The MainBuilder component is a high-performance, modular implementation of the resource building functionality in Envoy XDS Controller. It serves as an optimized replacement for the original ResourceBuilder implementation, with a focus on:

- Performance optimization through efficient caching
- Memory usage reduction with object pooling
- Modular design with clear separation of concerns
- Comprehensive metrics for monitoring
- Gradual rollout capability through feature flags

## Component Architecture

The MainBuilder architecture is organized into the following key components:

```
┌────────────────────────────────────────────────────────────────────────────┐
│                            ResourceBuilder                                 │
│                                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐  │
│  │                          MainBuilder                                │  │
│  │                                                                     │  │
│  │  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────────┐ │  │
│  │  │ HTTPFilter    │  │ FilterChain  │  │ Routing                  │ │  │
│  │  │ Builder       │  │ Builder      │  │ Builder                  │ │  │
│  │  └───────────────┘  └──────────────┘  └──────────────────────────┘ │  │
│  │                                                                     │  │
│  │  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────────┐ │  │
│  │  │ AccessLog     │  │ TLS          │  │ Cluster                  │ │  │
│  │  │ Builder       │  │ Builder      │  │ Extractor                │ │  │
│  │  └───────────────┘  └──────────────┘  └──────────────────────────┘ │  │
│  │                                                                     │  │
│  │  ┌─────────────────────────────────────────────────────────────┐   │  │
│  │  │                 Advanced LRU Cache                           │   │  │
│  │  └─────────────────────────────────────────────────────────────┘   │  │
│  │                                                                     │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### Key Components

1. **ResourceBuilder**: The main entry point that provides a unified interface for building Envoy resources. It can use either the original implementation or the new MainBuilder implementation based on feature flags.

2. **MainBuilder**: Coordinates the resource building process, delegating to specialized components and applying caching for performance optimization.

3. **Component Builders**:
   - **HTTPFilterBuilder**: Builds HTTP filters for the virtual service
   - **FilterChainBuilder**: Builds filter chains for the listener
   - **RoutingBuilder**: Builds routing configuration (virtual hosts, routes)
   - **AccessLogBuilder**: Builds access log configuration
   - **TLSBuilder**: Builds TLS/security configuration
   - **ClusterExtractor**: Extracts clusters from various sources

4. **Advanced LRU Cache**: Provides efficient caching with TTL, LRU eviction, and prewarming capabilities.

5. **Adapters**: (not shown in diagram) Bridge between MainBuilder components and existing ResourceBuilder components.

6. **Interfaces**: Define the contract between components, enabling modularity and testability.

## Component Interactions

The process of building resources follows these steps:

1. **ResourceBuilder.BuildResources** is called with a VirtualService
2. Based on feature flags, it either:
   - Uses the original implementation
   - Delegates to MainBuilder via `buildResourcesWithMainBuilder`
3. **MainBuilder.BuildResources**:
   - Checks the cache for previously built resources
   - If not found, builds resources from scratch:
     1. Applies VirtualService template if specified
     2. Builds listener
     3. Handles existing filter chains or builds new ones
     4. Builds virtual host and route configuration
     5. Builds filter chains
     6. Extracts clusters
     7. Builds secrets if needed
   - Records metrics
   - Caches the result
   - Returns the built resources

## Caching Strategy

The MainBuilder implements an advanced caching strategy:

1. **Cache Key Generation**: Creates a unique key based on the VirtualService's properties
2. **TTL-based Expiration**: Automatically expires cache entries after a configurable period
3. **LRU Eviction**: Uses Least Recently Used algorithm to remove entries when the cache is full
4. **Prewarming**: Can preload frequently accessed resources into the cache
5. **Metrics**: Tracks cache hits, misses, and evictions for monitoring

## Memory Optimization

Memory usage is optimized through:

1. **Object Pooling**: Reuses common objects (slices, clusters, HTTP filters) to reduce allocations
2. **Efficient Cache Eviction**: Removes expired entries first, then uses LRU for optimal memory usage
3. **Streaming Processing**: Where possible, processes data in a streaming fashion rather than building intermediate structures

## Metrics and Monitoring

The MainBuilder provides comprehensive metrics for monitoring:

1. **Cache Metrics**:
   - Hits/misses
   - Size
   - Evictions
   - Item age

2. **Performance Metrics**:
   - Build duration
   - Component-specific timing
   - Resource processing time

3. **Memory Metrics**:
   - Object pool usage
   - Resource counts
   - Memory usage estimates
   - Resource cardinality

4. **Feature Flag Metrics**:
   - Usage counts

## Feature Flags

The MainBuilder supports gradual rollout through feature flags:

1. **ENABLE_MAIN_BUILDER**: Environment variable to fully enable/disable MainBuilder
2. **MAIN_BUILDER_PERCENTAGE**: Controls the percentage of requests that use MainBuilder
3. **Programmatic Control**: `EnableMainBuilder(bool)` method for code-based control

## Error Handling

The MainBuilder implements robust error handling:

1. **Contextual Errors**: Adds context to errors for easier debugging
2. **Panic Recovery**: Recovers from panics to prevent service disruption
3. **Validation**: Validates inputs and outputs to catch issues early
4. **Null Safety**: Checks for nil values at critical points

## Extension Points

The MainBuilder is designed for extensibility:

1. **Component Interfaces**: Allow alternative implementations of each component
2. **Adapter Pattern**: Facilitates integration with existing components
3. **Feature Flags**: Enable gradual adoption and A/B testing