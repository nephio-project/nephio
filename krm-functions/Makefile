#  Copyright 2023 The Nephio Authors.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.
GO_VERSION ?= 1.20.2
VERSION ?= latest
REGISTRY ?= docker.io/nephio
ITFIMG ?= $(REGISTRY)/interface-fn:${VERSION}
DNNIMG ?= $(REGISTRY)/dnn-fn:${VERSION}
NADIMG ?= $(REGISTRY)/nad-fn:${VERSION}
IPAMIMG ?= $(REGISTRY)/ipam-fn:${VERSION}
VLANIMG ?= $(REGISTRY)/vlan-fn:${VERSION}
UPFIMG ?= $(REGISTRY)/upf-deploy-fn:${VERSION}
AMFIMG ?= $(REGISTRY)/amf-deploy-fn:${VERSION}
SMFIMG ?= $(REGISTRY)/smf-deploy-fn:${VERSION}

ITFCE_DIR ?= ./interface-fn
DNN_DIR ?= ./dnn-fn
NAD_DIR ?= ./nad-fn
IPAM_DIR ?= ./ipam-fn
VLAN_DIR ?= ./vlan-fn
UPF_DIR ?= ./nfdeploy-fn/upfdeployfn
SMF_DIR ?= ./nfdeploy-fn/smfdeployfn
AMF_DIR ?= ./nfdeploy-fn/amfdeployfn
NFD_DIR ?= ./nfdeploy-fn

# find all subdirectories with a go.mod file in them
GO_MOD_DIRS = $(shell find . -name 'go.mod' -exec sh -c 'echo \"$$(dirname "{}")\" ' \; )
# NOTE: the above line is complicated due to the limited capabilities of busybox's `find`.
# It meant to be equivalent with this:  find . -name 'go.mod' -printf "'%h' " 


.PHONY: all
all: test fmt vet

.PHONY: build
all: docker-build docker-push

.PHONY: test fmt vet
# delegate these targets to the Makefiles of individual go modules
test fmt vet: 
	for dir in $(GO_MOD_DIRS); do \
		$(MAKE) -C "$$dir" $@ ; \
	done

include ../default-go-test.mk
include ../default-go-lint.mk
include ../default-gosec.mk
include ../default-go-misc.mk

.PHONY: docker-build-itfimg docker-build-dnnimg docker-build-nadimg docker-build-ipamimg docker-build-vlanimg docker-build-upfimg docker-build-amfimg docker-build-smfimg
docker-build-itfimg:
	docker buildx build --load --tag  ${ITFIMG} -f $(ITFCE_DIR)/Dockerfile .
docker-build-dnnimg:
	docker buildx build --load --tag  ${DNNIMG} -f $(DNN_DIR)/Dockerfile .
docker-build-nadimg:
	docker buildx build --load --tag  ${NADIMG} -f $(NAD_DIR)/Dockerfile .
docker-build-ipamimg:
	docker buildx build --load --tag  ${IPAMIMG} -f $(IPAM_DIR)/Dockerfile .
docker-build-vlanimg:
	docker buildx build --load --tag  ${VLANIMG} -f $(VLAN_DIR)/Dockerfile .
docker-build-upfimg:
	docker buildx build --load --tag  ${UPFIMG} -f $(UPF_DIR)/Dockerfile .
docker-build-amfimg:
	docker buildx build --load --tag  ${AMFIMG} -f $(AMF_DIR)/Dockerfile .
docker-build-smfimg:
	docker buildx build --push --tag  ${SMFIMG} -f $(SMF_DIR)/Dockerfile .

docker-build: docker-build-itfimg docker-build-dnnimg docker-build-nadimg docker-build-ipamimg docker-build-vlanimg docker-build-upfimg docker-build-amfimg docker-build-smfimg

.PHONY: docker-push-itfimg docker-push-dnnimg docker-push-nadimg docker-push-ipamimg docker-push-vlanimg docker-push-upfimg docker-push-amfimg docker-push-smfimg
docker-push-itfimg:
	docker buildx build --push --tag  ${ITFIMG} -f $(ITFCE_DIR)/Dockerfile .
docker-push-dnnimg:
	docker buildx build --push --tag  ${DNNIMG} -f $(DNN_DIR)/Dockerfile .
docker-push-nadimg:
	docker buildx build --push --tag  ${NADIMG} -f $(NAD_DIR)/Dockerfile .
docker-push-ipamimg:
	docker buildx build --push --tag  ${IPAMIMG} -f $(IPAM_DIR)/Dockerfile .
docker-push-vlanimg:
	docker buildx build --push --tag  ${VLANIMG} -f $(VLAN_DIR)/Dockerfile .
docker-push-upfimg:
	docker buildx build --push --tag  ${UPFIMG} -f $(UPF_DIR)/Dockerfile .
docker-push-amfimg:
	docker buildx build --push --tag  ${AMFIMG} -f $(AMF_DIR)/Dockerfile .
docker-push-smfimg:
	docker buildx build --push --tag  ${SMFIMG} -f $(SMF_DIR)/Dockerfile .

docker-push: docker-push-itfimg docker-push-dnnimg docker-push-nadimg docker-push-ipamimg docker-push-vlanimg docker-push-upfimg docker-push-amfimg docker-push-smfimg

