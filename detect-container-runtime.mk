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


# detects if a container runtime is present, so that we can run tests, linters, etc. inside containers
ifeq ($(CONTAINER_RUNTIME),)
ifeq ($(shell command -v podman > /dev/null 2>&1; echo $$?), 0)
CONTAINER_RUNTIME=podman
else
CONTAINER_RUNTIME=docker
endif
endif

CONTAINER_RUNNABLE ?= $(shell command -v $(CONTAINER_RUNTIME) > /dev/null 2>&1; echo $$?)

RUN_CONTAINER_COMMAND ?= $(CONTAINER_RUNTIME) run -it --rm -v "$(abspath $(GIT_ROOT_DIR)):$(abspath $(GIT_ROOT_DIR))" -w "$(CURDIR)" --security-opt label=disable 
