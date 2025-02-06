###########################################################################
# Copyright 2025 The Nephio Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
##########################################################################

FROM python:3.11-alpine3.17 AS builder
COPY controllers/ /src/
COPY requirements.txt /
RUN pip install --user -r /requirements.txt --no-cache-dir
############### Target
FROM python:3.11-alpine3.17 AS target
COPY --from=builder /root/.local \
                   /src/ \
                  /root/.local
ENV PATH=/root/.local/bin:$PATH
CMD ["kopf", "run", "/root/.local/manager.py", "--all-namespaces"]
