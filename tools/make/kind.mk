# This is a wrapper to create and cleanup the kind cluster
#
# All jobs starting kind cluster with name "exc"

# This is a wrapper to create and cleanup the kind cluster
#
# All jobs starting kind cluster with name "exc"

KIND_VERSION_1_29 := v1.29.4
KIND_VERSION_1_30 := v1.30.2

.PHONY: kind-with-registry-1.29
kind-with-registry-1.29:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/kind-with-registry.sh -v $(KIND_VERSION_1_29)

.PHONY: kind-with-registry-1.30
kind-with-registry-1.30:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/kind-with-registry.sh -v $(KIND_VERSION_1_30)

.PHONY: cleanup-kind-with-registry
cleanup-kind-with-registry:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/kind-with-registry-cleanup.sh