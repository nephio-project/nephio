#  Copyright 2025 The Nephio Authors.
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

IMAGE_TAG ?= latest
REGISTRY ?= docker.io/nephio
IMAGE_NAME ?= gen-configmap-fn
IMG ?= $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# This includes the following targets:
#   test, unit, unit-clean,
#   gosec, lint,
#   fmt, vet
include ../../default-go.mk

# This includes the following targets:
#   docker-build, docker-push
include ../../default-docker.mk

# This includes the 'help' target that prints out all targets with their descriptions organized by categories
include ../../default-help.mk

.PHONY: all
all: fmt test docker-build docker-push
