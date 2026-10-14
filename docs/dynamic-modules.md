# Envoy Dynamic Modules

Envoy can load extensions from a shared library (`.so` file) at runtime. Such a library is called a dynamic module. This document shows how to load a module by its file path with the controller CRs.

## Requirements

| Requirement | Value | Notes |
|-------------|-------|-------|
| **Envoy** | 1.38 or newer | The `dynamic_module_config.module` field appeared in Envoy 1.38 |
| **Controller** | Built with go-control-plane `envoy` v1.38 or newer | Check `go.mod`. An older controller rejects the CR with `unknown field "module"` |
| **Module file** | Present on every host that runs Envoy | Envoy reads the file from its own file system |
| **Module build** | Same CPU architecture as Envoy, glibc not newer than in the Envoy image | A module built with the SDK for Envoy X.Y works with Envoy X.Y and X.(Y+1), see the [Envoy docs](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/advanced/dynamic_modules) |

## Load a Module by Path

Set the path in `dynamic_module_config.module.local.filename`.

Envoy can also find a module by `name` in the directory from the `ENVOY_DYNAMIC_MODULES_SEARCH_PATH` environment variable. Prefer the path: it is written in the CR and does not depend on the environment of the Envoy process. If both are set, Envoy uses `module`.

Envoy loads the module when the new configuration arrives, so you do not need to restart Envoy. Copy the file to the hosts first, then apply the CRs.

### HTTP Filter

Put the module in an HttpFilter:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: HttpFilter
metadata:
  name: my-module
spec:
  - name: envoy.filters.http.dynamic_modules
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.filters.http.dynamic_modules.v3.DynamicModuleFilter
      dynamic_module_config:
        module:
          local:
            filename: /opt/envoy-modules/libmy_module.so
      filter_name: my_filter
      filter_config:
        "@type": type.googleapis.com/google.protobuf.StringValue
        value: "my settings"
```

`filter_name` selects a filter inside the module. `filter_config` is passed to that filter. A `google.protobuf.StringValue` reaches the module as a plain string.

Then reference the HttpFilter from a VirtualService:

```yaml
spec:
  additionalHttpFilters:
    - name: my-module
```

### Listener Filter

Put the module in `listener_filters` of a Listener:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: Listener
metadata:
  name: https
spec:
  name: https
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 10443
  listener_filters:
    - name: envoy.filters.listener.dynamic_modules
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.filters.listener.dynamic_modules.v3.DynamicModuleListenerFilter
        dynamic_module_config:
          module:
            local:
              filename: /opt/envoy-modules/libmy_module.so
        filter_name: my_listener_filter
```

The controller sends `listener_filters` to Envoy as they are written in the Listener.

Do not add `filter_chains` to a Listener whose VirtualServices use HTTP filters, TLS or a virtual host. The controller treats such a Listener as having its own filter chains and rejects these VirtualService settings with a conflict.

## Modules Written in Go

Modules built with the Envoy Go SDK need two extra settings. Modules written in Rust or C++ need neither.

**Run Envoy with `GODEBUG=cgocheck=0`.** The Go SDK passes Go pointers to Envoy, and the Go runtime check stops the process because of that. Without the variable, Envoy aborts as soon as it loads the module configuration:

```
panic: runtime error: .../sdk/go/abi/internal.go:1929: result of Go function
envoy_dynamic_module_on_http_filter_config_new called from cgo is unpinned Go unsafe pointer ...
```

This is the message with Go 1.26. With Go 1.25 the message is `panic: runtime error: cgo result is unpinned Go pointer or points to unpinned Go pointer`.

The whole Envoy process stops, not only this filter. After a restart Envoy gets the same configuration and stops again, until the module is removed from the CRs. The Envoy maintainers recommend this setting ([envoyproxy/envoy#44123](https://github.com/envoyproxy/envoy/issues/44123)).

The variable only changes the Go code inside the modules. Envoy itself is written in C++ and ignores it.

For a systemd unit:

```ini
[Service]
Environment=GODEBUG=cgocheck=0
```

For a Kubernetes container:

```yaml
env:
  - name: GODEBUG
    value: cgocheck=0
```

**Set `do_not_close: true`** in `dynamic_module_config`. The Go runtime cannot be unloaded from a running process.

## Verification

Check the Envoy configuration with the admin endpoint `/config_dump`. The listener must contain the filter with your `filter_name` and `dynamic_module_config.module.local.filename`.

## Troubleshooting

| Error | Cause | Fix |
|-------|-------|-----|
| Webhook: `unknown field "module"` | The controller is built with go-control-plane `envoy` older than v1.38 | Update the controller |
| Envoy: `Failed to load dynamic module: ... cannot open shared object file: No such file or directory` | There is no file at this path on the Envoy host | Copy the file or fix the path |
| Envoy aborts, the log has `panic: runtime error:` with `unpinned Go pointer` or `unpinned Go unsafe pointer` | A Go module runs without `GODEBUG=cgocheck=0` | Set the variable for Envoy, see [Modules Written in Go](#modules-written-in-go) |

## Testing

The e2e test `test/e2e/envoy_dynamic_module_test.go` builds a small module with the Go SDK (`test/testdata/e2e/dynamic_module/module`). It loads the module by path in two places:

- a Listener, as a listener filter that counts accepted connections in an Envoy counter;
- an HttpFilter, as an HTTP filter that sets a response header.

The test checks that both filters work and that Envoy was not restarted. It runs only with Envoy 1.38 or newer.
