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

FROM python:3.12.10-alpine3.21@sha256:4cad1c099a56dafcfee656a1bdd88c97a1372db02e14fcffa9b1869515956deb AS builder

# Create a non-root user and group
RUN addgroup -g 65535 o2ims && \
    adduser -u 65535 -G o2ims -D -h /home/o2ims o2ims

# Define environment variables
ENV VIRTUAL_ENV=/home/o2ims/venv \
    PATH=/home/o2ims/venv/bin:$PATH

# Set working directory
WORKDIR /usr/o2ims

# Copy and install dependencies
COPY --chown=o2ims:o2ims requirements.txt .
RUN python -m venv $VIRTUAL_ENV && \
    pip install --no-cache-dir -r requirements.txt

# Copy application source
COPY --chown=o2ims:o2ims controllers/ ./src/

# Switch to non-root user
USER o2ims

# Entrypoint
CMD ["kopf", "run", "/usr/o2ims/src/manager.py", "--all-namespaces"]
