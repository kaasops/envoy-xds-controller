# Templates Guide: Envoy XDS Controller

This document provides guidance on using virtual service templates to reuse common configurations across multiple virtual services.

## Table of Contents

1. [How Templates Work](#how-templates-work)
2. [Template Options](#template-options)
3. [ExtraFields Feature](#extrafields-feature)
4. [Examples](#examples)
5. [Nested Fields and Merging Behavior](#nested-fields-and-merging-behavior)
6. [Template Rendering with Variable Substitution](#template-rendering-with-variable-substitution)
7. [Best Practices](#best-practices)

Virtual service templates provide a way to reuse common configurations across multiple virtual services. Templates define a base configuration that can be extended or modified by individual virtual services. This mechanism helps maintain consistency and reduces duplication in your Envoy configuration.

## How templates work

When a virtual service references a template, the following process occurs:

1. The template's configuration is used as the base
2. Any ExtraFields defined in the template are validated against the values provided by the virtual service
3. If ExtraFields are present, the template is rendered with the ExtraFields values as variables
4. The virtual service's configuration is applied on top of the template
5. Any template options specified in the virtual service are applied to control how specific fields are merged

The merging process happens during resource building, before the configuration is sent to Envoy. The template is applied to the virtual service's spec, and then the resulting configuration is used to build the Envoy resources.

## Template options

Template options allow you to control how specific fields from the template are handled when merging with the virtual service configuration. There are three modifiers available:

- **merge** (default) - Merges object fields, overrides primitive types in existing objects, merges lists by appending items
- **replace** - Completely replaces objects or lists instead of merging them
- **delete** - Deletes a field by key (does not work for list elements)

Each template option specifies a field path and a modifier. The field path identifies the field to apply the modifier to, and the modifier determines how the field is handled during merging.

## ExtraFields feature

The ExtraFields feature allows templates to define additional configurable fields that virtual services can provide values for. This enables parameterized templates where the configuration can be customized based on the values provided by the virtual service.

ExtraFields are defined in the template with the following properties:

- **name** - The name of the field (required)
- **description** - A description of the field's purpose (optional)
- **type** - The data type of the field (required, e.g., "string", "enum", "number", "boolean")
- **required** - Whether the field is required (default: false)
- **enum** - A list of valid values for enum type fields (required for enum type)
- **default** - A default value for the field (optional)

When a virtual service uses a template with ExtraFields, it must provide values for all required fields and can optionally provide values for non-required fields. The system validates that:

1. All required fields are provided and not empty
2. Enum fields have valid values from the predefined set
3. Only fields defined in the template are provided by the virtual service

## Examples

### Basic template usage

Here's a basic example of using a template:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: demo-virtual-service
  annotations:
    envoy.kaasops.io/node-id: test
spec:
  template:
    name: https-template
  virtualHost:
    domains:
      - example.com
```

Template definition:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualServiceTemplate
metadata:
  name: https-template
spec:
  listener:
    name: https
  accessLogConfig:
    name: access-log-config
  tlsConfig:
    autoDiscovery: true
  virtualHost:
    name: test-virtual-host
    routes:
      - match:
          prefix: "/"
        direct_response:
          status: 200
          body:
            inline_string: "{\"message\":\"Hello from template\"}"
```

In this example, the virtual service inherits all configuration from the template and adds the `domains` field to the `virtualHost`. The resulting configuration will have:
- The HTTPS listener from the template
- The access log configuration from the template
- The TLS configuration from the template
- The virtual host with the name from the template, the routes from the template, and the domains from the virtual service

### Using template options

Here's an example using template options to customize how fields are merged:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: demo-virtual-service
  annotations:
    envoy.kaasops.io/node-id: test
spec:
  template:
    name: https-template
  templateOptions:
    - field: accessLogConfig
      modifier: delete
    - field: additionalHttpFilters
      modifier: replace
  additionalHttpFilters:
    - my-filter-1
    - my-filter-2
  virtualHost:
    domains:
      - example.com
```

In this example:
1. The `accessLogConfig` field from the template is deleted (not used in the final configuration)
2. The `additionalHttpFilters` field is replaced entirely with the new list instead of being merged
3. The `virtualHost` field is merged with the template's virtual host (default behavior)

### Overriding fields

You can override specific fields from the template by providing them in the virtual service:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: demo-virtual-service
  annotations:
    envoy.kaasops.io/node-id: test
spec:
  template:
    name: https-template
  accessLogConfig:
    name: access-log-config-2  # Overrides the template's accessLogConfig
  virtualHost:
    domains:
      - exc.kaasops.io
```

In this example, the `accessLogConfig` from the template is overridden with a different access log configuration.

### Replacing with a different type

You can delete a field from the template and replace it with a different type of configuration:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: demo-virtual-service
  annotations:
    envoy.kaasops.io/node-id: test
spec:
  template:
    name: https-template
  templateOptions:
    - field: accessLogConfig
      modifier: delete
  accessLog:  # Different type of access log configuration
    name: envoy.access_loggers
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog
      path: /tmp/app.log
      log_format:
        json_format:
          message: "%LOCAL_REPLY_BODY%"
          status: "%RESPONSE_CODE%"
          duration: "%DURATION%"
  virtualHost:
    domains:
      - exc.kaasops.io
```

In this example, the `accessLogConfig` reference from the template is removed, and instead, an inline `accessLog` configuration is defined in the virtual service.

### Using ExtraFields

Here's an example of a template with ExtraFields:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualServiceTemplate
metadata:
  name: api-gateway-template
spec:
  listener:
    name: https
  tlsConfig:
    autoDiscovery: true
  virtualHost:
    name: api-gateway
    routes:
      - match:
          prefix: "/api"
        route:
          cluster: "{{.api_service}}"
  extraFields:
    - name: api_service
      description: "The name of the API service cluster"
      type: "string"
      required: true
    - name: rate_limit
      description: "Rate limit in requests per second"
      type: "number"
      default: "100"
    - name: auth_enabled
      description: "Enable authentication"
      type: "boolean"
      default: "true"
    - name: log_level
      description: "Logging level"
      type: "enum"
      enum: ["debug", "info", "warn", "error"]
      default: "info"
```

And a virtual service using this template:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: user-api-service
  annotations:
    envoy.kaasops.io/node-id: test
spec:
  template:
    name: api-gateway-template
  extraFields:
    api_service: "user-service"
    rate_limit: "200"
    auth_enabled: "false"
    log_level: "debug"
  virtualHost:
    domains:
      - api.example.com
```

In this example:
1. The template defines four extra fields: `api_service` (required), `rate_limit`, `auth_enabled`, and `log_level` (enum)
2. The virtual service provides values for all four fields
3. The template uses the `api_service` value in the route configuration with variable substitution (`{{.api_service}}`)
4. The resulting configuration will have a route to the "user-service" cluster

### Complex template with multiple ExtraFields

Here's a more complex example with multiple ExtraFields and variable substitution:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualServiceTemplate
metadata:
  name: microservice-template
spec:
  listener:
    name: https
  tlsConfig:
    autoDiscovery: true
  virtualHost:
    name: "{{.service_name}}-service"
    domains:
      - "{{.service_name}}.{{.domain}}"
    routes:
      - match:
          prefix: "/api/v{{.api_version}}"
        route:
          cluster: "{{.service_name}}-v{{.api_version}}"
          timeout: "{{.timeout}}s"
      - match:
          prefix: "/health"
        direct_response:
          status: 200
          body:
            inline_string: "{\"status\":\"healthy\"}"
  extraFields:
    - name: service_name
      description: "The name of the microservice"
      type: "string"
      required: true
    - name: domain
      description: "The domain for the service"
      type: "string"
      required: true
    - name: api_version
      description: "API version number"
      type: "number"
      default: "1"
    - name: timeout
      description: "Request timeout in seconds"
      type: "number"
      default: "30"
```

And a virtual service using this template:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: payment-service
  annotations:
    envoy.kaasops.io/node-id: test
spec:
  template:
    name: microservice-template
  extraFields:
    service_name: "payment"
    domain: "example.com"
    api_version: "2"
    timeout: "15"
  additionalHttpFilters:
    - name: envoy.filters.http.ratelimit
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit
        domain: "payment-ratelimit"
        rate_limit_service:
          grpc_service:
            envoy_grpc:
              cluster_name: ratelimit-service
```

In this example:
1. The template defines four extra fields: `service_name`, `domain`, `api_version`, and `timeout`
2. The virtual service provides values for all four fields
3. The template uses these values in multiple places with variable substitution
4. The resulting configuration will have:
   - A virtual host named "payment-service"
   - A domain "payment.example.com"
   - A route to the "payment-v2" cluster with a 15-second timeout
   - A health check route
   - An additional rate limit HTTP filter

## Nested fields and merging behavior

When merging nested fields, the behavior depends on the field type:

### Objects

For objects, fields are merged recursively. If a field exists in both the template and the virtual service, the value from the virtual service takes precedence for primitive types. For nested objects, the merging continues recursively.

Example:

**Template:**
```yaml
virtualHost:
  name: test-virtual-host
  routes:
    - match:
        prefix: "/"
      direct_response:
        status: 200
```

**Virtual Service:**
```yaml
virtualHost:
  domains:
    - example.com
  routes:
    - match:
        prefix: "/api"
      direct_response:
        status: 201
```

**Result (merged):**
```yaml
virtualHost:
  name: test-virtual-host
  domains:
    - example.com
  routes:
    - match:
        prefix: "/"
      direct_response:
        status: 200
    - match:
        prefix: "/api"
      direct_response:
        status: 201
```

### Lists

By default, lists are merged by appending items from the virtual service to the items from the template. If you want to replace the entire list, use the `replace` modifier.

Example with default merging:

**Template:**
```yaml
additionalHttpFilters:
  - template-filter-1
  - template-filter-2
```

**Virtual Service:**
```yaml
additionalHttpFilters:
  - service-filter-1
```

**Result (merged):**
```yaml
additionalHttpFilters:
  - template-filter-1
  - template-filter-2
  - service-filter-1
```

Example with replace modifier:

**Template:**
```yaml
additionalHttpFilters:
  - template-filter-1
  - template-filter-2
```

**Virtual Service:**
```yaml
templateOptions:
  - field: additionalHttpFilters
    modifier: replace
additionalHttpFilters:
  - service-filter-1
```

**Result (replaced):**
```yaml
additionalHttpFilters:
  - service-filter-1
```

## Template rendering with variable substitution

When a template includes ExtraFields, it can use the values provided by the virtual service for variable substitution in the template configuration. This is done using Go template syntax with the `{{.field_name}}` notation.

Here's an example of how variable substitution works:

**Template:**
```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualServiceTemplate
metadata:
  name: dynamic-route-template
spec:
  virtualHost:
    name: "{{.service_name}}-host"
    routes:
      - match:
          prefix: "/{{.path_prefix}}"
        route:
          cluster: "{{.target_cluster}}"
          timeout: "{{.timeout}}s"
  extraFields:
    - name: service_name
      description: "Service name"
      type: "string"
      required: true
    - name: path_prefix
      description: "URL path prefix"
      type: "string"
      required: true
    - name: target_cluster
      description: "Target cluster name"
      type: "string"
      required: true
    - name: timeout
      description: "Request timeout in seconds"
      type: "number"
      default: "30"
```

**Virtual Service:**
```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: user-service
spec:
  template:
    name: dynamic-route-template
  extraFields:
    service_name: "user"
    path_prefix: "users"
    target_cluster: "user-service"
    timeout: "15"
  virtualHost:
    domains:
      - api.example.com
```

**Result (after variable substitution and merging):**
```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: user-service
spec:
  virtualHost:
    name: "user-host"
    domains:
      - api.example.com
    routes:
      - match:
          prefix: "/users"
        route:
          cluster: "user-service"
          timeout: "15s"
```

Variable substitution happens before the merging process, so the template is first rendered with the provided ExtraFields values, and then the virtual service's configuration is merged with the rendered template.

## Best practices

1. Use templates for common configurations that are shared across multiple virtual services
2. Keep templates focused on a specific use case (e.g., HTTPS configuration, API gateway configuration)
3. Use descriptive names for templates to make their purpose clear
4. Document the expected fields that virtual services should provide when using the template
5. Use ExtraFields to make templates more flexible and reusable
6. Provide default values for non-required ExtraFields when possible
7. Use template options to customize the merging behavior when needed
8. Test your templates with different virtual service configurations to ensure they work as expected
9. Use variable substitution with ExtraFields to create dynamic configurations
10. Consider creating a library of templates for different use cases in your organization