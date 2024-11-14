WAIT_TIMEOUT ?= 15m

.PHONY: kube-manifests
kube-manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	@$(LOG_TARGET)
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=helm/charts/envoy-xds-controller/crds

.PHONY: kube-generate
kube-generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="tools/hack/boilerplate.go.txt" paths="./..."

.PHONY: kube-deploy
kube-deploy: kube-manifests ## Install Envoy xDS Controller into the Kubernetes cluster specified in ~/.kube/config.
	@$(LOG_TARGET)
	helm install exc --namespace envoy-xds-controller --create-namespace ./helm/charts/envoy-xds-controller --debug --timeout='$(WAIT_TIMEOUT)' --wait

.PHONY: kube-deploy-local
kube-deploy-local: kube-manifests ## Install Envoy xDS Controller into the local Kubernetes cluster specified in ~/.kube/config.
	@$(LOG_TARGET)
	helm install exc --set image.repository=$(LOCAL_IMAGE) --set image.tag=$(TAG) --namespace envoy-xds-controller --create-namespace ./helm/charts/envoy-xds-controller --debug --timeout='$(WAIT_TIMEOUT)' --wait

.PHONY: kube-deploy-with-ui-local
kube-deploy-with-ui-local: kube-manifests ## Install Envoy xDS Controller with UI into the local Kubernetes cluster specified in ~/.kube/config.
	@$(LOG_TARGET)
	helm install exc --set image.repository=$(LOCAL_IMAGE) --set image.tag=$(TAG) --set ui.enabled=true --set cacheAPI.enabled=true --set ui.image.repository=$(LOCAL_UI_IMAGE) --set ui.image.tag=$(TAG) --namespace envoy-xds-controller --create-namespace ./helm/charts/envoy-xds-controller --debug --timeout='$(WAIT_TIMEOUT)' --wait


.PHONY: kube-undeploy
kube-undeploy: kube-manifests ## Uninstall the Envoy xDS Controller from the Kubernetes cluster specified in ~/.kube/config.
	@$(LOG_TARGET)
	helm uninstall exc -n envoy-xds-controller


##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.15.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
