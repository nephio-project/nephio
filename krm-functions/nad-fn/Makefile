VERSION ?= latest
REGISTRY ?= docker.io/nephio
IMG ?= $(REGISTRY)/nad-fn:${VERSION}

.PHONY: all
all: fmt test docker-build

include ../../default-all.mk
