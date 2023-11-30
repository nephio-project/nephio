# Copyright 2022 The kpt Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

IMAGE_TAG  ?= latest
IMAGE_REPO ?= docker.io/nephio
IMAGE_NAME ?= porch-server

.PHONY: build-image docker-build
build-image docker-build:
	docker buildx build --load --tag $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) -f $(CURDIR)/Dockerfile ../..

.PHONY: push-image docker-push
push-image docker-push:
	docker buildx build --push --tag $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) -f $(CURDIR)/Dockerfile ../..
