# envoy-xds-controller

A Kubernetes-native control plane for Envoy proxies that provides dynamic configuration management through the xDS API.

## Description

Envoy xDS Controller is a Kubernetes controller that manages Envoy proxy configurations through the xDS API. It allows defining Envoy configurations as Kubernetes Custom Resources (CRs) and automatically transforms them into Envoy configurations, which are delivered to proxies via the xDS protocol in real-time.

Key features:
- Full support for Envoy xDS v3 API (LDS, RDS, CDS, SDS)
- Kubernetes-native integration with controller-runtime
- Dynamic configuration updates without proxy restarts
- Authentication and authorization with OIDC and RBAC
- Templating system for configuration reuse
- Web UI for configuration management

## Documentation

| Document | Description |
|----------|-------------|
| [Overview](docs/overview.md) | Project overview and concepts |
| [Architecture](docs/architecture.md) | Internal architecture and components |
| [xDS Server](docs/xds.md) | xDS implementation details |
| [Configuration](docs/configuration.md) | Configuration options |
| [Templates](docs/templates.md) | VirtualServiceTemplate usage |
| [TLS](docs/tls.md) | TLS configuration |
| [Deployment](docs/deployment.md) | Deployment guide |
| [Development](docs/development.md) | Development setup |
| [Testing](docs/testing.md) | Testing guide |
| [Troubleshooting](docs/troubleshooting.md) | Common issues and solutions |

## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+
- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster

### Installation

```sh
helm repo add envoy-xds-controller https://kaasops.github.io/envoy-xds-controller
helm install envoy-xds-controller envoy-xds-controller/envoy-xds-controller
```

With custom values:
```sh
helm install envoy-xds-controller envoy-xds-controller/envoy-xds-controller \
  --set image.tag=latest \
  --set ui.enabled=true
```

### Uninstall

```sh
helm uninstall envoy-xds-controller
```

## Contributing

We welcome contributions to the Envoy xDS Controller project! Here's how you can contribute:

1. **Code Contributions**:
   - Fork the repository
   - Create a feature branch (`git checkout -b feature/amazing-feature`)
   - Commit your changes (`git commit -m 'Add some amazing feature'`)
   - Push to the branch (`git push origin feature/amazing-feature`)
   - Open a Pull Request

2. **Bug Reports and Feature Requests**:
   - Use the GitHub issue tracker to report bugs or request features

3. **Development Environment**:
   - See the [development documentation](docs/development.md) for setting up your development environment

4. **Testing**:
   - Add tests for new features
   - Run existing tests with `make test`
   - Run end-to-end tests with `make test-e2e`

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
