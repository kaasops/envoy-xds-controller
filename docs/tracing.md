# Tracing in Envoy XDS Controller

This document explains how to configure HTTP tracing for your VirtualService resources using the Tracing custom resource or inline configuration.

## Overview
The controller supports Envoy HttpConnectionManager.Tracing configuration delivered in two ways:
- Inline in VirtualService: spec.tracing
- By reference to a Tracing CR: spec.tracingRef

Priority rule: inline spec.tracing takes precedence over spec.tracingRef. Setting both at the same time is not allowed (webhooks will reject such resources).

## Inline vs Reference
- Inline (spec.tracing): place raw Envoy Tracing configuration directly into the VirtualService. Useful for quick/local setups.
- Reference (spec.tracingRef): reuse a shared Tracing resource across many VirtualServices.

Only one of spec.tracing or spec.tracingRef may be set. If spec.tracingRef is set and namespace is omitted, the VirtualService namespace is used. Webhooks validate the XOR rule and the existence of the referenced Tracing.

## Tracing CR Examples
Two common providers are shown below. Ensure referenced clusters exist (otel-collector, zipkin).

OpenTelemetry (OTLP):

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: Tracing
metadata:
  name: tracing-otlp
spec:
  provider:
    name: envoy.tracers.opentelemetry
    typed_config:
      "@type": type.googleapis.com/envoy.config.trace.v3.OpenTelemetryConfig
      grpc_service:
        envoy_grpc:
          cluster_name: otel-collector
```

Zipkin:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: Tracing
metadata:
  name: tracing-zipkin
spec:
  provider:
    name: envoy.tracers.zipkin
    typed_config:
      "@type": type.googleapis.com/envoy.config.trace.v3.ZipkinConfig
      collector_cluster: zipkin
      collector_endpoint: /api/v2/spans
      collector_endpoint_version: HTTP_JSON
```

## VirtualService Examples
Inline tracing:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: vs-tracing-inline
  annotations:
    envoy.kaasops.io/node-id: "node1"
spec:
  listener:
    name: listener-sample
  virtualHost:
    name: inline-tracing-vh
    domains: ["*"]
    routes:
      - match: { prefix: "/" }
        route: { cluster: example }
  tracing:
    provider:
      name: envoy.tracers.zipkin
      typed_config:
        "@type": type.googleapis.com/envoy.config.trace.v3.ZipkinConfig
        collector_cluster: zipkin
        collector_endpoint: /api/v2/spans
        collector_endpoint_version: HTTP_JSON
```

Reference to Tracing CR:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: vs-tracing-ref
  annotations:
    envoy.kaasops.io/node-id: "node1"
spec:
  listener:
    name: listener-sample
  virtualHost:
    name: ref-tracing-vh
    domains: ["*"]
    routes:
      - match: { prefix: "/" }
        route: { cluster: example }
  tracingRef:
    name: tracing-zipkin
```

## Cluster Requirements
If your tracing config references a cluster (e.g., `otel-collector`, `zipkin`), the cluster must exist as an Envoy Cluster resource and be valid. The controller resolves and validates these references during snapshot build. Missing clusters will result in status.invalid=true on affected VirtualServices and corresponding error messages.

## Validation Rules
- XOR: only one of spec.tracing or spec.tracingRef may be set (enforced by webhooks for VirtualService and VirtualServiceTemplate).
- If spec.tracingRef is set, the referenced Tracing must exist (same namespace default if omitted).
- Tracing resources are validated via their own webhook (basic schema validation of the Envoy structure).

## Debugging
- /debug/store — observe loaded Tracing resources and current state.
- /debug/xds — inspect generated snapshots. Ensure HttpConnectionManager.Tracing is present when expected.
- Controller logs include context (resource names) on tracing-related errors.

## Common Errors
- "only one of spec.tracing or spec.tracingRef may be set" — remove one of them.
- "tracing <ns>/<name> not found" — create the Tracing CR or fix the reference/namespace.
- "cluster <name> not found" — create a Cluster resource with that name or update tracing config.
