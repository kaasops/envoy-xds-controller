# Templates Guide: Envoy XDS Controller

This document provides guidance on using virtual service templates to reuse common configurations across multiple virtual services.

## Table of Contents

1. [How Templates Work](#how-templates-work)
2. [Template Options](#template-options)
3. [Examples](#examples)
4. [Nested Fields and Merging Behavior](#nested-fields-and-merging-behavior)
5. [Best Practices](#best-practices)

Virtual service templates provide a way to reuse common configurations across multiple virtual services. Templates define a base configuration that can be extended or modified by individual virtual services. This mechanism helps maintain consistency and reduces duplication in your Envoy configuration.

## How templates work

When a virtual service references a template, the following process occurs:

1. The template's configuration is used as the base
2. The virtual service's configuration is applied on top of the template
3. Any template options specified in the virtual service are applied to control how specific fields are merged

The merging process happens during resource building, before the configuration is sent to Envoy. The template is applied to the virtual service's spec, and then the resulting configuration is used to build the Envoy resources.

## Template options

Template options allow you to control how specific fields from the template are handled when merging with the virtual service configuration. There are three modifiers available:

- **merge** (default) - Merges object fields, overrides primitive types in existing objects, merges lists by appending items
- **replace** - Completely replaces objects or lists instead of merging them
- **delete** - Deletes a field by key (does not work for list elements)

Each template option specifies a field path and a modifier. The field path identifies the field to apply the modifier to, and the modifier determines how the field is handled during merging.

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

## Best practices

1. Use templates for common configurations that are shared across multiple virtual services
2. Keep templates focused on a specific use case (e.g., HTTPS configuration, API gateway configuration)
3. Use descriptive names for templates to make their purpose clear
4. Document the expected fields that virtual services should provide when using the template
5. Use template options to customize the merging behavior when needed
6. Test your templates with different virtual service configurations to ensure they work as expected
