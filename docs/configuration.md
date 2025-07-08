# Configuration Guide: Envoy XDS Controller

This document provides a comprehensive guide to configuring the Envoy XDS Controller.

## Table of Contents

1. [Helm Chart Configuration](#helm-chart-configuration)
2. [Environment Variables](#environment-variables)
3. [Authentication Configuration](#authentication-configuration)
4. [xDS Server Configuration](#xds-server-configuration)
5. [Cache API Configuration](#cache-api-configuration)
6. [UI Configuration](#ui-configuration)
7. [Webhook Configuration](#webhook-configuration)

## Helm Chart Configuration

The Envoy XDS Controller is primarily configured through its Helm chart. Below are the key configuration parameters:

### General Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `development` | Enable development mode | `false` |
| `watchNamespaces` | List of namespaces to watch (empty means all) | `[]` |
| `createCRD` | Enable CRD creation and management | `true` |

### Image Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Controller image repository | `kaasops/envoy-xds-controller` |
| `image.tag` | Controller image tag | Chart AppVersion |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |

### Resource Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `1` |
| `resources.limits.memory` | Memory limit | `1Gi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `50Mi` |

## Environment Variables

The controller can be configured using environment variables through the `envs` parameter in the Helm chart:

```yaml
envs:
  LOG_LEVEL: "debug"
  WEBHOOK_DISABLE: "true"
```

## Authentication Configuration

Authentication is configured through the `auth` section in the Helm chart:

```yaml
auth:
  enabled: false
  oidc:
    clientId: "envoy-xds-controller"
    issuerUrl: "http://dex.dex:5556"
    scope: "openid profile groups"
    redirectUri: "http://localhost:8080/callback"
  acl:
    nodeIdsByGroup:
      admins:
        - "*"
      authors:
        - "node1"
      users:
        - "node1"
        - "node2"
  rbacPolicy: |-
    g, admins, role:editor, _
    g, admins, role:editor, dev
    # Additional policy rules...
```

For more details on authentication and authorization, see the [Auth Documentation](auth.md) and [Security Documentation](security.md).

## xDS Server Configuration

The xDS server is configured through the `xds` section:

```yaml
xds:
  port: 9000
```

For more details on the xDS server implementation, see the [xDS Documentation](xds.md).

## Cache API Configuration

The Cache API provides a REST interface for managing Envoy configurations:

```yaml
cacheAPI:
  enabled: false
  port: 9999
  grpcPort: 10000
  address: "localhost:9999"
  scheme: "http"
  ingress:
    enabled: false
    # Additional ingress configuration...
```

## UI Configuration

The Web UI for managing Envoy configurations:

```yaml
ui:
  enabled: false
  image:
    repository: kaasops/envoy-xds-controller-ui
    tag: "" # defaults to Chart.AppVersion
  cacheAPI: "http://exc-envoy-xds-controller-cache-api:9999"
  grpcAPI: "http://exc-envoy-xds-controller-grpc-api:10000"
  port: 8080
  ingress:
    enabled: false
    # Additional ingress configuration...
```

## Webhook Configuration

The validating webhook for Kubernetes resources:

```yaml
webhook:
  enabled: true
  name: "envoy-xds-controller-validating-webhook-configuration"
  port: 9443
  tls:
    name: "envoy-xds-controller-webhook-cert"
```

## Node and Access Group Configuration

Configure the available node IDs and access groups:

```yaml
config:
  nodeIds:
    - "node1"
    - "node2"
    - "test"
  accessGroups:
    - "group1"
    - "group2"
    - "test"
```