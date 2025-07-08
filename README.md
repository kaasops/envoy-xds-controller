# envoy-xds-controller

A Kubernetes-native control plane for Envoy proxies that provides dynamic configuration management through the xDS API.

## Description

Envoy xDS Controller is a Kubernetes controller that manages Envoy proxy configurations through the xDS API. It allows defining Envoy configurations as Kubernetes Custom Resources (CRs) and automatically transforms them into Envoy configurations, which are delivered to proxies via the xDS protocol in real-time.

Key features:
- Full support for Envoy xDS v3 API (LDS, RDS, CDS, EDS)
- Kubernetes-native integration with controller-runtime
- Dynamic configuration updates without proxy restarts
- Authentication and authorization with OIDC and RBAC
- Templating system for configuration reuse
- Web UI for configuration management

## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/envoy-xds-controller:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/envoy-xds-controller:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/envoy-xds-controller:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/envoy-xds-controller/<tag or branch>/dist/install.yaml
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
   - Check [contributing guidelines](docs/contributing/development.md) for webhook setup

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
