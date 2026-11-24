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

The controller optimizes xDS updates by tracking resource changes. Snapshot versions only increment when actual resource content changes:

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

Envoy answers every xDS response with an ACK or a NACK with a reason. The controller keeps the result per stream and logs each rejection once:

```json
{"level":"error","logger":"xds.server.callbacks","msg":"Envoy NACKed xDS response","nodeId":"edge","id":5,"protocol":"delta","typeUrl":"type.googleapis.com/envoy.config.listener.v3.Listener","address":"10.0.0.7:45334","envoyVersion":"v1.40.2","nonce":"5","responseVersion":"3","resources":["default/https"],"resourceCount":1,"error":"Error adding/updating listener(s) default/https: unable to open file '/var/log/envoy/missing/access.log': Permission denied"}
```

| Message | Level | When |
|---------|-------|------|
| `Envoy NACKed xDS response` | error | Envoy rejected a response |
| `xDS rejection cleared` | info | A rejected resource was accepted in a later version (`reason: accepted`), removed (`removed`), or Envoy unsubscribed from it (`unsubscribed`) |
| `Envoy NACKed an xDS response unknown to this stream` | info | A NACK named no response of the stream. Logged once per stream |

How to read a rejection:

- **A NACK is about a response, not everything in it.** For listeners and clusters Envoy applies the valid ones and names those it skipped; `resources` lists them. For routes, secrets and endpoints Envoy names nothing, so `resources` lists the whole response — as it does for listeners and clusters when Envoy's text is cut at 4096 characters or names none. The log shows up to 20 names, `resourceCount` is the total. With state-of-the-world `resources` is empty: the whole version is rejected.
- **An ACK means Envoy found the resources valid.** A listener may still be warming up.
- **A rejected resource is not resent by itself.** Delta ADS sends a resource once per version: after fixing the host, change the resource or restart that Envoy.
- **Instances sharing a node ID are told apart by stream.** `id` is the stream number within its `protocol`. Behind a proxy `address` is the proxy's; use `host` or `forwardedAddress` (see [Checking That Every Instance Accepted a Change](troubleshooting.md#checking-that-every-instance-accepted-a-change)).
- **`error` is Envoy's text with quoted data removed.** On a validation failure Envoy echoes the whole message, keys included, so everything from the first double quote is dropped. The full text is in Envoy's own log.

With state-of-the-world the server answers each NACK with the same snapshot, about a thousand times a second, until the snapshot changes; the rejection is logged once per version.

The logger thins identical lines above a hundred per second, so across a large fleet not every instance is logged. The per-stream state — last acknowledged version and current rejections — is served by `GET /api/v1/clients` and shown in the console under **xDS streams**; in development mode also at `:4444/debug/connected-clients`.
