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


# Conformance tests

.PHONY: conformance
conformance: conformance-1.29

.PHONY: conformance-1.29
conformance-1.29: prepare-kind kind-with-registry-1.29 image.build-local image.push-local kube-deploy-local run-conformance cleanup-kind-with-registry

.PHONY: run-conformance
run-conformance:
	@$(LOG_TARGET)
	go test -v -tags conformance ./test/conformance

# E2E tests

.PHONY: e2e
e2e: e2e-1.29

.PHONY: e2e-1.29
e2e-1.29: prepare-kind kind-with-registry-1.29 image.build-local image.push-local kube-deploy-local envoy-1.30 run-e2e cleanup-kind-with-registry

.PHONY: run-e2e
run-e2e:
	@$(LOG_TARGET)
	go test -v -tags e2e ./test/e2e
