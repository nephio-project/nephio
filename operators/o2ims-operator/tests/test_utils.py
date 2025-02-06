###########################################################################
# Copyright 2022-2025 The Nephio Authors.
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

import responses
import os
import pytest
import random
import string

from controllers.utils import *

# Constants used for testing
NAME = "test_name"
NAMESPACE = "test_ns"
TEST_JSON = {"status": {"conditions": [{"message": "test"}]}, "message": "message"}
PV_PARAM = {
    "name": "name",
    "repo_location": "location",
    "template_name": "template",
    "template_version": "version",
    "cluster_name": "cluster",
    "mutators": "mutators",
    "namespace": "namespace",
    "create": False,
}
PV_REV = {
    "items": [
        {
            "metadata": {"name": "name"},
            "spec": {"lifecycle": "lifecycle", "packageName": NAME},
        }
    ]
}
PR_PARAMS = {
    "status": {
        "provisioningStatus": "provisioningStatus",
        "provisionedResourceSet": "provisionedResourceSet",
    }
}
PACKAGE_VARIANTS_URI = f"{KUBERNETES_BASE_URL}/apis/config.porch.kpt.dev/v1alpha1/namespaces/{NAMESPACE}/packagevariants"
PACKAGE_REVISIONS_URI = f"{KUBERNETES_BASE_URL}/apis/porch.kpt.dev/v1alpha1/namespaces/{NAMESPACE}/packagerevisions"
PROVISIONING_REQUEST_URI = f"{KUBERNETES_BASE_URL}/apis/o2ims.provisioning.oran.org/v1alpha1/provisioningrequests"
CAPI_URI = f"{KUBERNETES_BASE_URL}/apis/cluster.x-k8s.io/v1beta1/namespaces/{NAMESPACE}/clusters/{NAME}"


@pytest.fixture(autouse=True)
def setup_and_teardown():
    # Create a test token in /tmp
    test_utils_token_path = "/tmp/test_utils_token"
    test_utils_token_path += "".join(random.choices(string.ascii_letters + string.digits, k=10))
    os.environ["TOKEN"] = test_utils_token_path
    with open(test_utils_token_path, "w") as fp:
        pass
    # Wait for tests to finish
    yield
    # Cleanup token
    if os.path.exists(test_utils_token_path):
        os.remove(test_utils_token_path)


@responses.activate
@pytest.mark.parametrize(
    "get_code, post_code, status, create, response_2, response_2_value, exception",
    [
        (200, None, True, False, "name", NAME, False),
        (401, None, False, False, "reason", "unauthorized", False),
        (403, None, False, False, "reason", "unauthorized", False),
        (404, 200, True, True, "name", NAME, False),
        (404, 201, True, True, "name", NAME, False),
        (404, 401, False, True, "reason", "unauthorized", False),
        (404, 403, False, True, "reason", "unauthorized", False),
        (404, 404, False, True, "reason", "notFound", False),
        (404, 400, False, True, "reason", TEST_JSON["message"], False),
        (404, 1234, False, True, "reason", TEST_JSON, False),
        (404, None, False, True, "reason", "NotAbleToCommunicateWithTheCluster ", True),
        (404, 200, False, False, "reason", "notFound", False),
        (500, None, False, False, "reason", "k8sApi server is not reachable", False),
        (1234, None, False, False, "reason", TEST_JSON, False),
        (None, None, False, False, "reason", "NotAbleToCommunicateWithTheCluster ", True),
    ],
)
def test_create_package_variant(get_code, post_code, status, create, response_2, response_2_value, exception):
    if not exception:
        responses.get(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            json=TEST_JSON,
            status=get_code,
        )
    else:
        responses.get(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            body=Exception(""),
        )

    pv_params = PV_PARAM.copy()
    if get_code == 404 and create:
        responses.post(
            PACKAGE_VARIANTS_URI,
            json=TEST_JSON,
            status=post_code,
        )
        pv_params.update({"create": True})

    response = create_package_variant(NAME, NAMESPACE, pv_params)
    assert response["status"] == status and response[response_2] == response_2_value


@responses.activate
@pytest.mark.parametrize(
    "http_code, status, response_2, response_2_value, exception",
    [
        (200, True, "name", NAME, False),
        (202, True, "name", NAME, False),
        (204, True, "name", NAME, False),
        (401, False, "reason", "unauthorized", False),
        (403, False, "reason", "unauthorized", False),
        (404, False, "reason", "notFound", False),
        (1234, False, "reason", TEST_JSON, False),
        (None, False, "reason", "NotAbleToCommunicateWithTheCluster ", True),
    ],
)
def test_delete_package_variant(http_code, status, response_2, response_2_value, exception):
    if not exception and http_code == 204:
        responses.delete(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            status=http_code,
        )
    elif not exception:
        responses.delete(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            json=TEST_JSON,
            status=http_code,
        )
    else:
        responses.delete(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            body=Exception(""),
        )
    response = delete_package_variant(NAME, NAMESPACE)
    assert response["status"] == status and response[response_2] == response_2_value


@responses.activate
@pytest.mark.parametrize(
    "http_code, status, response_2, response_2_value, response_3, response_3_value, exception",
    [
        (200, True, "name", NAME, "body", TEST_JSON, False),
        (401, False, "reason", "unauthorized", None, None, False),
        (403, False, "reason", "unauthorized", None, None, False),
        (404, False, "reason", "notFound", None, None, False),
        (1234, False, "reason", TEST_JSON, None, None, False),
        (None, False, "reason", "NotAbleToCommunicateWithTheCluster ", None, None, True),
    ],
)
def test_get_package_variant(
    http_code,
    status,
    response_2,
    response_2_value,
    response_3,
    response_3_value,
    exception,
):
    if not exception:
        responses.get(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            json=TEST_JSON,
            status=http_code,
        )
    else:
        responses.get(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            body=Exception(""),
        )
    response = get_package_variant(NAME, NAMESPACE)
    assert response["status"] == status and response[response_2] == response_2_value
    if response_3:
        assert response[response_3] == response_3_value


@responses.activate
@pytest.mark.parametrize(
    "http_code, status, response_2, response_2_value, exception",
    [
        (
            200,
            True,
            "packages",
            [
                {
                    "name": PV_REV["items"][0]["metadata"]["name"],
                    "lifecycle": PV_REV["items"][0]["spec"]["lifecycle"],
                }
            ],
            False,
        ),
        (401, False, "reason", "unauthorized", False),
        (403, False, "reason", "unauthorized", False),
        (404, False, "reason", "notFound", False),
        (1234, False, "reason", "Error in querying for package revision", False),
        (None, False, "reason", "NotAbleToCommunicateWithTheCluster ", True),
    ],
)
def test_get_package_revisions_for_package_variant(http_code, status, response_2, response_2_value, exception):
    if not exception:
        responses.get(
            PACKAGE_REVISIONS_URI,
            json=PV_REV,
            status=http_code,
        )
    else:
        responses.get(
            PACKAGE_REVISIONS_URI,
            body=Exception(""),
        )
    response = get_package_revisions_for_package_variant(NAME, NAMESPACE)
    assert response["status"] == status and response[response_2] == response_2_value


@responses.activate
@pytest.mark.parametrize(
    "http_code, status, response_2, response_2_value, exception",
    [
        (200, True, "name", NAME, False),
        (202, True, "name", NAME, False),
        (204, True, "name", NAME, False),
        (401, False, "reason", "unauthorized", False),
        (403, False, "reason", "unauthorized", False),
        (404, False, "reason", "notFound", False),
        (1234, False, "reason", TEST_JSON, False),
        (None, False, "reason", "NotAbleToCommunicateWithTheCluster ", True),
    ],
)
def test_delete_package_revision(http_code, status, response_2, response_2_value, exception):
    if not exception and http_code == 204:
        responses.delete(
            f"{PACKAGE_REVISIONS_URI}/{NAME}",
            status=http_code,
        )
    elif not exception:
        responses.delete(
            f"{PACKAGE_REVISIONS_URI}/{NAME}",
            json=TEST_JSON,
            status=http_code,
        )
    else:
        responses.delete(
            f"{PACKAGE_REVISIONS_URI}/{NAME}",
            body=Exception(""),
        )
    response = delete_package_revision(NAME, NAMESPACE)
    assert response["status"] == status and response[response_2] == response_2_value


@responses.activate
@pytest.mark.parametrize(
    "pr_code, status, status_response, pv_code, response_2, response_2_value, response_3, response_3_value, response_3_exception, exception",
    [
        (200, True, True, None, "provisioningStatus", PR_PARAMS["status"]["provisioningStatus"], None, None, None, False),
        (
            200,
            True,
            False,
            None,
            "provisioningStatus",
            {
                "provisioningMessage": "Cluster provisioning request received",
                "provisioningState": "progressing",
            },
            None,
            None,
            None,
            False,
        ),
        (401, False, False, None, "reason", "unauthorized", None, None, None, False),
        (403, False, False, None, "reason", "unauthorized", None, None, None, False),
        (404, False, False, 200, "reason", "notFound", "pv", True, None, False),
        (404, False, False, 401, "reason", "notFound", "pv", False, None, False),
        (404, False, False, 403, "reason", "notFound", "pv", False, None, False),
        (404, False, False, 404, "reason", "notFound", "pv", False, None, False),
        (404, False, False, 1234, "reason", "notFound", "pv", False, None, False),
        (404, False, False, None, "reason", "notFound", "pv", False, True, False),
        (1234, False, False, None, "reason", PR_PARAMS, None, None, None, False),
        (None, False, False, None, "reason", "NotAbleToCommunicateWithTheCluster ", None, None, None, True),
    ],
)
def test_check_o2ims_provisioning_request(
    pr_code,
    status,
    status_response,
    pv_code,
    response_2,
    response_2_value,
    response_3,
    response_3_value,
    response_3_exception,
    exception,
):
    if not exception:
        pr_params = PR_PARAMS.copy()
        if pr_code == 200 and not status_response:
            pr_params.pop("status")

        responses.get(
            PROVISIONING_REQUEST_URI,
            json=pr_params,
            status=pr_code,
        )

    else:
        responses.get(
            PROVISIONING_REQUEST_URI,
            body=Exception(""),
        )

    if pv_code and not response_3_exception:
        responses.get(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            json=TEST_JSON,
            status=pv_code,
        )
    elif pv_code and response_3_exception:
        responses.get(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            body=Exception(""),
        )
    response = check_o2ims_provisioning_request(NAME, NAMESPACE)
    print(response)
    assert response["status"] == status and response[response_2] == response_2_value

    if pv_code:
        assert response[response_3] == response_3_value


@responses.activate
@pytest.mark.parametrize(
    "http_code, status, response_2, response_2_value, exception",
    [
        (200, True, "body", TEST_JSON, False),
        (401, False, "reason", "unauthorized", False),
        (403, False, "reason", "unauthorized", False),
        (404, False, "reason", "notFound", False),
        (1234, False, "reason", TEST_JSON["status"]["conditions"][0]["message"], False),
        (None, False, "reason", "NotAbleToCommunicateWithTheCluster ", True),
    ],
)
def test_get_capi_cluster(http_code, status, response_2, response_2_value, exception):
    if not exception:
        responses.get(
            CAPI_URI,
            json=TEST_JSON,
            status=http_code,
        )
    else:
        responses.get(
            CAPI_URI,
            body=Exception(""),
        )
    response = get_capi_cluster(NAME, NAMESPACE)
    assert response["status"] == status and response[response_2] == response_2_value
