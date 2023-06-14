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


##@ Go tests & formatting

self_dir := $(dir $(lastword $(MAKEFILE_LIST)))
include $(self_dir)/default-go-misc.mk 
include $(self_dir)/default-go-test.mk 
include $(self_dir)/default-go-lint.mk 
include $(self_dir)/default-gosec.mk
