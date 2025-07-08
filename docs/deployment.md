# Deployment Guide: Envoy XDS Controller

This document provides instructions for deploying the Envoy XDS Controller in a Kubernetes environment.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Deployment Options](#deployment-options)
3. [Helm Deployment](#helm-deployment)
4. [Manual Deployment](#manual-deployment)
5. [Verifying the Deployment](#verifying-the-deployment)
6. [Upgrading](#upgrading)
7. [Uninstalling](#uninstalling)

## Prerequisites

Before deploying the Envoy XDS Controller, ensure you have the following:

- Kubernetes cluster v1.11.3+
- kubectl v1.11.3+
- Helm v3+ (for Helm deployment)
- Docker v17.03+ (for building custom images)
- Go v1.22.0+ (for development)

## Deployment Options

There are two main ways to deploy the Envoy XDS Controller:

1. **Helm Deployment**: Recommended for most users, provides easy configuration and upgrades
2. **Manual Deployment**: Using kubectl and the generated YAML bundle

## Helm Deployment

### Add the Helm Repository

```bash
# If using a custom repository
helm repo add envoy-xds-controller <repository-url>
helm repo update
```

### Install with Default Configuration

```bash
helm install envoy-xds-controller \
  --namespace envoy-xds-controller \
  --create-namespace \
  helm/charts/envoy-xds-controller
```

### Install with Custom Configuration

Create a `values.yaml` file with your custom configuration:

```yaml
replicaCount: 2

xds:
  port: 9000

auth:
  enabled: true
  oidc:
    clientId: "envoy-xds-controller"
    issuerUrl: "https://your-identity-provider"
    scope: "openid profile groups"
    redirectUri: "https://your-ui-url/callback"

ui:
  enabled: true
  ingress:
    enabled: true
    hosts:
      - host: envoy-xds-controller-ui.example.com
        paths:
          - path: /
            pathType: Prefix
```

Then install with your custom values:

```bash
helm install envoy-xds-controller \
  --namespace envoy-xds-controller \
  --create-namespace \
  -f values.yaml \
  helm/charts/envoy-xds-controller
```

## Manual Deployment

### Build the Installer

Build the installer YAML bundle:

```bash
make build-installer IMG=<registry>/envoy-xds-controller:<tag>
```

This generates an `install.yaml` file in the `dist` directory.

### Deploy Using the Installer

```bash
kubectl apply -f dist/install.yaml
```

Alternatively, you can use the published installer:

```bash
kubectl apply -f https://raw.githubusercontent.com/<org>/envoy-xds-controller/<tag>/dist/install.yaml
```

## Verifying the Deployment

Check that the controller is running:

```bash
kubectl get pods -n envoy-xds-controller
```

Verify the CRDs are installed:

```bash
kubectl get crds | grep envoy.kaasops.io
```

## Upgrading

### Upgrading with Helm

```bash
helm upgrade envoy-xds-controller \
  --namespace envoy-xds-controller \
  -f values.yaml \
  helm/charts/envoy-xds-controller
```

### Upgrading Manual Deployment

```bash
make build-installer IMG=<registry>/envoy-xds-controller:<new-tag>
kubectl apply -f dist/install.yaml
```

## Uninstalling

### Uninstalling Helm Deployment

```bash
helm uninstall envoy-xds-controller -n envoy-xds-controller
```

### Uninstalling Manual Deployment

```bash
kubectl delete -f dist/install.yaml
```

### Removing CRDs and Resources

To completely remove all resources, including CRDs and their instances:

```bash
# Delete instances of custom resources
kubectl delete -k config/samples/

# Delete the CRDs
make uninstall

# Undeploy the controller
make undeploy
```