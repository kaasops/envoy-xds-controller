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

4. **Envoy Rejected the Configuration (NACK)**
   - Envoy received an update and reported an error, such as a missing file, an unavailable module or an unsupported extension. Instances sharing a node ID respond independently.
   - Open **xDS streams** in the console or query the cache API with its URL in `CACHE_API` and the Envoy node ID in `NODE_ID`. Both views require `cacheAPI.enabled`. With OIDC, set `TOKEN` to your access token and add `-H "Authorization: Bearer $TOKEN"`. The controller also logs `Envoy NACKed xDS response`.
   ```bash
   curl -fsS --get "$CACHE_API/api/v1/clients" --data-urlencode "node_id=$NODE_ID"
   ```
   - Read the error for the affected stream and resource type. A rejection can list several resources when the controller cannot identify which one failed. After fixing a host problem, change the resource or restart that Envoy to request it again over Delta ADS. See [Rejected Configuration (NACK)](xds.md#rejected-configuration-nack).

#### Checking That Every Instance Accepted a Change

Query each controller pod directly, for example with a port-forward to that pod's cache API. Each builds its own snapshot and reports only its connected streams.

1. Verify that each pod serves the change. For a new listener, check `GET /api/v1/listeners?node_id=<node-id>`.
2. Use `GET /api/v1/clients?node_id=<node-id>` on each pod (`controllerPod`). Account for every expected Envoy instance across the responses. A stream count alone cannot identify which hosts are connected.
3. Inspect each stream's `types[]`. A non-empty `rejections` lists outstanding NACKs. `awaitingAckSince` is the time of the oldest tracked response of that type still awaiting ACK or NACK. If `droppedRejections` or the stream's `unknownNacks` is positive, the rejection history is incomplete.
4. For the changed type, compare `ackedVersion` with `snapshotVersions[<node-id>][<typeUrl>]` from that pod. If versions differ or there is no ACK on this stream, inspect the changed resource in that Envoy's config dump. Check the affected port and traffic path on each instance. An acknowledged listener may still be warming.

To identify an instance behind a gateway:

- `host` comes from the first request identifying the node. The controller checks the top-level `node.metadata` keys `instance`, `hostname` and `HOSTNAME` in order. It accepts printable strings up to 255 bytes, trims whitespace and skips empty values. If none qualifies, the API omits `host`. The client can also report `locality.region`, `locality.zone` and `locality.subZone`.
- `forwardedAddress` is the first entry in `X-Forwarded-For` if it is a valid IP address. Otherwise the API omits the field. Clients and proxies can set this header. Use it for diagnostics only.
- `address` identifies the immediate gRPC peer, which can be the gateway.

What the versions do not tell:

- Compare versions only within the same controller process. Ordinary build counters can restart after a pod restart. Resume and Rollback can publish higher versions for all types, even if only one resource changed.
- On a Delta stream, unchanged subscribed resources can keep their earlier ACK while the snapshot version advances. This also applies to listeners.
- `awaitingFirstResponse: true` means the controller saw a request for this type and has yet to send its first response of that type on this stream. For example, a reconnecting Delta client can already have the current routes or secrets. An absent type has no recorded subscription or response on this stream.

An empty `clients` list means that no stream of that node is connected to the pod that answered, or that the node is outside the node IDs your groups may access.

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
- `Envoy NACKed xDS response` - Envoy rejected an update; the line names the node, the stream, the resources and Envoy's reason
- `xDS rejection cleared` - Envoy accepted a newer version of a rejected resource, or the resource is gone

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
