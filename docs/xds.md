# xDS Server Guide

This document explains how the xDS server is implemented in the Envoy XDS Controller, including its responsibilities, structure, and integration with Kubernetes.

## Table of Contents

1. [What is xDS?](#what-is-xds)
2. [Implementation Overview](#implementation-overview)
3. [Flow of Updates](#flow-of-updates)
4. [Dynamic Configuration](#dynamic-configuration)
5. [Envoy Bootstrap](#envoy-bootstrap)
6. [Snapshot Version Stability](#snapshot-version-stability)

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
