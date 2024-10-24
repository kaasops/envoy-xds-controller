# Wrapper for running tests

include tools/make/envoy.mk

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...
	
# unit tests
.PHONY: unit-test
unit-test: kube-manifests kube-generate fmt vet envtest ## Run unit tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out


### Conformance tests ###

# Conformance tests for Kubernetes 1.29
.PHONY: conformance-1.29
conformance-1.29: prepare-kind kind-with-registry-1.29 build-deploy-exc run-conformance cleanup-kind-with-registry

# Conformance tests for Kubernetes 1.30
.PHONY: conformance-1.30
conformance-1.30: prepare-kind kind-with-registry-1.30 build-deploy-exc run-conformance cleanup-kind-with-registry

.PHONY: run-conformance
run-conformance:
	@$(LOG_TARGET)
	go test -v -tags conformance -count=1 ./test/conformance

### E2E tests ###

# E2E tests for Envoy in Kubernetes 1.29
.PHONY: e2e-1.29
e2e-1.29: prepare-kind kind-with-registry-1.29 build-deploy-exc e2e-1.29-1.30 e2e-1.29-1.31 cleanup-kind-with-registry

# E2E test on Kubernetes 1.29 for Envoy 1.30
.PHONY: e2e-1.29-1.30
e2e-1.29-1.30: envoy-1.30
	$(RUN_E2E)
	$(CLEANUP_ENVOY)

# E2E test on Kubernetes 1.29 for Envoy 1.31
.PHONY: e2e-1.29-1.31
e2e-1.29-1.31: envoy-1.31
	$(RUN_E2E)
	$(CLEANUP_ENVOY)

# E2E tests for Envoy in Kubernetes 1.30
.PHONY: e2e-1.30
e2e-1.30: prepare-kind kind-with-registry-1.30 build-deploy-exc e2e-1.30-1.30 e2e-1.30-1.31 cleanup-kind-with-registry

# E2E test on Kubernetes 1.30 for Envoy 1.30
.PHONY: e2e-1.30-1.30
e2e-1.30-1.30: envoy-1.30
	$(RUN_E2E)
	$(CLEANUP_ENVOY)

# E2E test for Envoy 1.31 on Kubernetes 1.31
.PHONY: e2e-1.30-1.31
e2e-1.31-1.30: envoy-1.31
	$(RUN_E2E)
	$(CLEANUP_ENVOY)

define RUN_E2E
    echo "Run e2e tests"
    go test -v -tags e2e -count=1 ./test/e2e
endef

.PHONY: build-deploy-exc
build-deploy-exc: image.build-local image.push-local kube-deploy-local

.PHONY: run-local
run-local: prepare-kind kind-with-registry-1.30 build-deploy-exc envoy-1.30