# xDS Server Guide

This document explains how the xDS server is implemented in the Envoy XDS Controller, including its responsibilities, structure, and integration with Kubernetes.

## Table of Contents

1. [What is xDS?](#what-is-xds)
2. [Implementation Overview](#implementation-overview)
3. [Flow of Updates](#flow-of-updates)
4. [Dynamic Configuration](#dynamic-configuration)
5. [Envoy Bootstrap](#envoy-bootstrap)
6. [Snapshot Version Stability](#snapshot-version-stability)
7. [Rejected Configuration (NACK)](#rejected-configuration-nack)

## What is xDS?

xDS is a set of discovery service APIs used by [Envoy Proxy](https://www.envoyproxy.io/) to dynamically receive configuration updates from a control plane. The core xDS APIs used in this controller:

| API | Name | Description |
|-----|------|-------------|
| **LDS** | Listener Discovery Service | Configures listeners (ports, protocols, filter chains) |
| **RDS** | Route Discovery Service | Configures routing rules and virtual hosts |
| **CDS** | Cluster Discovery Service | Configures upstream clusters |
| **SDS** | Secret Discovery Service | Configures TLS certificates and keys |

---

## Implementation Overview

The controller uses [go-control-plane](https://github.com/envoyproxy/go-control-plane) to implement an xDS server compatible with Envoy v3 APIs.

### Key Packages

| Package | Description |
|---------|-------------|
| `internal/xds/cache` | Snapshot cache storing xDS configurations per Envoy node |
| `internal/xds/updater` | Processes Kubernetes events and rebuilds xDS snapshots |
| `internal/xds/resbuilder` | Transforms CRs into Envoy xDS resources |
| `internal/xds/api` | gRPC server serving xDS endpoints |

---

## Flow of Updates

1. **Controller** watches Kubernetes CRs (VirtualService, Listener, Cluster, etc.)
2. **Updater** receives change notifications and updates the Store
3. **ResourceBuilder** transforms CRs into xDS resources (listeners, routes, clusters, secrets)
4. **SnapshotCache** stores the built configuration per node ID
5. **gRPC Server** pushes updates to connected Envoy proxies via xDS protocol

---

## Dynamic Configuration

The controller supports hot reload of configuration without restarting Envoy. When a Kubernetes CR changes, the update is propagated to Envoy proxies within milliseconds.

---

## Envoy Bootstrap

Envoy must connect over delta ADS: `ads_config.api_type: DELTA_GRPC`.

The snapshot cache runs in ADS mode, so an update reaches Envoy in order: clusters before the listeners that use them. With state-of-the-world (`api_type: GRPC`) the same mode holds back a whole RDS or SDS response while the snapshot has a resource Envoy did not request, for example the route of a rejected listener.

---

## Snapshot Version Stability

The table shows typical version changes during ordinary rebuilds. Resume and Rollback can assign new versions to unchanged resource types when publishing a snapshot.

| Resource Change | Listeners | Routes | Clusters |
|-----------------|-----------|--------|----------|
| VirtualService re-apply (no spec changes) | stable | stable | stable |
| accessLogConfig added/changed | **changes** | stable | stable |
| Route content changed (path, response) | stable | **changes** | stable |
| Cluster reference added | stable | **changes** | **changes** |

This behavior is achieved through:
- `IsEqual()` comparison on VirtualService/VirtualServiceTemplate before rebuilding snapshots
- `NormalizeSpec()` to ensure consistent comparison regardless of field order
- Deterministic resource ordering to prevent spurious version increments from map iteration

---

## Rejected Configuration (NACK)

Envoy ACKs a response when it considers its resources valid. A listener may still be warming up. A NACK means at least one resource failed. Envoy may already have applied others in the same response. [Envoy ACK and NACK semantics](https://www.envoyproxy.io/docs/envoy/v1.40.2/api-docs/xds_protocol#ack-and-nack-semantics-summary)

The controller records rejections per stream and resource type. Here, a listener failed to open its access log.

```json
{"level":"error","logger":"xds.server.callbacks","msg":"Envoy NACKed xDS response","nodeId":"edge","id":5,"protocol":"delta","typeUrl":"type.googleapis.com/envoy.config.listener.v3.Listener","address":"10.0.0.7:45334","envoyVersion":"v1.40.2","nonce":"5","responseVersion":"3","resources":["default/https"],"resourceCount":1,"error":"Error adding/updating listener(s) default/https: unable to open file '/var/log/envoy/missing/access.log': Permission denied"}
```

| Message | Level | When |
|---------|-------|------|
| `Envoy NACKed xDS response` | error | The controller recorded a rejection |
| `xDS rejection cleared` | info | A rejected resource was accepted in a later version (`reason: accepted`), removed (`removed`), or Envoy unsubscribed from it (`unsubscribed`) |
| `Envoy NACKed an xDS response unknown to this stream` | info | The controller could not match the NACK to a recorded response. Logged once per stream |

- **Resources.** For Delta listeners and clusters, the controller lists the resources named in Envoy's rejection. If the message is truncated or the names cannot be identified, it lists all resources delivered in that response as candidates. It does the same for routes, secrets and endpoints, where Envoy can stop at the first error. The log shows up to 20 names. Its `resourceCount` gives the total before truncation. The API lists the names and versions in each current rejection. With state-of-the-world (SotW), the controller records the rejected type version and leaves `resources` empty.
- **Retries.** On this server, a Delta NACK alone does not trigger another delivery. After fixing the host, change the rejected resource or restart the affected Envoy. With SotW, the server can immediately resend the same version. The controller logs the first NACK for that version on each stream and type. Repeats increment `count` in the API.
- **Error details.** A validation error can contain the whole resource, including keys. The controller removes data starting at the first double quote or PEM header and may retain an unquoted validation reason. Other unquoted text, including paths and header values, can appear in the log and API. Envoy's own log has the full error.

With default production settings, the logger samples entries that share a level and message after the first 100 per second, even across instances. For the last ACK and current rejections, open **xDS streams** in the console or read `GET /api/v1/clients`. The log's `id` matches the API's `streamId` on the same controller pod and `protocol`. Behind a proxy, the API's `host` and `forwardedAddress` can help identify the instance. Development mode also exposes `:4444/debug/connected-clients`. [Checking that every instance accepted a change](troubleshooting.md#checking-that-every-instance-accepted-a-change)
