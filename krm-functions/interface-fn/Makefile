VERSION ?= latest
REGISTRY ?= docker.io/nephio
IMG ?= $(REGISTRY)/interface-fn:${VERSION}

.PHONY: all
all: fmt test docker-build

include ../../default-all.mk
