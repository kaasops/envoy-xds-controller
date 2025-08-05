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
8. [Virtual Service Template Parameterization](#virtual-service-template-parameterization)

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

## Virtual Service Template Parameterization

The Envoy XDS Controller supports parameterization of Virtual Service Templates, allowing you to create reusable templates with customizable fields that can be filled in when creating Virtual Services.

### Overview

Template parameterization enables you to:
- Define custom fields in your templates that must be provided when creating a Virtual Service
- Set validation rules for these fields (required, enum values)
- Provide default values for optional fields
- Reference these fields in your template using Go template syntax

### Defining Extra Fields

Extra fields are defined in the `VirtualServiceTemplate` resource under the `spec.extraFields` section:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualServiceTemplate
metadata:
  name: my-template
spec:
  # ... other template configuration ...
  extraFields:
    - name: ServiceName
      type: string
      description: "The name of the service"
      required: true
    - name: Environment
      type: enum
      enum: ["dev", "staging", "prod"]
      description: "Deployment environment"
      required: true
      default: "dev"
    - name: Timeout
      type: string
      description: "Request timeout in seconds"
      required: false
      default: "30s"
```

### Field Properties

Each extra field can have the following properties:

| Property | Description | Required |
|----------|-------------|----------|
| `name` | The name of the field (used in template references) | Yes |
| `type` | Data type of the field. Valid types are: `string` and `enum` | Yes |
| `description` | Human-readable description of the field | No |
| `required` | Whether the field must be provided (true/false) | No (defaults to false) |
| `enum` | List of allowed values (for enum type) | Only for enum type |
| `default` | Default value if not provided | No |

### Using Field Values in Templates

You can reference the extra fields in your template using Go template syntax with the field name:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualServiceTemplate
metadata:
  name: header-template
spec:
  virtualHost:
    request_headers_to_add:
      - append_action: OVERWRITE_IF_EXISTS_OR_ADD
        header:
          key: X-Service-Name
          value: "{{ .ServiceName }}"
      - append_action: OVERWRITE_IF_EXISTS_OR_ADD
        header:
          key: X-Environment
          value: "{{ .Environment }}"
  extraFields:
    - name: ServiceName
      type: string
      description: "The name of the service"
      required: true
    - name: Environment
      type: enum
      enum: ["dev", "staging", "prod"]
      description: "Deployment environment"
      required: true
      default: "dev"
```

### Example

Here's a complete example of a template with extra fields and how they're used:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualServiceTemplate
metadata:
  name: api-service-template
spec:
  listener:
    name: https
  accessLogConfig:
    name: access-log-config
  tlsConfig:
    autoDiscovery: true
  virtualHost:
    request_headers_to_add:
      - append_action: OVERWRITE_IF_EXISTS_OR_ADD
        header:
          key: X-API-Version
          value: "{{ .ApiVersion }}"
      - append_action: OVERWRITE_IF_EXISTS_OR_ADD
        header:
          key: X-Service-Tier
          value: "{{ .ServiceTier }}"
    routes:
      - match:
          prefix: "/{{ .ApiVersion }}/{{ .ServicePath }}"
        route:
          cluster: "{{ .ServiceName }}"
          timeout: "{{ .Timeout }}"
  extraFields:
    - name: ServiceName
      type: string
      description: "The name of the service cluster"
      required: true
    - name: ServicePath
      type: string
      description: "The path segment for the service"
      required: true
    - name: ApiVersion
      type: string
      description: "API version (v1, v2, etc.)"
      required: true
      default: "v1"
    - name: ServiceTier
      type: enum
      enum: ["frontend", "backend", "data"]
      description: "Service tier"
      required: true
    - name: Timeout
      type: string
      description: "Request timeout"
      required: false
      default: "30s"
```

When creating a Virtual Service from this template through the UI, users will be prompted to fill in these fields, and the values will be substituted into the template.