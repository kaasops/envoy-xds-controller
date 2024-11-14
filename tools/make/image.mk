# This is a wrapper to build and push docker image
#
# All make targets related to docker image are defined in this file.

include tools/make/env.mk

DOCKER := docker
DOCKER_SUPPORTED_API_VERSION ?= 1.32

.PHONY: image.verify
image.verify:
	@$(LOG_TARGET)
	$(eval API_VERSION := $(shell $(DOCKER) version | grep -E 'API version: {1,6}[0-9]' | head -n1 | awk '{print $$3} END { if (NR==0) print 0}' ))
	$(eval PASS := $(shell echo "$(API_VERSION) > $(DOCKER_SUPPORTED_API_VERSION)" | bc))
	@if [ $(PASS) -ne 1 ]; then \
		$(DOCKER) -v ;\
		$(call log, Unsupported docker version. Docker API version should be greater than $(DOCKER_SUPPORTED_API_VERSION)); \
		exit 1; \
	fi

.PHONY: image.build
image.build: image.verify
	@$(LOG_TARGET)
	@$(call log, "Building image $(IMAGE):$(TAG) in linux/$(GOARCH)")
	$(eval BUILD_SUFFIX := --pull --load -t $(IMAGE):$(TAG) -f $(ROOT_DIR)/Dockerfile ./)
	@$(call log, "Creating image tag $(REGISTRY)/$(IMAGE):$(TAG) in linux/$(GOARCH)")
	$(DOCKER) buildx build --platform linux/$(GOARCH) $(BUILD_SUFFIX)

.PHONY: image.build-local
image.build-local: image.verify
	@$(LOG_TARGET)
	@$(call log, "Building image $(LOCAL_IMAGE):$(TAG) in linux/$(GOARCH)")
	$(eval BUILD_SUFFIX := --pull --load -t $(LOCAL_IMAGE):$(TAG) -f $(ROOT_DIR)/Dockerfile ./)
	@$(call log, "Creating image tag $(LOCAL_IMAGE):$(TAG) in linux/$(GOARCH)")
	$(DOCKER) buildx build --platform linux/$(GOARCH) $(BUILD_SUFFIX)

.PHONY: image.push
image.push: image.build
	@$(LOG_TARGET)
	$(eval COMMAND := $(word 1,$(subst ., ,$*)))
	@$(call log, "Pushing docker image tag $(IMAGE):$(TAG) in linux/$(GOARCH)")
	$(DOCKER) push $(IMAGE):$(TAG)

.PHONY: image.push-local
image.push-local: image.build-local
	@$(LOG_TARGET)
	$(eval COMMAND := $(word 1,$(subst ., ,$*)))
	@$(call log, "Pushing docker image tag $(LOCAL_IMAGE):$(TAG) in linux/$(GOARCH)")
	$(DOCKER) push $(LOCAL_IMAGE):$(TAG)

# ui

.PHONY: image.build-ui-local
image.build-ui-local:
	@$(LOG_TARGET)
	@$(call log, "Building image $(LOCAL_UI_IMAGE):$(TAG) in linux/$(GOARCH)")
	$(eval BUILD_SUFFIX := --pull --load -t $(LOCAL_UI_IMAGE):$(TAG) -f $(ROOT_DIR)/ui/Dockerfile ./ui)
	@$(call log, "Creating image tag $(LOCAL_UI_IMAGE):$(TAG) in linux/$(GOARCH)")
	$(DOCKER) buildx build --platform linux/$(GOARCH) $(BUILD_SUFFIX)

.PHONY: image.push-ui-local
image.push-ui-local: image.build-ui-local
	@$(LOG_TARGET)
	$(eval COMMAND := $(word 1,$(subst ., ,$*)))
	@$(call log, "Pushing docker image tag $(LOCAL_UI_IMAGE):$(TAG) in linux/$(GOARCH)")
	$(DOCKER) push $(LOCAL_UI_IMAGE):$(TAG)