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

"""Northbound API tests, through Flask's own routing and serialization.

A route string, a status code and a nested Response object are all things a
hand-rolled call of the view function cannot see.
"""

from unittest.mock import Mock

import pytest
from kubernetes import client

import northbound_restapi
from northbound_restapi import COLLECTION, app

REQUEST = {
    "provisioningRequestId": "edge-01",
    "name": "Human readable display name",
    "description": "unit",
    "templateName": "cluster-template",
    "templateVersion": "main",
    "templateParameters": {"clusterName": "cluster-a"},
}


def provisioning_request(name, state=None, message="", resources=None):
    resource = {
        "apiVersion": "o2ims.provisioning.oran.org/v1alpha1",
        "kind": "ProvisioningRequest",
        "metadata": {"name": name},
        "spec": {"name": f"display {name}", "templateName": "t"},
    }
    if state is not None:
        resource["status"] = {"provisioningStatus": {
            "provisioningState": state,
            "provisioningMessage": message,
            "provisioningUpdateTime": "2026-09-16T00:00:00Z",
        }}
        if resources is not None:
            resource["status"]["provisionedResourceSet"] = resources
    return resource


@pytest.fixture
def api(monkeypatch):
    """Replace the shared client, so no cluster configuration is read."""
    stub = Mock()
    monkeypatch.setattr(northbound_restapi, "_api", stub)
    return stub


@pytest.fixture
def http():
    return app.test_client()


def api_exception(status, reason="failed"):
    return client.exceptions.ApiException(status=status, reason=reason)


def test_the_routes_have_no_trailing_space():
    """A path ending in a space is not the path the SMO calls."""
    # One rule per decorator, so the collection appears twice: POST and GET.
    paths = sorted({str(rule) for rule in app.url_map.iter_rules()
                    if "provisioningRequests" in str(rule)})
    assert paths == [COLLECTION, f"{COLLECTION}/<request_id>"]
    assert not any(path.endswith(" ") for path in paths)


def test_a_created_request_answers_201_and_its_own_id(api, http):
    api.create_cluster_custom_object.return_value = provisioning_request("edge-01")

    answer = http.post(COLLECTION, json=REQUEST)

    assert answer.status_code == 201
    assert answer.get_json()["provisioningRequestId"] == "edge-01"
    sent = api.create_cluster_custom_object.call_args.kwargs
    assert sent["plural"] == "provisioningrequests"
    # The identity is the SMO's id, not the display name.
    assert sent["body"]["metadata"]["name"] == "edge-01"
    assert sent["body"]["spec"]["name"] == REQUEST["name"]
    assert sent["_request_timeout"] == northbound_restapi.API_TIMEOUT


def test_a_duplicate_request_id_is_a_conflict(api, http):
    api.create_cluster_custom_object.side_effect = api_exception(409, "AlreadyExists")
    assert http.post(COLLECTION, json=REQUEST).status_code == 409


@pytest.mark.parametrize("body, missing", [
    ({}, "templateName"),
    ({**REQUEST, "templateParameters": {}}, "templateParameters"),
    ({**REQUEST, "templateParameters": {"other": 1}}, "clusterName"),
])
def test_an_invalid_request_is_the_caller_s_error(api, http, body, missing):
    """These used to be answered with 500, which says the server broke."""
    answer = http.post(COLLECTION, json=body)
    assert answer.status_code == 400
    assert missing in answer.get_json()["status"]["message"]
    api.create_cluster_custom_object.assert_not_called()


def test_a_body_that_is_not_json_is_the_caller_s_error(api, http):
    answer = http.post(COLLECTION, data="not json", content_type="application/json")
    assert answer.status_code == 400
    api.create_cluster_custom_object.assert_not_called()


def test_an_id_that_cannot_be_an_object_name_is_refused(api, http):
    """Rewriting it would hand back an id the caller cannot ask with again."""
    answer = http.post(COLLECTION, json={**REQUEST, "provisioningRequestId": "Edge 01"})
    assert answer.status_code == 400
    api.create_cluster_custom_object.assert_not_called()


def test_an_empty_collection_is_an_empty_collection(api, http):
    api.list_cluster_custom_object.return_value = {"items": []}
    answer = http.get(COLLECTION)
    assert answer.status_code == 200
    assert answer.get_json() == {"items": []}


def test_every_request_is_returned_not_only_the_first(api, http):
    api.list_cluster_custom_object.return_value = {"items": [
        provisioning_request("edge-01", "fulfilled", "done"),
        provisioning_request("edge-02", "failed", "no"),
        provisioning_request("edge-03"),
    ]}

    items = http.get(COLLECTION).get_json()["items"]

    assert [item["provisioningRequestId"] for item in items] == [
        "edge-01", "edge-02", "edge-03"]
    assert [item["status"]["provisioningPhase"] for item in items] == [
        "FULFILLED", "FAILED", "PENDING"]


def test_the_recorded_time_is_reported_not_the_time_of_the_read(api, http):
    api.list_cluster_custom_object.return_value = {"items": [
        provisioning_request("edge-01", "progressing", "on it")]}
    item = http.get(COLLECTION).get_json()["items"][0]
    assert item["status"]["updateTime"] == "2026-09-16T00:00:00Z"


def test_the_resource_set_is_what_was_provisioned(api, http):
    api.list_cluster_custom_object.return_value = {"items": [
        provisioning_request("edge-01", "fulfilled", "done",
                             resources={"oCloudNodeClusterId": "uid-1",
                                        "oCloudInfrastructureResourceIds": ["uid-2"]})]}
    item = http.get(COLLECTION).get_json()["items"][0]
    assert item["provisionedResourceSet"] == {
        "oCloudNodeClusterId": "uid-1", "oCloudInfrastructureResourceIds": ["uid-2"]}


def test_nothing_provisioned_is_an_empty_set_not_a_placeholder(api, http):
    api.list_cluster_custom_object.return_value = {"items": [
        provisioning_request("edge-01", "progressing", "on it")]}
    item = http.get(COLLECTION).get_json()["items"][0]
    assert item["provisionedResourceSet"] == {}


def test_a_listing_that_is_not_a_listing_is_an_upstream_error(api, http):
    api.list_cluster_custom_object.return_value = {"kind": "Status"}
    assert http.get(COLLECTION).status_code == 502


def test_one_request_can_be_read_by_its_id(api, http):
    api.get_cluster_custom_object.return_value = provisioning_request(
        "edge-01", "progressing", "on it")

    answer = http.get(f"{COLLECTION}/edge-01")

    assert answer.status_code == 200
    assert answer.get_json()["provisioningRequestId"] == "edge-01"
    assert api.get_cluster_custom_object.call_args.kwargs["name"] == "edge-01"


def test_an_unknown_id_is_not_found(api, http):
    api.get_cluster_custom_object.side_effect = api_exception(404, "NotFound")
    assert http.get(f"{COLLECTION}/edge-99").status_code == 404


# What the API server said decides. Answering every one of these the same way
# tells a caller the server broke when the request was the thing at fault.
@pytest.mark.parametrize("from_api, to_caller", [
    (422, 400),
    (400, 400),
    (409, 409),
    (429, 503),
    (403, 502),
    (500, 502),
])
def test_the_api_server_s_answer_decides_the_status(api, http, from_api, to_caller):
    api.create_cluster_custom_object.side_effect = api_exception(from_api, "no")
    assert http.post(COLLECTION, json=REQUEST).status_code == to_caller


@pytest.mark.parametrize("request_id", [
    "a.-b",                  # a label starting with a hyphen
    "a..b",                  # an empty label
    "A-1",                   # upper case
    "a_b",                   # underscore
    "a" * 254,               # past the length a name may be
])
def test_an_id_the_api_server_would_reject_is_refused_here(api, http, request_id):
    """Sending it on would come back 422 and read as the server's fault."""
    answer = http.post(COLLECTION, json={**REQUEST, "provisioningRequestId": request_id})
    assert answer.status_code == 400
    api.create_cluster_custom_object.assert_not_called()


@pytest.mark.parametrize("request_id", ["edge-01", "a", "a.b.c", "a" * 253])
def test_an_id_the_api_server_would_take_is_accepted(api, http, request_id):
    api.create_cluster_custom_object.return_value = provisioning_request(request_id)
    assert http.post(COLLECTION, json={**REQUEST,
                                       "provisioningRequestId": request_id}).status_code == 201
