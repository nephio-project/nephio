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

from utils import validate_template_parameters


def validate_cluster_creation_request(params: dict = None) -> dict:
    """Validate the template parameters of a provisioning request.

    Kept as the name the reconciler has always imported. The rule itself lives
    in utils with the northbound one, because the two disagreed about what a
    valid request is and about whether an invalid one raises or is reported.

    :param params: Parameters to provide to the template
    :type params: dict
    :return: ``{"status": True}``, or ``{"status": False, "reason": ...}``
    :rtype: dict
    """
    return validate_template_parameters(params)
