IMAGE_TAG ?= latest
REGISTRY ?= docker.io/nephio
IMAGE_NAME ?= dnn-fn
IMG ?= $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

include ../../default-all.mk

.PHONY: all
all: modsync fmt test docker-build docker-push

.PHONY: modsync
modsync: 
	go mod tidy
	go mod vendor

.PHONY: docker-build
build-image:
	docker buildx build --load --tag $(IMG) -f Dockerfile .

.PHONY: docker-push
push-image:
    docker buildx build --push --tag $(IMG) -f Dockerfile .