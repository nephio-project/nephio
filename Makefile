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

# CONTAINER_RUNNABLE checks if tests and lint check can be run inside container.
ifeq ($(shell command -v podman > /dev/null 2>&1; echo $$?), 0)
CONTAINER_RUNTIME=podman
else
CONTAINER_RUNTIME=docker
endif
CONTAINER_RUNNABLE ?= $(shell command -v $(CONTAINER_RUNTIME) > /dev/null 2>&1; echo $$?)

export CONTAINER_RUNTIME CONTAINER_RUNNABLE

# find all subdirectories with a go.mod file in them
GO_MOD_DIRS = $(shell find . -name 'go.mod' -exec sh -c 'echo \"$$(dirname "{}")\" ' \; )
# NOTE: the above line is complicated due to the limited capabilities of busybox's `find`.
# It meant to be equivalent with this:  find . -name 'go.mod' -printf "'%h' " 


.PHONY: unit lint gosec test
# delegate these targets to the Makefiles of individual go modules
unit lint gosec test: 
	for dir in $(GO_MOD_DIRS); do \
		$(MAKE) -C "$$dir" $@ ; \
	done

.PHONY: unit-clean docker-build docker-push
# delegate these targets to the Makefiles of individual go modules, 
# but skip the module if the target doesn't exists, or an error happened
docker-build docker-push unit-clean: 
	for dir in $(GO_MOD_DIRS); do \
		$(MAKE) -C "$$dir" $@  || true ; \
	done
