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


def validate_cluster_creation_request(params: dict = None):
    """
    :param params: Parameters to provide to the template
    :type params: dict
    :return: request_validation
    :rtype: dict
    """
    # Checking if clusterName and clusterProvisioner are in parameters
    if "clusterName" in params:
        request_validation = {"status": True}
    else:
        request_validation = {
            "reason": "clusterName is missing in template parameters",
            "status": False,
        }

    return request_validation
