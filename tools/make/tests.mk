# Wrapper for running tests

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

# conformance tests
.PHONY: conformance-1.29
conformance-1.29: kind-with-registry-1.29 image.build-local image.push-local kube-deploy-local #run-conformance #cleanup-kind-with-registry

.PHONY: run-conformance
run-conformance:
	@$(LOG_TARGET)
	go test -v -tags conformance ./test/conformance