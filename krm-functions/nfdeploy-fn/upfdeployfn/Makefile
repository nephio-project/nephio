IMAGE_TAG ?= latest
REGISTRY ?= docker.io/nephio
IMAGE_NAME ?= upf-deploy-fn
IMG ?= $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# This includes the following targets:
#   docker-build, docker-push
include ../../../default-docker.mk

# This includes the 'help' target that prints out all targets with their descriptions organized by categories
include ../../../default-help.mk
