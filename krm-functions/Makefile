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

# find all subdirectories with a go.mod file in them
GO_MOD_DIRS = $(shell find . -name 'go.mod' -exec sh -c 'echo \"$$(dirname "{}")\" ' \;)
# find all subdirectories with a Dockerfile in them
DOCKERFILE_DIRS = $(shell find . -iname 'Dockerfile' -exec sh -c 'echo \"$$(dirname "{}")\" ' \;)

# This includes the 'help' target that prints out all targets with their descriptions organized by categories
include ../default-help.mk

.PHONY: docker-build docker-push
docker-build docker-push:  ## Build docker images.
	for dir in $(DOCKERFILE_DIRS); do \
		$(MAKE) -C "$$dir" $@ ; \
	done

.PHONY: test fmt vet unit unit-clean
test fmt vet unit unit-clean: ## These targets are delegated to the Makefiles of individual go modules in this directory subtree
	for dir in $(GO_MOD_DIRS); do \
		$(MAKE) -C "$$dir" $@ ; \
	done

