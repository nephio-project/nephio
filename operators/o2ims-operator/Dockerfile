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

FROM python:3.12.9-alpine3.21 AS builder

# Create a non-root user
RUN addgroup --gid 65535 o2ims && \
    adduser --uid 65535 --disabled-password --home /home/o2ims -G o2ims o2ims

# Set working directory
WORKDIR /usr/o2ims

# Copy requirements first and install dependencies
COPY --chown=o2ims:o2ims /operators/o2ims-operator/requirements.txt requirements.txt
RUN python -m venv /home/o2ims/venv && \
    /home/o2ims/venv/bin/pip install --no-cache-dir -r requirements.txt

# Copy application files
COPY --chown=o2ims:o2ims /operators/o2ims-operator/controllers/ src/

# Switch to non-root user
USER o2ims

# Set environment variables
ENV PATH="/home/o2ims/venv/bin:${PATH}" \
    VIRTUAL_ENV="/home/o2ims/venv"

# Run the application
CMD ["kopf", "run", "/usr/o2ims/src/manager.py", "--all-namespaces"]
