# Overview Guide: Envoy XDS Controller

This document provides a high-level overview of the Envoy XDS Controller project, its purpose, features, and components.

## Table of Contents

1. [Purpose](#purpose)
2. [Key Features](#key-features)
3. [Use Cases](#use-cases)
4. [Components at a Glance](#components-at-a-glance)
5. [Project Status](#project-status)

## Purpose

`envoy-xds-controller` is a Kubernetes-native control plane component and xDS server designed to dynamically manage the configuration of Envoy proxies based on cluster state and user-defined APIs.

It enables centralized, declarative control of routing, service discovery, and security policies for Envoy without manual editing of configuration files.

---

## Key Features

- 📡 **XDS API (v3) support**:
    - Cluster Discovery Service (CDS)
    - Endpoint Discovery Service (EDS)
    - Listener Discovery Service (LDS)
    - Route Discovery Service (RDS)

- ⚙️ **Kubernetes-native integration**:
    - Watches native resources (e.g., Services, Endpoints, ConfigMaps)
    - Uses `controller-runtime` for reconciliation
    - Manages CRDs for custom behavior (if applicable)

- 🔐 **Authentication & Authorization**:
    - OIDC support via `go-oidc`
    - Role-based access control using Casbin

- 🧠 **Dynamic configuration updates**:
    - Real-time propagation of routing and cluster changes
    - Declarative configuration via CRDs or API

- 🧰 **Extensible architecture**:
    - Modular design using internal packages for `xds`, `cache`, `updater`
    - gRPC/HTTP API support via Connect RPC
    - Can be embedded into other control planes

---

## Use Cases

- Managing Envoy configuration in a dynamic microservices environment
- Implementing a custom service mesh control plane
- Enforcing fine-grained access control over xDS data

---

## Components at a Glance

- `cmd/`: application entry point and setup
- `internal/xds/`: core xDS server logic
- `pkg/api/`: gRPC service definitions
- `proto/`: Protocol Buffers
- `helm/`: deployment charts
- `docs/`: technical and user documentation

---

## Project Status

This project is actively developed and used in production-like environments. Contributions and feedback are welcome.
