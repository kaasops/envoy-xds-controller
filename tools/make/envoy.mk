# This is a wrapper to create and cleanup the kind cluster
#
# All jobs starting kind cluster with name "exc"

ENVOY_VERSION_1_30 := v1.30.2
ENVOY_VERSION_1_29 := v1.29.5
ENVOY_VERSION_1_28 := v1.28.4
ENVOY_VERSION_1_27 := v1.27.6

.PHONY: envoy-1.30
envoy-1.30:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/install-envoy-in-kubernetes.sh -v $(ENVOY_VERSION_1_30)

.PHONY: envoy-1.29
envoy-1.29:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/install-envoy-in-kubernetes.sh -v $(ENVOY_VERSION_1_29)

.PHONY: envoy-1.28
envoy-1.28:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/install-envoy-in-kubernetes.sh -v $(ENVOY_VERSION_1_28)

.PHONY: envoy-1.27
envoy-1.27:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/install-envoy-in-kubernetes.sh -v $(ENVOY_VERSION_1_27)

.PHONY: cleanup-envoy
cleanup-envoy:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/install-envoy-in-kubernetes-cleanup.sh