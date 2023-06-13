IMAGE_TAG ?= latest
REGISTRY ?= docker.io/nephio
UPFIMG ?= $(REGISTRY)/upf-deploy-fn:${IMAGE_TAG}
AMFIMG ?= $(REGISTRY)/amf-deploy-fn:${IMAGE_TAG}
SMFIMG ?= $(REGISTRY)/smf-deploy-fn:${IMAGE_TAG}

include ../../default-all.mk

.PHONY: all
all: modsync fmt test

.PHONY: modsync
modsync: 
	go mod tidy
	go mod vendor

