# This is a wrapper to hold common environment variables used in other make wrappers
#
# This file does not contain any specific make targets.


# Docker variables

# REGISTRY is the image registry to use for build and push image targets.
REGISTRY ?= docker.io/kaasops

# LOCAL_REGISTRY is the local image registry to use for build and push image targets.
LOCAL_REGISTRY ?= localhost:5001

# IMAGE_NAME is the name of EXC image
# Use envoy-xds-controller-dev in default when developing
# Use envoy-xds-controller when releasing an image.
IMAGE_NAME ?= envoy-xds-controller
UI_IMAGE_NAME ?= envoy-xds-controller-ui

# IMAGE is the image URL for build and push image targets.
IMAGE ?= ${REGISTRY}/${IMAGE_NAME}
UI_IMAGE ?= ${REGISTRY}/${UI_IMAGE_NAME}

# LOCAL_IMAGE is the local image URL for build and push image targets.
LOCAL_IMAGE ?= ${LOCAL_REGISTRY}/${IMAGE_NAME}
LOCAL_UI_IMAGE ?= ${LOCAL_REGISTRY}/${UI_IMAGE_NAME}

# Tag is the tag to use for build and push image targets.
TAG ?= $(REV)
