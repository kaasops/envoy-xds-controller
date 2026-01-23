# Development Guide: Envoy XDS Controller

This document provides guidelines and instructions for developing the Envoy XDS Controller.

## Table of Contents

1. [Development Environment Setup](#development-environment-setup)
   - [Quick Start with Kind (Recommended)](#quick-start-with-kind-recommended)
   - [Running Locally Without Webhook](#running-locally-without-webhook)
   - [Running Locally With Webhook (Advanced)](#running-locally-with-webhook-advanced)
2. [Project Structure](#project-structure)
3. [Building](#building)
4. [Testing](#testing)
5. [Debugging](#debugging)
6. [Code Style and Conventions](#code-style-and-conventions)
7. [Adding New Features](#adding-new-features)

## Development Environment Setup

### Prerequisites

- Go v1.24.0+
- Docker v17.03+
- kubectl v1.11.3+
- Access to a Kubernetes cluster v1.11.3+
- Git

### Setting Up Local Development Environment

1. Clone the repository:

```bash
git clone https://github.com/your-org/envoy-xds-controller.git
cd envoy-xds-controller
```

2. Install dependencies:

```bash
go mod download
```

3. Install development tools:

```bash
make install-tools
```

### Quick Start with Kind (Recommended)

The easiest way to run the full development environment is using Kind with `make dev`:

1. Create a Kind cluster with local registry:

```bash
make kr
```

2. Run the interactive development setup:

```bash
make dev
```

This will prompt you with options (defaults shown in brackets):

```
Enable UI? [Y/n]:
Enable Auth (OIDC via Dex)? [Y/n]:
Install Prometheus Operator? [y/N]:
Development mode (verbose logging)? [Y/n]:
Deploy test Envoy proxy? [Y/n]:
Apply test resources (VirtualServices, etc.)? [y/N]:
```

Press Enter to accept defaults. The script will build images, push to local registry, and deploy via Helm.

3. **Important: Configure /etc/hosts for Dex authentication**

If you enabled Auth (Dex), add the following to your `/etc/hosts`:

```bash
echo "127.0.0.1 dex.dex" | sudo tee -a /etc/hosts
```

This is required because the OIDC issuer URL uses `dex.dex` as the hostname.

4. Start port-forwards (the script will show these commands):

```bash
# UI (in terminal 1)
kubectl -n envoy-xds-controller port-forward svc/exc-envoy-xds-controller-ui 8080:8080

# Dex - required for auth (in terminal 2)
kubectl -n dex port-forward svc/dex 5556:5556

# Envoy proxy (in terminal 3)
kubectl -n default port-forward svc/envoy 10080:80 10443:443 19000:19000
```

5. Access the UI at http://localhost:8080

Test credentials (if Auth enabled):
- Admin: `admin@example.com` / `admin`
- User: `user@example.com` / `user`

#### Useful Development Commands

```bash
# Check status of all components
make dev-status

# Start all port-forwards in one terminal
make dev-port-forward

# View logs
make dev-logs          # Controller logs
make dev-logs-ui       # UI logs

# Restart pods without rebuild
make dev-restart       # Restart controller
make dev-restart-ui    # Restart UI
make dev-restart-all   # Restart all

# Rebuild and redeploy
make dev-update            # Rebuild everything
make dev-update-backend    # Rebuild only controller
make dev-update-frontend   # Rebuild only UI

# Show test credentials
make dev-creds
```

### Running Locally Without Webhook

If you don't need the Validation Webhook for development, you can start the Envoy xDS Controller locally with:

```bash
export WEBHOOK_DISABLE=true
make run
```

### Running Locally With Webhook (Advanced)

For full installation with Validation Webhook logic on a local instance, you need Kubernetes with network access to your workstation. You can use [KIND](https://kind.sigs.k8s.io/) for this purpose.

1. Deploy Helm Envoy xDS Controller to your Kubernetes cluster:

```bash
cd helm/charts/envoy-xds-controller
helm upgrade envoy --install --namespace envoy-xds-controller --create-namespace .
```

2. Wait for the Pod to start, then set Replicas for Envoy xDS Controller to 0:

```bash
kubectl scale deployment -n envoy-xds-controller envoy-envoy-xds-controller --replicas 0
```

3. Create a directory for local certificates for the Webhook Server:

```bash
mkdir -p /tmp/k8s-webhook-server/serving-certs
```

4. Copy the generated certificate and key for the Webhook Server:

```bash
kubectl get secrets -n envoy-xds-controller envoy-xds-controller-tls -o jsonpath='{.data.tls\.crt}' | base64 -D > /tmp/k8s-webhook-server/serving-certs/tls.crt
kubectl get secrets -n envoy-xds-controller envoy-xds-controller-tls -o jsonpath='{.data.tls\.key}' | base64 -D > /tmp/k8s-webhook-server/serving-certs/tls.key
```

5. Delete the service for the Webhook:

```bash
kubectl delete service -n envoy-xds-controller envoy-xds-controller-webhook-service
```

6. Apply a new service with your workstation's IP:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: envoy-xds-controller-webhook-service
  namespace: envoy-xds-controller
spec:
  ports:
    - protocol: TCP
      port: 443
      targetPort: 9443
---
apiVersion: v1
kind: Endpoints
metadata:
  name: envoy-xds-controller-webhook-service
  namespace: envoy-xds-controller
subsets:
  - addresses:
      - ip: <WORKSTATION_IP>  # Replace with your IP
    ports:
      - port: 9443
```

7. Run the controller locally:

```bash
make run
```

## Project Structure

The project follows a standard Go project layout:

- `api/`: API definitions and generated code
- `cmd/`: Application entry points
- `config/`: Kubernetes manifests and configuration
- `docs/`: Documentation
- `helm/`: Helm charts for deployment
- `internal/`: Internal packages
  - `xds/`: xDS server implementation
  - `cache/`: Cache implementation
  - `updater/`: Configuration updaters
  - `grpcapi/`: gRPC API implementation
- `pkg/`: Public packages
- `proto/`: Protocol buffer definitions
- `test/`: Test code and e2e tests
- `ui/`: Web UI code

## Building

### Building the Controller

```bash
make build
```

### Building Docker Images

```bash
make docker-build IMG=<registry>/envoy-xds-controller:<tag>
```

### Building the UI

```bash
cd ui
npm install
npm run build
```

### Building the Installer

```bash
make build-installer IMG=<registry>/envoy-xds-controller:<tag>
```

## Testing

### Running Unit Tests

```bash
make test
```

### Running End-to-End Tests

```bash
make test-e2e
```

### Running Linters

```bash
make lint
```

### Fixing Lint Issues

```bash
make lint-fix
```

## Debugging

### Enabling Debug Logs

Set the `LOG_LEVEL` environment variable to `debug`:

```bash
export LOG_LEVEL=debug
make run
```

### Using Delve for Debugging

```bash
dlv debug ./cmd/main.go
```

## Code Style and Conventions

- Follow standard Go code style and conventions
- Use `gofmt` to format code
- Document all exported functions, types, and constants
- Write unit tests for all functionality
- Use meaningful variable and function names

## Adding New Features

1. Create a new branch for your feature
2. Implement the feature with appropriate tests
3. Update documentation
4. Submit a pull request

