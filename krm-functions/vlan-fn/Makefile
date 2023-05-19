VERSION ?= latest
REGISTRY ?= docker.io/nephio
IMG ?= $(REGISTRY)/vlan-fn:${VERSION}

.PHONY: all
all: fmt test docker-build

include ../../default-all.mk
