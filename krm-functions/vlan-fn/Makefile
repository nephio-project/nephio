IMAGE_TAG ?= latest
REGISTRY ?= docker.io/nephio
IMAGE_NAME ?= vlan-fn
IMG ?= $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# This includes the following targets:
#   test, unit, unit-clean,
#   gosec, lint,
#   fmt, vet
include ../../default-go.mk

# This includes the following targets:
#   docker-build, docker-push
include ../../default-docker.mk

.PHONY: all
all: modsync fmt test docker-build docker-push

.PHONY: modsync
modsync: 
	go mod tidy
	go mod vendor

.PHONY: build-image
build-image: docker-build

.PHONY: push-image
push-image:
    docker buildx build --push --tag $(IMG) -f Dockerfile ../..