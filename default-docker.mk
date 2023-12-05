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
GIT_ROOT_DIR ?= $(dir $(lastword $(MAKEFILE_LIST)))
include $(GIT_ROOT_DIR)/detect-container-runtime.mk

##@ Container images

.PHONY: docker-build
docker-build:  ## Build a container image from the local Dockerfile
	$(CONTAINER_RUNTIME) buildx build --load --tag  ${IMG} -f ./Dockerfile "$(GIT_ROOT_DIR)"

.PHONY: docker-push
docker-push: docker-build ## Build and push the container image
	$(CONTAINER_RUNTIME) push ${IMG}
