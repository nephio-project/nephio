# Copyright 2026 The Nephio Authors.
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

"""Start every run from the same environment, whatever is running it.

controllers.utils resolves the API address and the CA bundle at import
time, so a fixture cannot undo an ambient setting: by the time one runs,
the test module has already imported the package. pytest loads conftest
before it imports test modules, which is early enough.

A pod is where this matters. The kubelet injects KUBERNETES_SERVICE_HOST
and KUBERNETES_SERVICE_PORT even when automountServiceAccountToken is
false, so a CI pod with no service account volume looks like a cluster
with no CA and the import fails during collection.
"""

import os

AMBIENT_KUBERNETES_ENV = (
    "KUBERNETES_SERVICE_HOST",
    "KUBERNETES_SERVICE_PORT",
    "KUBERNETES_BASE_URL",
    "KUBERNETES_CA_FILE",
    "UNSAFE_SKIP_TLS_VERIFY",
    "HTTPS_VERIFY",
    # requests rewrites verify=True to whichever bundle these name.
    "REQUESTS_CA_BUNDLE",
    "CURL_CA_BUNDLE",
)

for _name in AMBIENT_KUBERNETES_ENV:
    os.environ.pop(_name, None)
