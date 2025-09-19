# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Envoy xDS Controller is a Kubernetes-native control plane for Envoy proxies that provides dynamic configuration management through the xDS API. It transforms Kubernetes Custom Resources into Envoy configurations delivered via xDS protocol in real-time.

## Key Commands

### Development
```bash
make run                # Run controller locally (set WEBHOOK_DISABLE=true to skip webhook)
make build              # Build manager binary
make generate           # Generate DeepCopy methods
make manifests          # Generate CRDs and RBAC
make fmt                # Format Go code
make vet                # Run go vet
make lint               # Run golangci-lint
make lint-fix           # Run golangci-lint with auto-fix
```

### Testing
```bash
make test               # Run unit tests (uses Ginkgo/Gomega with envtest)
make test-e2e           # Run e2e tests (requires kind cluster)

# Run specific test
go test ./path/to/package -run TestName

# Run benchmarks
go test -bench=. ./internal/xds/resbuilder_v2/
```

### Local Development with Kind
```bash
make kr                 # Create kind cluster with local registry
make dev-local          # Build and deploy everything to local cluster
make dev-backend        # Deploy backend only
make dev-frontend       # Run frontend dev server
```

### Building and Deployment
```bash
make docker-build IMG=<registry>/<image>:<tag>      # Build controller image
make docker-build-ui IMG=<registry>/<image>:<tag>   # Build UI image
make docker-build-all                                # Build all images
make docker-push IMG=<registry>/<image>:<tag>       # Push images
make deploy IMG=<registry>/<image>:<tag>            # Deploy to cluster
make helm-deploy-local                               # Deploy using Helm
```

### Protocol Buffers
```bash
make bufgen             # Generate protobuf code using buf
```

## Architecture

### Core Components
1. **Controllers** (`/internal/controller/`) - Kubernetes controllers for each CRD
2. **xDS Server** (`/internal/xds/`) - Implements Envoy xDS API
3. **Resource Builders** (`/internal/xds/resbuilder_v2/`) - Transform CRs to xDS resources
4. **Store** (`/internal/store/`) - In-memory resource storage
5. **Cache** (`/internal/cache/`) - xDS snapshot cache
6. **Webhooks** (`/internal/webhook/`) - Admission webhook validators

### Custom Resources (CRDs)
- **Cluster** - Upstream clusters for Envoy
- **Listener** - Network listeners (ports/protocols)
- **Route** - Routing rules
- **VirtualService** - Complete service configuration
- **VirtualServiceTemplate** - Reusable templates
- **AccessLogConfig** - Access logging
- **HttpFilter** - HTTP filter chains
- **Policy** - Security/access policies

### Key Patterns

1. **Controller Pattern**: Uses controller-runtime with reconciliation loops
2. **Resource Building**: Two-phase process - collect resources, then build xDS
3. **Caching**: LRU cache with TTL for performance
4. **Testing**: Ginkgo BDD tests with envtest for controllers
5. **Validation**: Webhook validators for all CRDs

### ResBuilder V2 Optimization

The project is optimizing resource builders with:
- Modular architecture (adapters, transformers, builders)
- Object pools to reduce allocations
- Direct protobuf access for performance
- Comprehensive benchmarking infrastructure

Key packages:
- `/internal/xds/resbuilder_v2/adapter/` - Resource adapters
- `/internal/xds/resbuilder_v2/transformer/` - Resource transformers
- `/internal/xds/resbuilder_v2/builder/` - Main builders
- `/internal/xds/resbuilder_v2/core/` - Core interfaces

## Development Tips

1. **Local Development with Webhook**: Follow `/docs/development.md` for certificate setup
2. **Testing**: Always run `make test` before committing
3. **Linting**: Project uses strict linting - run `make lint-fix` to auto-fix issues
4. **Generated Code**: Run `make generate manifests` after API changes
5. **Benchmarking**: Use comparison tests in resbuilder_v2 when optimizing

## Important Files

- `/Makefile` - All build/test/deploy commands
- `/config/` - Kubernetes manifests and Kustomization
- `/helm/charts/envoy-xds-controller/` - Helm chart
- `/.golangci.yml` - Linting configuration
- `/buf.yaml` & `/buf.gen.yaml` - Protobuf configuration
- `/PROJECT` - Kubebuilder metadata