# ==============================================================================
# Git Information
# ==============================================================================

GIT_COMMIT := $(shell git rev-parse --short HEAD)

# ==============================================================================
# Container Images
# ==============================================================================

# Image registries
REGISTRY       ?= docker.io/kaasops
LOCAL_REGISTRY ?= localhost:5001

# Image tag (defaults to git commit)
IMAGE_TAG ?= $(GIT_COMMIT)

# Image repositories
IMG_REPO           ?= $(REGISTRY)/envoy-xds-controller
UI_IMG_REPO        ?= $(REGISTRY)/envoy-xds-controller-ui
INIT_CERT_IMG_REPO ?= $(REGISTRY)/envoy-xds-controller-init-cert

# Full image references
IMG           ?= $(IMG_REPO):$(IMAGE_TAG)
UI_IMG        ?= $(UI_IMG_REPO):$(IMAGE_TAG)
INIT_CERT_IMG ?= $(INIT_CERT_IMG_REPO):$(IMAGE_TAG)

# Container runtime (docker or podman)
CONTAINER_TOOL ?= docker

# ==============================================================================
# Versions
# ==============================================================================

PROM_OPERATOR_VERSION    ?= v0.77.1
KUSTOMIZE_VERSION        ?= v5.5.0
CONTROLLER_TOOLS_VERSION ?= v0.16.4
ENVTEST_VERSION          ?= release-0.19
ENVTEST_K8S_VERSION      ?= 1.31.0
GOLANGCI_LINT_VERSION    ?= v1.64.8

# ==============================================================================
# Deployment
# ==============================================================================

DEPLOY_TIMEOUT ?= 5m
HELM_REPO_URL  ?= https://kaasops.github.io/envoy-xds-controller/helm

# ==============================================================================
# Build Tools
# ==============================================================================

LOCALBIN       ?= $(shell pwd)/bin
KUBECTL        ?= kubectl
KUSTOMIZE      ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST        ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT  ?= $(LOCALBIN)/golangci-lint

# Go binary path
ifeq (,$(shell go env GOBIN))
GOBIN := $(shell go env GOPATH)/bin
else
GOBIN := $(shell go env GOBIN)
endif

# ==============================================================================
# Shell Configuration
# ==============================================================================

SHELL := /usr/bin/env bash -o pipefail
.SHELLFLAGS := -ec

.DEFAULT_GOAL := help

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: version
version: ## Show version information
	@echo "Tag: $(IMAGE_TAG)"
	@echo "Commit: $(GIT_COMMIT)"
	@echo "Registry: $(REGISTRY)"
	@echo "Main Image: $(IMG)"
	@echo "UI Image: $(UI_IMG)"
	@echo "Init-Cert Image: $(INIT_CERT_IMG)"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf dist/
	@rm -f cover.out
	@echo "✅ Build artifacts cleaned"

##@ Development

.PHONY: deps-update
deps-update: ## Update Go dependencies
	go get -u ./...
	go mod tidy

.PHONY: deps-verify
deps-verify: ## Verify Go dependencies
	go mod verify
	go mod tidy

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=helm/charts/envoy-xds-controller/crds

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# TODO(user): To use a different vendor for e2e tests, modify the setup under 'tests/e2e'.
# The default setup assumes Kind is pre-installed and builds/loads the Manager Docker image locally.
# Prometheus and CertManager are installed by default; skip with:
# - PROMETHEUS_INSTALL_SKIP=true
# - CERT_MANAGER_INSTALL_SKIP=true
.PHONY: test-e2e
test-e2e: manifests generate fmt vet ## Run the e2e tests. Expected an isolated environment using Kind.
	@command -v kind >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	@kind get clusters | grep -q 'kind' || { \
		echo "No Kind cluster is running. Please start a Kind cluster before running the e2e tests."; \
		exit 1; \
	}
	go test ./test/e2e/ -v -ginkgo.v -timeout 15m

E2E_REPORTS_DIR ?= .e2e-reports

.PHONY: test-e2e-report
test-e2e-report: manifests generate fmt vet ## Run the e2e tests with report saved to .e2e-reports directory.
	@command -v kind >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	@kind get clusters | grep -q 'kind' || { \
		echo "No Kind cluster is running. Please start a Kind cluster before running the e2e tests."; \
		exit 1; \
	}
	@mkdir -p $(E2E_REPORTS_DIR)
	@REPORT_FILE="$(E2E_REPORTS_DIR)/e2e-test-$$(date +%Y%m%d-%H%M%S).log"; \
	echo "Running e2e tests with report saved to $$REPORT_FILE"; \
	go test ./test/e2e/ -v -ginkgo.v -timeout 15m -ginkgo.no-color 2>&1 | tee $$REPORT_FILE; \
	TEST_EXIT_CODE=$${PIPESTATUS[0]}; \
	echo ""; \
	echo "Report saved to: $$REPORT_FILE"; \
	exit $$TEST_EXIT_CODE

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager \
		-ldflags "-X github.com/kaasops/envoy-xds-controller/internal/buildinfo.Version=$(IMAGE_TAG) \
		-X github.com/kaasops/envoy-xds-controller/internal/buildinfo.CommitHash=$(GIT_COMMIT) \
		-X github.com/kaasops/envoy-xds-controller/internal/buildinfo.BuildDate=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")" \
		cmd/main.go

.PHONY: build-init-cert
build-init-cert: manifests generate fmt vet ## Build init-cert binary.
	go build -o bin/init-cert cmd/init-cert/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build \
		--build-arg VERSION=$(IMAGE_TAG) \
		--build-arg COMMIT_HASH=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		-t ${IMG} .

.PHONY: docker-build-ui
docker-build-ui: ## Build docker image with the ui.
	$(CONTAINER_TOOL) build -t ${UI_IMG} -f ui/Dockerfile ./ui

.PHONY: docker-build-init-cert
docker-build-init-cert: ## Build docker image with the init-cert.
	$(CONTAINER_TOOL) build -t ${INIT_CERT_IMG} -f cmd/init-cert/Dockerfile .

.PHONY: docker-build-all
docker-build-all: docker-build docker-build-ui docker-build-init-cert ## Build all docker images

.PHONY: docker-cache-clear
docker-cache-clear: ## Clear Docker build cache
	@echo "Clearing Docker build cache..."
	@docker builder prune -af
	@echo "Docker build cache cleared!"

.PHONY: docker-cache-info
docker-cache-info: ## Show Docker cache and images information
	@echo "=== Docker system info ==="
	@docker system df
	@echo ""
	@echo "=== Docker BuildKit cache ==="
	@docker buildx du 2>/dev/null || echo "No buildx cache found"
	@echo ""
	@echo "=== Envoy XDS Controller images ==="
	@docker images --filter=reference='*envoy-xds-controller*' --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}\t{{.CreatedSince}}"

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

.PHONY: docker-push-ui
docker-push-ui: ## Push docker image with the ui
	$(CONTAINER_TOOL) push ${UI_IMG}

.PHONY: docker-push-init-cert
docker-push-init-cert: ## Push docker image with the init-cert
	$(CONTAINER_TOOL) push ${INIT_CERT_IMG}

.PHONY: docker-push-all
docker-push-all: docker-push docker-push-ui docker-push-init-cert ## Push all docker images

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name envoy-xds-controller-builder
	$(CONTAINER_TOOL) buildx use envoy-xds-controller-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm envoy-xds-controller-builder
	rm Dockerfile.cross

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > dist/install.yaml

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

##@ Helm

.PHONY: helm-lint
helm-lint: ## Lint Helm chart
	helm lint helm/charts/envoy-xds-controller

.PHONY: helm-package
helm-package: ## Package Helm chart
	helm package helm/charts/* -d helm/packages

.PHONY: helm-index
helm-index: ## Update Helm repo index
	helm repo index --url $(HELM_REPO_URL) ./helm

.PHONY: helm-template
helm-template: ## Render Helm chart templates locally
	helm template exc -n envoy-xds-controller ./helm/charts/envoy-xds-controller/

##@ Local Development

.PHONY: dev
dev: ## Interactive local development setup in Kind
	@bash scripts/dev.sh

.PHONY: dev-update
dev-update: ## Rebuild and redeploy to existing dev cluster
	@bash scripts/dev-update.sh

.PHONY: dev-clean
dev-clean: ## Remove Helm release from Kind cluster
	helm uninstall exc -n envoy-xds-controller || true

.PHONY: kr
kr: ## Create Kind cluster with local registry
	bash scripts/kind-with-registry.sh

.PHONY: kd
kd: ## Delete Kind cluster
	kind delete cluster

.PHONY: dev-apply-resources
dev-apply-resources: ## Apply test resources to cluster
	kubectl -n envoy-xds-controller apply -f dev/testdata/common

.PHONY: dev-delete-resources
dev-delete-resources: ## Delete test resources from cluster
	kubectl -n envoy-xds-controller delete -f dev/testdata/common

.PHONY: dev-auth
dev-auth: ## Setup Dex + LDAP for authentication testing
	bash scripts/dev-auth.sh

.PHONY: dev-creds
dev-creds: ## Show test credentials for auth
	@bash scripts/dev-creds.sh

.PHONY: dev-auth-generate
dev-auth-generate: ## Regenerate LDAP config from users.yaml
	@bash scripts/ldap/generate.sh

.PHONY: dev-envoy
dev-envoy: ## Deploy test Envoy instance
	kubectl apply -f dev/envoy

.PHONY: dev-frontend
dev-frontend: ## Run UI development server (npm run dev)
	cd ui && npm run dev

.PHONY: install-prometheus
install-prometheus: ## Install Prometheus Operator
	kubectl create -f https://github.com/prometheus-operator/prometheus-operator/releases/download/$(PROM_OPERATOR_VERSION)/bundle.yaml

.PHONY: uninstall-prometheus
uninstall-prometheus: ## Uninstall Prometheus Operator
	kubectl delete -f https://github.com/prometheus-operator/prometheus-operator/releases/download/$(PROM_OPERATOR_VERSION)/bundle.yaml

##@ E2E Testing

.PHONY: deploy-e2e
deploy-e2e: manifests ## Deploy controller for e2e tests
	helm install exc-e2e \
		--set metrics.address=:8443 \
		--set 'watchNamespaces={default,exc-secrets-ns1,exc-secrets-ns2}' \
		--set image.repository=$(IMG_REPO) \
		--set image.tag=$(IMAGE_TAG) \
		--set initCert.image.repository=$(INIT_CERT_IMG_REPO) \
		--set initCert.image.tag=$(IMAGE_TAG) \
		--set cacheAPI.enabled=true \
		--set resourceAPI.enabled=true \
		--set development=true \
		--namespace envoy-xds-controller \
		--create-namespace ./helm/charts/envoy-xds-controller \
		--debug --timeout='$(DEPLOY_TIMEOUT)' --wait

.PHONY: undeploy-e2e
undeploy-e2e: ## Remove e2e test deployment
	helm uninstall -n envoy-xds-controller exc-e2e

##@ Code Generation

.PHONY: bufgen
bufgen: ## Generate protobuf code
	buf generate