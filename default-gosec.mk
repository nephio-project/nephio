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

GOSEC_VER ?= 2.15.0

# Install link at https://github.com/securego/gosec#install if not running inside a container
.PHONY: gosec
gosec: ## Inspect the source code for security problems by scanning the Go Abstract Syntax Tree
ifeq ($(CONTAINER_RUNNABLE), 0)
		$(CONTAINER_RUNTIME) run -it -v "$(CURDIR):/go/src" -w /go/src docker.io/securego/gosec:${GOSEC_VER} ./...
else
		gosec ./...
endif
