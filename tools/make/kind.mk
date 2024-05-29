# This is a wrapper to create and cleanup the kind cluster
#
# All jobs starting kind cluster with name "exc"

KIND_VERSION_1_29 := v1.29.2
KIND_VERSION_1_28 := v1.28.7
KIND_VERSION_1_27 := v1.27.11
KIND_VERSION_1_26 := v1.26.14

.PHONY: kind-with-registry-1.29
kind-with-registry-1.29:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/kind-with-registry.sh -v $(KIND_VERSION_1_29)

.PHONY: kind-with-registry-1.28
kind-with-registry-1.28:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/kind-with-registry.sh -v $(KIND_VERSION_1_28)

.PHONY: kind-with-registry-1.27
kind-with-registry-1.27:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/kind-with-registry.sh -v $(KIND_VERSION_1_27)

.PHONY: kind-with-registry-1.26
kind-with-registry-1.26:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/kind-with-registry.sh -v $(KIND_VERSION_1_26)

.PHONY: cleanup-kind-with-registry
cleanup-kind-with-registry:
	@$(LOG_TARGET)
	$(ROOT_DIR)/tools/kind-with-registry-cleanup.sh