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

# find all subdirectories with a Dockerfile in them
DOCKERFILE_DIRS = $(shell find . -iname 'Dockerfile' -printf "'%h' " )

# This includes the following targets:
#   test, unit, unit-clean,
#   gosec, lint,
#   fmt, vet
include ../../default-go.mk

# This includes the 'help' target that prints out all targets with their descriptions organized by categories
include ../../default-help.mk

.PHONY: all
all: fmt test docker-build docker-push

##@ Container images

.PHONY: docker-build docker-push
docker-build: fmt test ## Build container images.
	for dir in $(DOCKERFILE_DIRS); do \
		$(MAKE) -C "$$dir" $@  ; \
	done

docker-push: ## Push container images.
	for dir in $(DOCKERFILE_DIRS); do \
		$(MAKE) -C "$$dir" $@  ; \
	done

.PHONY: upf-docker-build
upf-docker-build:  ## Build docker images.
	$(MAKE) -C upfdeployfn docker-build

.PHONY: upf-docker-push
upf-docker-push: ## Build docker images.
	$(MAKE) -C upfdeployfn docker-push

.PHONY: smf-docker-build
smf-docker-build:  ## Build docker images.
	$(MAKE) -C smfdeployfn docker-build

.PHONY: smf-docker-push
smf-docker-push: ## Build docker images.
	$(MAKE) -C smfdeployfn docker-push

.PHONY: amf-docker-build
amf-docker-build:  ## Build docker images.
	$(MAKE) -C amfdeployfn docker-build

.PHONY: amf-docker-push
amf-docker-push: ## Build docker images.
	$(MAKE) -C amfdeployfn docker-push
