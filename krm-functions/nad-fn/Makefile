IMAGE_TAG ?= latest
REGISTRY ?= docker.io/nephio
IMAGE_NAME ?= nad-fn
IMG ?= $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

include ../../default-all.mk

.PHONY: all
all: modsync fmt test docker-build docker-push

.PHONY: modsync
modsync: 
	go mod tidy
	go mod vendor

