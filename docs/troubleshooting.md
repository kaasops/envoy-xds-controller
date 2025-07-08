# Troubleshooting Guide: Envoy XDS Controller

This document provides guidance for diagnosing and resolving common issues with the Envoy XDS Controller.

## Table of Contents

1. [Common Issues](#common-issues)
2. [Debugging Techniques](#debugging-techniques)
3. [Logs and Monitoring](#logs-and-monitoring)
4. [Known Issues](#known-issues)
5. [Getting Help](#getting-help)

## Common Issues

### Controller Not Starting

**Symptoms:**
- Controller pod is in CrashLoopBackOff state
- Controller logs show startup errors

**Possible Causes and Solutions:**

1. **Invalid Configuration**
   - Check the controller's configuration in the ConfigMap or Helm values
   - Verify that all required environment variables are set correctly
   - Solution: Correct the configuration and redeploy

2. **Missing Permissions**
   - Controller may not have the necessary RBAC permissions
   - Solution: Verify and correct the ClusterRole and ClusterRoleBinding

3. **Webhook Certificate Issues**
   - If using the validating webhook, certificate issues can prevent startup
   - Solution: Check the webhook certificate and secret
   ```bash
   kubectl get secret -n envoy-xds-controller envoy-xds-controller-webhook-cert
   ```

### Envoy Not Receiving Configuration

**Symptoms:**
- Envoy logs show connection issues to xDS server
- Envoy configuration is not updated when CRs change

**Possible Causes and Solutions:**

1. **Network Connectivity**
   - Ensure Envoy can reach the xDS server
   - Check network policies and firewall rules
   - Solution: Verify connectivity with a simple test
   ```bash
   kubectl exec -it <envoy-pod> -- curl -v <xds-server-address>:<port>
   ```

2. **Node ID Mismatch**
   - Envoy's node ID must match what the controller expects
   - Solution: Verify Envoy's node ID configuration and the controller's nodeIds list

3. **TLS Configuration**
   - If TLS is enabled, certificate issues can prevent connections
   - Solution: Check TLS certificates and verify Envoy's TLS configuration

### Custom Resources Not Applied

**Symptoms:**
- Custom resources (CRs) are created but not reflected in Envoy
- Controller logs show validation errors

**Possible Causes and Solutions:**

1. **Validation Errors**
   - CRs may not pass validation
   - Solution: Check controller logs for validation errors and fix the CR definition

2. **Incorrect Node ID**
   - CRs must be associated with the correct node ID
   - Solution: Verify the node ID annotation on the CR
   ```yaml
   metadata:
     annotations:
       envoy.kaasops.io/node-id: "node1"
   ```

3. **Controller Not Watching Namespace**
   - Controller may not be watching the namespace where CRs are created
   - Solution: Check the `watchNamespaces` configuration

## Debugging Techniques

### Enabling Debug Logs

To enable debug logs for the controller:

```bash
# For Helm deployment
helm upgrade envoy-xds-controller \
  --namespace envoy-xds-controller \
  --set envs.LOG_LEVEL=debug \
  helm/charts/envoy-xds-controller

# For manual deployment
kubectl set env deployment/envoy-xds-controller -n envoy-xds-controller LOG_LEVEL=debug
```

### Checking Controller Status

```bash
kubectl get pods -n envoy-xds-controller
kubectl describe pod -n envoy-xds-controller <controller-pod>
kubectl logs -n envoy-xds-controller <controller-pod>
```

### Verifying Custom Resources

```bash
# List all VirtualServices
kubectl get virtualservices.envoy.kaasops.io --all-namespaces

# Describe a specific VirtualService
kubectl describe virtualservice.envoy.kaasops.io <name> -n <namespace>
```

### Checking Envoy Configuration

To check Envoy's current configuration:

```bash
# Using Envoy's admin interface
kubectl exec -it <envoy-pod> -- curl localhost:9901/config_dump

# Check specific configuration
kubectl exec -it <envoy-pod> -- curl localhost:9901/clusters
kubectl exec -it <envoy-pod> -- curl localhost:9901/listeners
```

## Logs and Monitoring

### Important Log Patterns

Look for these patterns in the controller logs:

- `ERROR` - Critical errors that need attention
- `Reconciling` - Shows reconciliation of resources
- `Validation failed` - Indicates validation issues with CRs
- `Updated snapshot` - Indicates successful configuration updates

### Prometheus Metrics

The controller exposes Prometheus metrics at `/metrics` endpoint. Key metrics to monitor:

- `controller_runtime_reconcile_total` - Total number of reconciliations
- `controller_runtime_reconcile_errors_total` - Total number of reconciliation errors
- `xds_cache_updates_total` - Number of xDS cache updates
- `xds_cache_update_errors_total` - Number of xDS cache update errors

## Known Issues

### Template Processing Performance

**Issue:** Processing large templates with many substitutions can be slow.

**Workaround:** Break down large templates into smaller ones or reduce the number of substitutions.

### Webhook Timeout

**Issue:** Webhook validation may time out for large resources.

**Workaround:** Increase the webhook timeout in the ValidatingWebhookConfiguration.

```yaml
timeoutSeconds: 30  # Increase from default 10
```

### Multiple Node IDs

**Issue:** When using multiple node IDs, resources may be incorrectly assigned.

**Workaround:** Ensure each resource has the correct node ID annotation and verify the node ID list in the configuration.

## Getting Help

If you encounter issues not covered in this guide:

1. Check the [GitHub Issues](https://github.com/your-org/envoy-xds-controller/issues) for similar problems
2. Search the project documentation
3. Open a new issue with:
   - Detailed description of the problem
   - Steps to reproduce
   - Relevant logs and configuration
   - Environment details (Kubernetes version, Envoy version, etc.)