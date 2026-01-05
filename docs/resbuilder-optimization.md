# ResBuilder Architecture

## Overview

This document describes the architecture of the `internal/xds/resbuilder` package in the Envoy XDS Controller. The package is responsible for building Envoy resources (listeners, routes, clusters, secrets) from VirtualService configurations.

## Architecture

The implementation follows a modular component-based architecture:

```
resbuilder/
├── main_builder/      # Core orchestration component
├── clusters/          # Cluster building and extraction (implements ClusterExtractor)
├── filters/           # HTTP filter construction (implements HTTPFilterBuilder, AccessLogBuilder)
├── filter_chains/     # Filter chain building (implements FilterChainBuilder)
├── routes/            # Route configuration (implements RoutingBuilder)
├── secrets/           # TLS secret management (implements TLSBuilder)
├── utils/             # Shared utilities and object pools
├── interfaces/        # Component contracts for dependency injection
├── builder.go         # ResourceBuilder - main entry point
└── init.go            # Component initialization
```

### Key Components

1. **Builder**: Main orchestrator that coordinates all components
2. **HTTPFilterBuilder**: Constructs HTTP filters for virtual services
3. **FilterChainBuilder**: Builds listener filter chains
4. **RoutingBuilder**: Creates route configurations
5. **ClusterExtractor**: Extracts cluster references from various sources
6. **TLSBuilder**: Manages TLS configuration and secrets

## Performance Optimizations

### Optimization Techniques

1. **Advanced LRU Caching**
    - TTL-based expiration
    - Configurable cache size
    - Cache prewarming capabilities

2. **Object Pooling**
    - Reusable slices for clusters, strings, and HTTP filters
    - Reduced garbage collection pressure

3. **Direct Processing**
    - Eliminated unnecessary JSON marshal/unmarshal cycles
    - Direct struct field access instead of reflection

4. **Memory Management**
    - Pre-allocated slices with capacity estimation
    - Efficient string operations

## Monitoring and Metrics

The implementation includes comprehensive Prometheus metrics:

```
# Cache metrics
envoy_xds_resbuilder_cache_hits_total
envoy_xds_resbuilder_cache_misses_total
envoy_xds_resbuilder_cache_evictions_total
envoy_xds_resbuilder_cache_size

# Performance metrics
envoy_xds_resbuilder_build_duration_seconds
envoy_xds_resbuilder_component_duration_seconds

# Resource metrics
envoy_xds_resbuilder_resources_created_total
envoy_xds_resbuilder_memory_usage_bytes
```

## Testing

### Running Tests

```bash
# Run unit tests
go test ./internal/xds/resbuilder/...

# Run benchmarks
go test ./internal/xds/resbuilder/... -bench=. -benchmem
```

## API Usage

```go
// Build resources for a VirtualService
resources, err := resbuilder.BuildResources(virtualService, store)
```

## Known Issues and Limitations

1. **TLS Proto Registration**: The TLS configuration test is temporarily disabled due to proto type registration issues for `TlsInspector`
2. **Cache Eviction**: Currently uses simple "clear all" strategy when full

## Future Improvements

1. **Performance**
    - Implement sophisticated LRU eviction strategy
    - Expand object pooling coverage
    - Profile and optimize hot paths

2. **Code Quality**
    - Further reduce cyclomatic complexity
    - Increase test coverage to 90%+
    - Add performance regression tests
