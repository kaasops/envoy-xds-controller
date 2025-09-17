# MainBuilder Production Rollout Plan

This document outlines the strategy for safely rolling out the MainBuilder component to production environments.

## Table of Contents

1. [Gradual Rollout Strategy](#gradual-rollout-strategy)
2. [Monitoring Plan](#monitoring-plan)
3. [Rollback Procedures](#rollback-procedures)
4. [Feature Flag Configuration](#feature-flag-configuration)
5. [Dashboards Setup](#dashboards-setup)
6. [Success Criteria](#success-criteria)
7. [Timeline](#timeline)

## Gradual Rollout Strategy

The MainBuilder component includes built-in feature flags that enable a controlled, gradual rollout to production. Follow these steps for a safe rollout:

### Phase 1: Initial Deployment (Week 1)

1. Deploy the new code with MainBuilder disabled by default
2. Enable MainBuilder for 0% of traffic (`MAIN_BUILDER_PERCENTAGE=0`)
3. Verify deployment success and baseline metrics
4. Begin internal testing with specific test VirtualServices

### Phase 2: Canary Testing (Week 2)

1. Enable MainBuilder for 5% of traffic (`MAIN_BUILDER_PERCENTAGE=5`)
2. Monitor for 24 hours
3. Review metrics and logs for any issues
4. If stable, increase to 10% (`MAIN_BUILDER_PERCENTAGE=10`)
5. Monitor for 24 hours
6. Review metrics and logs again

### Phase 3: Expanded Rollout (Week 3)

1. If Phase 2 is successful, increase to 25% (`MAIN_BUILDER_PERCENTAGE=25`)
2. Monitor for 24 hours
3. Review metrics and logs
4. If stable, increase to 50% (`MAIN_BUILDER_PERCENTAGE=50`)
5. Monitor for 48 hours
6. Conduct comprehensive review of metrics and performance

### Phase 4: Full Rollout (Week 4)

1. If Phase 3 is successful, increase to 75% (`MAIN_BUILDER_PERCENTAGE=75`)
2. Monitor for 24 hours
3. If stable, increase to 100% (`MAIN_BUILDER_PERCENTAGE=100`)
4. Monitor for 72 hours
5. If stable, consider setting `ENABLE_MAIN_BUILDER=true` to bypass percentage-based routing

## Monitoring Plan

### Key Metrics to Monitor

1. **Build Performance**
   - `envoy_xds_resbuilder_build_duration_seconds` - Compare between original and MainBuilder implementations
   - `envoy_xds_resbuilder_component_duration_seconds` - Monitor individual component performance
   - `envoy_xds_resbuilder_resource_processing_seconds` - Track resource processing time

2. **Error Rates**
   - Error rates for both implementations
   - Compare error types and frequencies
   - Monitor for new error types specific to MainBuilder

3. **Memory Usage**
   - `envoy_xds_resbuilder_memory_usage_bytes` - Track memory consumption
   - `envoy_xds_resbuilder_resource_cardinality` - Monitor resource collection sizes
   - `envoy_xds_resbuilder_cache_size` - Track cache size

4. **Cache Efficiency**
   - `envoy_xds_resbuilder_cache_hits_total` - Monitor cache hit rate
   - `envoy_xds_resbuilder_cache_misses_total` - Monitor cache miss rate
   - `envoy_xds_resbuilder_cache_evictions_total` - Track cache evictions
   - `envoy_xds_resbuilder_cache_item_age_seconds` - Monitor cache item age

5. **Feature Flag Usage**
   - `envoy_xds_resbuilder_feature_flag_usage_total` - Track feature flag usage

### Alerting Thresholds

Set up alerts for the following conditions:

1. **Error Rate Increase**
   - Alert if error rate with MainBuilder is 10% higher than the original implementation
   - Alert if any new error types appear with MainBuilder

2. **Performance Degradation**
   - Alert if average build duration with MainBuilder is 20% higher than the original implementation
   - Alert if p95 build duration with MainBuilder is 30% higher than the original implementation

3. **Memory Usage**
   - Alert if memory usage increases by more than 30% compared to baseline

4. **Cache Issues**
   - Alert if cache hit rate drops below 70%
   - Alert if cache eviction rate exceeds 10% of cache size per hour

## Rollback Procedures

If issues are detected during the rollout, follow these procedures to safely roll back:

### Immediate Rollback

If critical issues are detected (service disruption, significant performance degradation, or high error rates):

1. Disable MainBuilder immediately by setting `ENABLE_MAIN_BUILDER=false`
2. Verify traffic is no longer being routed to MainBuilder
3. Confirm metrics return to baseline
4. Notify the development team and begin investigation

### Gradual Rollback

For less critical issues:

1. Reduce the percentage of traffic to MainBuilder (e.g., from 50% to 25%)
2. Monitor the impact on metrics
3. Continue reducing if issues persist
4. If issues are resolved at a lower percentage, maintain that level while investigating

### Post-Rollback Analysis

After any rollback:

1. Collect and analyze logs, metrics, and error reports
2. Identify the root cause of the issue
3. Create a detailed report with:
   - Description of the issue
   - Impact on the system
   - Root cause analysis
   - Steps taken to resolve
   - Recommendations for fixes
4. Develop a plan to address the issue before the next rollout attempt

## Feature Flag Configuration

The MainBuilder component supports the following feature flags:

### Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `ENABLE_MAIN_BUILDER` | Fully enables or disables MainBuilder | `false` | `ENABLE_MAIN_BUILDER=true` |
| `MAIN_BUILDER_PERCENTAGE` | Percentage of requests that use MainBuilder | `0` | `MAIN_BUILDER_PERCENTAGE=50` |

### Configuration File

Add the following to your configuration file:

```yaml
feature_flags:
  main_builder:
    enabled: false  # Set to true to enable
    percentage: 0   # Percentage of traffic (0-100)
```

### Programmatic Configuration

```go
// Enable MainBuilder
resourceBuilder.EnableMainBuilder(true)

// Update from environment variables
resourceBuilder.UpdateFeatureFlags()
```

### Runtime Updates

The feature flags can be updated at runtime without restarting the service:

1. Update the environment variables
2. Call `resourceBuilder.UpdateFeatureFlags()` (if available)
3. Or update through your configuration management system

## Dashboards Setup

Create the following Grafana dashboards to monitor the rollout:

### 1. MainBuilder Overview Dashboard

**Panels:**
- Build Duration Comparison (Original vs MainBuilder)
- Error Rate Comparison
- Memory Usage Comparison
- Cache Hit Rate
- Feature Flag Status

**Sample Query (Build Duration):**
```
histogram_quantile(0.95, sum by(le, implementation) (rate(envoy_xds_resbuilder_build_duration_seconds_bucket[5m])))
```

### 2. MainBuilder Component Performance Dashboard

**Panels:**
- Component-level Performance Metrics
- Resource Processing Time
- HTTP Filters Build Time
- Filter Chains Build Time
- Routing Configuration Build Time

**Sample Query (Component Duration):**
```
histogram_quantile(0.95, sum by(le, component, method) (rate(envoy_xds_resbuilder_component_duration_seconds_bucket[5m])))
```

### 3. MainBuilder Cache Efficiency Dashboard

**Panels:**
- Cache Hit/Miss Rate
- Cache Size
- Cache Evictions
- Cache Item Age
- Cache Operation Duration

**Sample Query (Cache Hit Rate):**
```
sum(rate(envoy_xds_resbuilder_cache_hits_total[5m])) / (sum(rate(envoy_xds_resbuilder_cache_hits_total[5m])) + sum(rate(envoy_xds_resbuilder_cache_misses_total[5m])))
```

### 4. MainBuilder Rollout Progress Dashboard

**Panels:**
- Feature Flag Status
- Percentage of Traffic to MainBuilder
- Build Duration Trend During Rollout
- Error Rate Trend During Rollout
- Success Rate by Implementation

**Sample Query (Feature Flag Usage):**
```
sum by(flag_name, value) (rate(envoy_xds_resbuilder_feature_flag_usage_total[5m]))
```

## Success Criteria

The rollout will be considered successful if:

1. **Performance Improvement**
   - MainBuilder shows at least 15% lower average build duration
   - MainBuilder shows at least 10% lower memory usage
   - P95 build duration is at least 20% lower

2. **Stability**
   - Error rate remains the same or lower than the original implementation
   - No new error types are introduced
   - No service disruptions during the rollout

3. **Resource Efficiency**
   - Cache hit rate exceeds 75% in production
   - Memory usage remains stable or decreases
   - CPU usage remains stable or decreases

4. **User Impact**
   - No negative impact on API response times
   - No increase in user-facing errors

## Timeline

| Phase | Duration | Start Date | End Date | Key Activities |
|-------|----------|------------|----------|----------------|
| Phase 1: Initial Deployment | 1 week | 2025-09-25 | 2025-10-02 | Deploy with MainBuilder disabled, internal testing |
| Phase 2: Canary Testing | 1 week | 2025-10-02 | 2025-10-09 | 5-10% traffic, monitoring |
| Phase 3: Expanded Rollout | 1 week | 2025-10-09 | 2025-10-16 | 25-50% traffic, comprehensive review |
| Phase 4: Full Rollout | 1 week | 2025-10-16 | 2025-10-23 | 75-100% traffic, final validation |
| Post-Rollout Review | 1 week | 2025-10-23 | 2025-10-30 | Performance analysis, documentation updates |

## Conclusion

This rollout plan provides a structured approach to safely deploying the MainBuilder component to production. By using feature flags for gradual rollout, comprehensive monitoring, and clear rollback procedures, we can minimize risk while realizing the performance benefits of the new implementation.