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

.SHELLFLAGS = -ec

GO_VERSION ?= 1.20.2
IMG_REGISTRY ?= docker.io/nephio

# find all subdirectories with a go.mod file in them
GO_MOD_DIRS = $(shell find . -name 'go.mod' -exec sh -c 'echo \"$$(dirname "{}")\" ' \; )
# NOTE: the above line is complicated for Mac and busybox compatibilty reasons.
# It is meant to be equivalent with this:  find . -name 'go.mod' -printf "'%h' " 

# find all subdirectories with a Dockerfile in them
DOCKERFILE_DIRS = $(shell find . -iname 'Dockerfile' -exec sh -c 'echo \"$$(dirname "{}")\" ' \; )

# This includes the 'help' target that prints out all targets with their descriptions organized by categories
include default-help.mk
include default-mockery.mk

##@ Go tests & formatting

.PHONY: unit lint gosec test unit-clean 
unit lint gosec test: ## These targets are delegated to the Makefiles of individual Go modules
	for dir in $(GO_MOD_DIRS); do \
		$(MAKE) -C "$$dir" $@ ; \
	done

# delegate these targets to the Makefiles of individual go modules, 
# but simply skip the module if the target doesn't exists, or if an error happened
unit-clean: ## These targets are delegated to the Makefiles of individual Go modules
	for dir in $(GO_MOD_DIRS); do \
		$(MAKE) -C "$$dir" $@ || true ; \
	done


##@ Container images

.PHONY: docker-build docker-push
docker-build docker-push: ## These targets are delegated to the Makefiles next to Dockerfiles
	for dir in $(DOCKERFILE_DIRS); do \
		$(MAKE) -C "$$dir" $@  ; \
	done


##@ Mockery code

.PHONY: install-mockery
install-mockery: ## install mockery
ifeq ($(CONTAINER_RUNNABLE), 0)
		$(CONTAINER_RUNTIME) pull docker.io/vektra/mockery:v${MOCKERY_VERSION}
else
		wget -qO- https://github.com/vektra/mockery/releases/download/v${MOCKERY_VERSION}/mockery_${MOCKERY_VERSION}_${OS}_${OS_ARCH}.tar.gz | sudo tar -xvzf - -C /usr/local/bin
endif

.PHONY: generate-mocks
generate-mocks:
	for dir in $(GO_MOD_DIRS); do \
		$(MAKE) -C "$$dir" $@ || true ; \
	done
