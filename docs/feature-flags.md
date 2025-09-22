# Feature Flags for ResBuilder V2

## Overview

The Envoy XDS Controller supports gradual rollout of the new ResBuilder V2 implementation using feature flags. This allows safe migration from the legacy implementation to the optimized MainBuilder.

## Configuration

Feature flags are controlled through environment variables:

### ENABLE_MAIN_BUILDER

Controls whether to use the new MainBuilder implementation.

- **Type**: Boolean
- **Default**: `false`
- **Values**: `true`, `false`, `yes`, `no`, `1`, `0`

Example:
```bash
export ENABLE_MAIN_BUILDER=true
```

### MAIN_BUILDER_PERCENTAGE

Controls the percentage of VirtualServices that use the MainBuilder implementation.

- **Type**: Integer
- **Default**: `0`
- **Range**: `0-100`
- **Note**: Values outside the range are automatically clamped
- **Important**: Uses consistent hashing based on VirtualService namespace/name, ensuring the same VirtualService always uses the same implementation

Example:
```bash
export ENABLE_MAIN_BUILDER=true
export MAIN_BUILDER_PERCENTAGE=25  # 25% of VirtualServices use MainBuilder
```

## Rollout Strategies

### 1. Disabled (Default)
```bash
# No environment variables set, or:
export ENABLE_MAIN_BUILDER=false
```
All requests use the legacy implementation.

### 2. Full Enable
```bash
export ENABLE_MAIN_BUILDER=true
# MAIN_BUILDER_PERCENTAGE not set or set to 0 or 100
```
All requests use the new MainBuilder implementation.

### 3. Gradual Rollout
```bash
export ENABLE_MAIN_BUILDER=true
export MAIN_BUILDER_PERCENTAGE=10  # Start with 10%
```
Routes the specified percentage of VirtualServices to MainBuilder using consistent hashing. Each VirtualService will consistently use the same implementation.

## Production Rollout Example

### Week 1: Initial Testing
```bash
# Day 1-2: 5% traffic
export ENABLE_MAIN_BUILDER=true
export MAIN_BUILDER_PERCENTAGE=5

# Day 3-7: 10% traffic
export MAIN_BUILDER_PERCENTAGE=10
```

### Week 2: Increase Confidence
```bash
# Day 1-3: 25% traffic
export MAIN_BUILDER_PERCENTAGE=25

# Day 4-7: 50% traffic
export MAIN_BUILDER_PERCENTAGE=50
```

### Week 3: Majority Traffic
```bash
# 75% traffic
export MAIN_BUILDER_PERCENTAGE=75
```

### Week 4: Full Migration
```bash
# 100% traffic
export MAIN_BUILDER_PERCENTAGE=100
# Or simply:
unset MAIN_BUILDER_PERCENTAGE  # Defaults to 100% when enabled
```

## Monitoring

When verbose logging is enabled (log level 2), the controller logs which implementation is used:

```
INFO Using MainBuilder implementation {"virtualservice": "my-vs", "namespace": "default", "EnableMainBuilder": true, "MainBuilderPercentage": 25}
```

or

```
INFO Using legacy resbuilder implementation {"virtualservice": "my-vs", "namespace": "default"}
```

## Rollback

To quickly rollback to the legacy implementation:

```bash
export ENABLE_MAIN_BUILDER=false
# Or simply unset the variable:
unset ENABLE_MAIN_BUILDER
```

## Testing

To verify the current configuration:

```bash
# Check current settings
echo "ENABLE_MAIN_BUILDER: ${ENABLE_MAIN_BUILDER:-not set}"
echo "MAIN_BUILDER_PERCENTAGE: ${MAIN_BUILDER_PERCENTAGE:-not set}"

# Test with specific settings
ENABLE_MAIN_BUILDER=true MAIN_BUILDER_PERCENTAGE=50 kubectl logs -n envoy-system envoy-xds-controller -f | grep "Using.*Builder"
```

## Best Practices

1. **Start Small**: Begin with 5-10% traffic to detect any issues early
2. **Monitor Metrics**: Watch for changes in performance, errors, or resource usage
3. **Gradual Increase**: Increase percentage gradually, monitoring at each step
4. **Document Changes**: Log when percentage changes are made for troubleshooting
5. **Have a Rollback Plan**: Know how to quickly disable the new implementation if issues arise

## Troubleshooting

### Feature flags not taking effect

1. Verify environment variables are set correctly:
   ```bash
   kubectl exec -n envoy-system deployment/envoy-xds-controller -- env | grep MAIN_BUILDER
   ```

2. Check controller logs for which implementation is being used

3. Ensure the controller was restarted after setting environment variables

### Consistent Hashing Behavior

The implementation uses consistent hashing to ensure:
- The same VirtualService always uses the same implementation for a given percentage setting
- Distribution is deterministic and repeatable
- Changes to the percentage will affect a predictable subset of VirtualServices
- This makes debugging easier as behavior is consistent per VirtualService