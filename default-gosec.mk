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

GOSEC_VER ?= 2.22.0
GIT_ROOT_DIR ?= $(dir $(lastword $(MAKEFILE_LIST)))
include $(GIT_ROOT_DIR)/detect-container-runtime.mk

# Install link at https://github.com/securego/gosec#install if not running inside a container
.PHONY: gosec
gosec: ## Inspect the source code for security problems by scanning the Go Abstract Syntax Tree
ifeq ($(CONTAINER_RUNNABLE), 0)
		$(RUN_CONTAINER_COMMAND) docker.io/securego/gosec:${GOSEC_VER} -exclude-dir=test -exclude-generated ./...
else
		gosec -exclude-dir=test -exclude-generated ./...
endif
