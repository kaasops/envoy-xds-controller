# TLS Configuration Guide

This document describes how to configure TLS for VirtualServices in the Envoy XDS Controller.

## Table of Contents

1. [Overview](#overview)
2. [TLS Configuration Modes](#tls-configuration-modes)
3. [Secret Reference Mode](#secret-reference-mode)
4. [Auto Discovery Mode](#auto-discovery-mode)
5. [Secret Selection Algorithm](#secret-selection-algorithm)
6. [Certificate Requirements](#certificate-requirements)
7. [Examples](#examples)
8. [Troubleshooting](#troubleshooting)

## Overview

The Envoy XDS Controller supports TLS termination for VirtualServices. TLS configuration is specified in the `spec.tlsConfig` field of a VirtualService or VirtualServiceTemplate.

There are two modes for configuring TLS certificates:
- **Secret Reference** (`secretRef`) - explicitly reference a Kubernetes TLS secret
- **Auto Discovery** (`autoDiscovery`) - automatically find secrets based on domain annotations

## TLS Configuration Modes

### Secret Reference Mode

Use `secretRef` when you want to explicitly specify which Kubernetes secret contains the TLS certificate:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: my-service
  annotations:
    envoy.kaasops.io/node-id: my-node
spec:
  listener:
    name: https
  tlsConfig:
    secretRef:
      name: my-tls-secret
      namespace: my-namespace  # optional, defaults to VirtualService namespace
  virtualHost:
    domains:
      - example.com
    routes:
      - match:
          prefix: "/"
        route:
          cluster: my-cluster
```

### Auto Discovery Mode

Use `autoDiscovery` when you want the controller to automatically find TLS secrets based on domain annotations:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: my-service
  annotations:
    envoy.kaasops.io/node-id: my-node
spec:
  listener:
    name: https
  tlsConfig:
    autoDiscovery: true
  virtualHost:
    domains:
      - example.com
      - api.example.com
    routes:
      - match:
          prefix: "/"
        route:
          cluster: my-cluster
```

For auto discovery to work, your TLS secrets must have the `envoy.kaasops.io/domains` annotation:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: example-tls
  namespace: my-namespace
  annotations:
    envoy.kaasops.io/domains: "example.com,api.example.com"
type: kubernetes.io/tls
data:
  tls.crt: <base64-encoded-certificate>
  tls.key: <base64-encoded-private-key>
```

#### Wildcard Domains

Auto discovery supports wildcard domain matching. A secret annotated with `*.example.com` will match any subdomain:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: wildcard-tls
  annotations:
    envoy.kaasops.io/domains: "*.example.com"
type: kubernetes.io/tls
data:
  tls.crt: <base64-encoded-certificate>
  tls.key: <base64-encoded-private-key>
```

This secret will be used for `api.example.com`, `www.example.com`, etc.

## Secret Selection Algorithm

When multiple TLS secrets exist for the same domain (e.g., in different namespaces), the controller uses a deterministic algorithm to select the best secret.

### Selection Priority

The algorithm evaluates candidates in the following order:

| Priority | Criterion | Description |
|----------|-----------|-------------|
| 1 | **Certificate Validity** | Valid (not expired) certificates have highest priority |
| 2 | **Namespace Preference** | Secrets in the same namespace as VirtualService are preferred |
| 3 | **Alphabetical Order** | Tie-breaker: sorted by namespace, then by name |

### Validity States

Certificates are classified into three validity states:

| State | Description | Priority |
|-------|-------------|----------|
| **Valid** | Certificate `NotAfter` date is in the future | Highest |
| **Unknown** | Certificate could not be parsed (zero `NotAfter`) | Medium |
| **Expired** | Certificate `NotAfter` date is in the past | Lowest |

### Selection Examples

**Example 1: Valid certificate preferred over expired**

```
Domain: example.com
VirtualService namespace: ns2

Secrets:
  ns1/cert-a: valid,   expires 2025-12-31
  ns2/cert-b: expired, expired 2024-01-01

Result: ns1/cert-a (valid > expired, even though ns2 is preferred namespace)
```

**Example 2: Same namespace preferred when both valid**

```
Domain: example.com
VirtualService namespace: ns2

Secrets:
  ns1/cert-a: valid, expires 2025-12-31
  ns2/cert-b: valid, expires 2025-06-30

Result: ns2/cert-b (both valid, ns2 matches VirtualService namespace)
```

**Example 3: Alphabetical fallback**

```
Domain: example.com
VirtualService namespace: ns3

Secrets:
  ns1/cert-a: valid, expires 2025-12-31
  ns2/cert-b: valid, expires 2025-06-30

Result: ns1/cert-a (both valid, neither matches ns3, alphabetically ns1 < ns2)
```

**Example 4: Three-tier priority**

```
Domain: example.com
VirtualService namespace: ns2

Secrets:
  ns1/cert-valid:   valid,   expires 2025-12-31
  ns2/cert-unknown: unknown, could not parse certificate
  ns3/cert-expired: expired, expired 2024-01-01

Result: ns1/cert-valid (valid > unknown > expired)
```

### Certificate Chain Handling

When a secret contains a certificate chain (multiple certificates), the controller uses the **minimum** `NotAfter` date from all certificates in the chain. This ensures that the most restrictive expiration is considered.

## Certificate Requirements

TLS secrets must meet the following requirements:

1. **Secret Type**: `kubernetes.io/tls` or `Opaque`
2. **Required Keys**:
   - `tls.crt` - PEM-encoded certificate (or certificate chain)
   - `tls.key` - PEM-encoded private key
3. **Certificate Format**: PEM-encoded X.509 certificate

### Certificate Chain Order

For certificate chains, include certificates in the following order:
1. End-entity (server) certificate
2. Intermediate CA certificates
3. Root CA certificate (optional)

## Examples

### Basic HTTPS VirtualService with secretRef

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: secure-api
  annotations:
    envoy.kaasops.io/node-id: production
spec:
  listener:
    name: https
  tlsConfig:
    secretRef:
      name: api-tls-cert
  virtualHost:
    domains:
      - api.example.com
    routes:
      - match:
          prefix: "/"
        route:
          cluster: api-backend
```

### HTTPS VirtualService with Auto Discovery

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: multi-domain-service
  annotations:
    envoy.kaasops.io/node-id: production
spec:
  listener:
    name: https
  tlsConfig:
    autoDiscovery: true
  virtualHost:
    domains:
      - www.example.com
      - api.example.com
      - admin.example.com
    routes:
      - match:
          prefix: "/"
        route:
          cluster: main-backend
```

### TLS Secret with Domain Annotation

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: example-com-tls
  namespace: production
  annotations:
    envoy.kaasops.io/domains: "www.example.com,api.example.com,admin.example.com"
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTi... # base64-encoded certificate
  tls.key: LS0tLS1CRUdJTi... # base64-encoded private key
```

### Wildcard Certificate Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: wildcard-example-tls
  namespace: production
  annotations:
    envoy.kaasops.io/domains: "*.example.com"
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTi... # base64-encoded wildcard certificate
  tls.key: LS0tLS1CRUdJTi... # base64-encoded private key
```

## Troubleshooting

### Common Issues

#### "can't find secret for domain X"

**Cause**: Auto discovery is enabled but no secret with matching domain annotation exists.

**Solutions**:
1. Verify the secret has `envoy.kaasops.io/domains` annotation
2. Check that the domain in annotation matches exactly (or use wildcard)
3. Ensure the secret is in a namespace the controller can access

#### "secret X not found"

**Cause**: Referenced secret doesn't exist or is in a different namespace.

**Solutions**:
1. Verify secret exists: `kubectl get secret <name> -n <namespace>`
2. Check the `secretRef.namespace` field if referencing cross-namespace

#### VirtualService uses wrong certificate

**Cause**: Multiple secrets exist for the same domain, and the selection algorithm picked a different one.

**Solutions**:
1. Use explicit `secretRef` instead of `autoDiscovery` for precise control
2. Check certificate validity - expired certs have lowest priority
3. Place the preferred secret in the same namespace as VirtualService
4. Review the [Secret Selection Algorithm](#secret-selection-algorithm) section

#### Certificate appears valid but treated as "unknown"

**Cause**: Certificate parsing failed (malformed PEM, corrupted data).

**Solutions**:
1. Verify certificate is valid PEM: `openssl x509 -in cert.pem -text -noout`
2. Check for encoding issues (must be base64-encoded in secret)
3. Ensure `tls.crt` key exists in secret data

### Debug Commands

Check secret annotations:
```bash
kubectl get secret <name> -n <namespace> -o jsonpath='{.metadata.annotations}'
```

Verify certificate expiration:
```bash
kubectl get secret <name> -n <namespace> -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | openssl x509 -noout -dates
```

List all TLS secrets with domain annotations:
```bash
kubectl get secrets --all-namespaces -o json | \
  jq -r '.items[] | select(.metadata.annotations["envoy.kaasops.io/domains"] != null) |
  "\(.metadata.namespace)/\(.metadata.name): \(.metadata.annotations["envoy.kaasops.io/domains"])"'
```
