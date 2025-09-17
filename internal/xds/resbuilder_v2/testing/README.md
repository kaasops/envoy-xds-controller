# ResBuilder Testing Framework

This directory contains the testing framework for comparing and benchmarking the original and MainBuilder implementations of ResourceBuilder.

## Comparison Testing

The comparison testing framework allows for testing that both implementations produce equivalent results for the same input. This helps ensure that the new MainBuilder implementation is compatible with the original implementation.

### Running Comparison Tests

```bash
# Run all comparison tests
go test -v ./internal/xds/resbuilder_v2/testing -run TestCompareImplementations

# Run a specific test
go test -v ./internal/xds/resbuilder_v2/testing -run TestBasicHTTPRouting
```

## Performance Benchmarking

The benchmark suite allows for comparing the performance of both implementations across different VirtualService configurations.

### Running Benchmarks

```bash
# Run all benchmarks
go test -v ./internal/xds/resbuilder_v2/testing -bench=BenchmarkResourceBuilder -benchmem

# Run specific benchmarks
go test -v ./internal/xds/resbuilder_v2/testing -bench=BenchmarkResourceBuilder/Original-BasicHTTPRouting -benchmem
go test -v ./internal/xds/resbuilder_v2/testing -bench=BenchmarkResourceBuilder/MainBuilder-BasicHTTPRouting -benchmem
```

### Benchmark Configurations

The benchmark suite includes the following configurations:

1. **BasicHTTPRouting** - A simple VirtualService with basic HTTP routing
2. **TLSConfiguration** - A VirtualService with TLS configuration
3. **RBACConfiguration** - A VirtualService with RBAC policies
4. **ComplexConfiguration** - A VirtualService with multiple features (TLS, RBAC, filters, etc.)

### Analyzing Benchmark Results

The benchmark results include:

- **Operations per second** - Higher is better
- **Time per operation** - Lower is better
- **Bytes per operation** - Lower is better
- **Allocations per operation** - Lower is better

Example output:
```
BenchmarkResourceBuilder/Original-BasicHTTPRouting-8         10000        123456 ns/op        1234 B/op        12 allocs/op
BenchmarkResourceBuilder/MainBuilder-BasicHTTPRouting-8      12000        101234 ns/op        1000 B/op        10 allocs/op
```

This indicates that the MainBuilder implementation is faster (101234 ns/op vs 123456 ns/op), uses less memory (1000 B/op vs 1234 B/op), and has fewer allocations (10 allocs/op vs 12 allocs/op) than the original implementation for the BasicHTTPRouting configuration.

### Comparing Results

To compare results between runs, you can use the `benchstat` tool:

```bash
# Save benchmark results to files
go test -v ./internal/xds/resbuilder_v2/testing -bench=BenchmarkResourceBuilder -benchmem > old.txt
# Make changes to the code
go test -v ./internal/xds/resbuilder_v2/testing -bench=BenchmarkResourceBuilder -benchmem > new.txt
# Compare results
benchstat old.txt new.txt
```

## Feature Flags

The testing framework respects the feature flags set in the environment:

- `ENABLE_MAIN_BUILDER` - Controls whether to use the MainBuilder implementation (true/false)
- `MAIN_BUILDER_PERCENTAGE` - Controls what percentage of requests should use MainBuilder (0-100)

Example:
```bash
# Run tests with MainBuilder enabled
ENABLE_MAIN_BUILDER=true go test -v ./internal/xds/resbuilder_v2/testing

# Run tests with 50% of requests using MainBuilder
MAIN_BUILDER_PERCENTAGE=50 go test -v ./internal/xds/resbuilder_v2/testing
```