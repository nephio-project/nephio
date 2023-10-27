IMAGE_TAG ?= latest
REGISTRY ?= docker.io/nephio
IMAGE_NAME ?= nfdeploy-fn
IMG ?= $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# This includes the following targets:
#   test, unit, unit-clean,
#   gosec, lint,
#   fmt, vet
include ../../default-go.mk

# This includes the following targets:
#   docker-build, docker-push
include ../../default-docker.mk

# This includes the 'help' target that prints out all targets with their descriptions organized by categories
include ../../default-help.mk

.PHONY: all
all: fmt test docker-build docker-push