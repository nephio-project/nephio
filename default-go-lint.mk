#  Copyright 2023,2025 The Nephio Authors.
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

GOLANG_CI_VER ?= v1.63.4
GIT_ROOT_DIR ?= $(dir $(lastword $(MAKEFILE_LIST)))
include $(GIT_ROOT_DIR)/detect-container-runtime.mk

# Install link at https://golangci-lint.run/usage/install/ if not running inside a container
.PHONY: lint
lint: ## Run Go linter against the codebase
ifeq ($(CONTAINER_RUNNABLE), 0)
	$(RUN_CONTAINER_COMMAND) docker.io/nephio/gotests:1922925584309030912 \
	golangci-lint run ./... -v
else
	golangci-lint run ./... -v
endif
