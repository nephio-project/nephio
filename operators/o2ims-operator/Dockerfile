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

FROM python:3.12.9-alpine3.21@sha256:28b8a72c4e0704dd2048b79830e692e94ac2d43d30c914d54def6abf74448a4e AS builder

# Create a non-root user
RUN addgroup --gid 65535 o2ims && \
    adduser --uid 65535 --disabled-password --home /home/o2ims -G o2ims o2ims

# Set working directory
WORKDIR /usr/o2ims

# Copy application files
COPY --chown=o2ims:o2ims controllers/ src/
COPY --chown=o2ims:o2ims requirements.txt requirements.txt

# Switch to non-root user
USER o2ims

# Create a virtual environment and install dependencies
RUN python -m venv /home/o2ims/venv && \
    /home/o2ims/venv/bin/pip install --no-cache-dir -r requirements.txt

# Set environment variables
ENV PATH="/home/o2ims/venv/bin:${PATH}" \
    VIRTUAL_ENV="/home/o2ims/venv"

# Run the application
CMD ["kopf", "run", "/usr/o2ims/src/manager.py", "--all-namespaces"]
