# This is a wrapper to create and cleanup the kind cluster
#
# All jobs starting kind cluster with name "exc"

ENVOY_VERSION_1_30 := v1.30.2
ENVOY_VERSION_1_31 := v1.31.0

.PHONY: envoy-1.30
envoy-1.30:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/install-envoy-in-kubernetes.sh -v $(ENVOY_VERSION_1_30)

.PHONY: envoy-1.31
envoy-1.31:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/install-envoy-in-kubernetes.sh -v $(ENVOY_VERSION_1_31)

define CLEANUP_ENVOY
    echo "Cleanup Envoy"
    $(ROOT_DIR)/tools/install-envoy-in-kubernetes-cleanup.sh
endef