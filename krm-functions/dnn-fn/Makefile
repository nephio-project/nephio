IMG_VERSION ?= latest
IMG_REGISTRY ?= docker.io/nephio
IMG ?= $(IMG_REGISTRY)/dnn-fn:${IMG_VERSION}

.PHONY: all
all: fmt test docker-build

include ../../default-all.mk
