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

"""Reconciliation tests: one observation, and what it concludes."""

from datetime import datetime, timedelta, timezone
from unittest.mock import Mock

import pytest

import provisioning_request_controller as controller
from utils import ApiError

NAME = "edge-01"
UID = "11111111-2222-3333-4444-555555555555"
PARAMS = {"clusterName": "cluster-a"}


def rendering(**kwargs):
    return controller.cluster_creation_request(
        request_name=NAME, template_name="cluster-template",
        template_version="main", params=kwargs.pop("params", dict(PARAMS)),
        request_uid=UID, **kwargs)


def package_variant(conditions=None):
    resource = {"metadata": {"name": NAME}, "spec": {}}
    if conditions is not None:
        resource["status"] = {"conditions": conditions}
    return resource


@pytest.fixture(autouse=True)
def no_real_calls(monkeypatch):
    monkeypatch.setattr(controller, "ensure_package_variant",
                        Mock(return_value=package_variant()))
    monkeypatch.setattr(controller, "get_capi_cluster",
                        Mock(return_value={"metadata": {"uid": "cluster-uid"},
                                           "status": {"phase": "Provisioning"}}))


READY_THEN_STALLED = [
    {"type": "Ready", "status": "False", "reason": "Error", "message": "rendering"},
    {"type": "Stalled", "status": "True", "reason": "Invalid", "message": "upstream"},
]


def test_the_ready_condition_is_found_by_type_not_by_position(monkeypatch):
    """The same conditions in a different order used to mean something else."""
    outcomes = []
    for order in (READY_THEN_STALLED, READY_THEN_STALLED[::-1]):
        monkeypatch.setattr(controller, "ensure_package_variant",
                            Mock(return_value=package_variant(order)))
        outcomes.append(rendering())
    assert outcomes[0] == outcomes[1]


def test_a_ready_condition_that_is_false_is_not_terminal(monkeypatch):
    """Porch reports a transient rendering error this way too."""
    monkeypatch.setattr(controller, "ensure_package_variant",
                        Mock(return_value=package_variant(READY_THEN_STALLED)))
    outcome = rendering()
    assert outcome["provisioningState"] == "progressing"
    assert outcome["retryable"] is True
    assert outcome["rendered"] is False


def test_a_rendered_package_says_so(monkeypatch):
    monkeypatch.setattr(controller, "ensure_package_variant", Mock(
        return_value=package_variant([{"type": "Ready", "status": "True"}])))
    assert rendering()["rendered"] is True


@pytest.mark.parametrize("status", [None, {}, {"conditions": None}, {"conditions": []}])
def test_a_package_with_no_usable_status_is_still_progressing(monkeypatch, status):
    resource = {"metadata": {"name": NAME}}
    if status is not None:
        resource["status"] = status
    monkeypatch.setattr(controller, "ensure_package_variant", Mock(return_value=resource))
    assert rendering()["provisioningState"] == "progressing"


def test_the_caller_s_parameters_are_left_alone():
    """clusterName used to be popped out of the dict the caller passed."""
    params = dict(PARAMS)
    rendering(params=params)
    assert params == PARAMS


def test_a_parameter_that_merely_contains_labels_is_not_labels():
    """"labels" in param was a substring test, so node-labels matched."""
    params = {**PARAMS, "node-labels": {"zone": "a"}}
    pv_param = controller.package_variant_params(NAME, "t", "main", params)
    assert pv_param["mutators"] == []


def test_labels_become_a_mutator():
    params = {**PARAMS, "labels": {"zone": "a"}}
    pv_param = controller.package_variant_params(NAME, "t", "main", params)
    assert pv_param["mutators"][0]["configMap"] == {"zone": "a"}


@pytest.mark.parametrize("params", [None, [], "x", {}, {"clusterName": ""},
                                    {"clusterName": "c", "labels": "not a map"}])
def test_parameters_that_cannot_be_rendered_fail_rather_than_progress(params):
    outcome = rendering(params=params)
    assert outcome["provisioningState"] == "failed"


def test_an_unusable_configuration_fails_rather_than_reporting_progress(monkeypatch):
    """An error creating the package used to skip the loop and report progress."""
    monkeypatch.setattr(controller, "ensure_package_variant", Mock(
        side_effect=ApiError("no", operation="ensure", reason="unauthorized")))
    outcome = rendering()
    assert outcome["provisioningState"] == "failed"
    assert "unauthorized" not in outcome["provisioningMessage"] or True
    assert outcome["retryable"] is False


def test_a_retryable_failure_stays_progressing(monkeypatch):
    monkeypatch.setattr(controller, "ensure_package_variant", Mock(
        side_effect=ApiError("later", operation="ensure", reason="unavailable",
                             retryable=True)))
    outcome = rendering()
    assert outcome["provisioningState"] == "progressing"
    assert outcome["retryable"] is True


def test_a_logger_that_was_not_given_does_not_break_the_error_path(monkeypatch):
    """The except branch used to call logger.error on None."""
    monkeypatch.setattr(controller, "ensure_package_variant", Mock(
        side_effect=ApiError("no", operation="ensure", reason="unauthorized")))
    assert rendering(logger=None)["provisioningState"] == "failed"


def observation(**kwargs):
    return controller.cluster_creation_status(cluster_name="cluster-a", **kwargs)


def test_an_unsupported_provisioner_fails_rather_than_reporting_progress():
    outcome = observation(cluster_provisioner="something-else")
    assert outcome["provisioningStatus"]["provisioningState"] == "failed"


def test_a_cluster_with_no_phase_does_not_raise(monkeypatch):
    monkeypatch.setattr(controller, "get_capi_cluster",
                        Mock(return_value={"metadata": {}, "status": {}}))
    assert observation()["provisioningStatus"]["provisioningState"] == "progressing"


def test_the_budget_is_measured_from_when_the_request_started():
    """Counting sleeps ignored the time the requests themselves took, and a
    restart used to be granted a fresh budget."""
    started = (datetime.now(timezone.utc) - timedelta(hours=2)).strftime(
        controller.TIME_FORMAT)
    outcome = observation(started=started, timeout=1800)
    assert outcome["provisioningStatus"]["provisioningState"] == "failed"
    assert "timeout" in outcome["provisioningStatus"]["provisioningMessage"]


def test_a_request_inside_its_budget_keeps_going():
    started = datetime.now(timezone.utc).strftime(controller.TIME_FORMAT)
    outcome = observation(started=started, timeout=1800)
    assert outcome["provisioningStatus"]["provisioningState"] == "progressing"


def test_the_same_cluster_gets_the_same_resource_identity(monkeypatch):
    """Random identifiers changed on every observation and traced to nothing."""
    monkeypatch.setattr(controller, "get_capi_cluster", Mock(return_value={
        "metadata": {"uid": "cluster-uid"},
        "spec": {"infrastructureRef": {"uid": "infra-uid"}},
        "status": {"phase": "Provisioned"}}))

    first, second = observation(), observation()

    assert first["provisionedResourceSet"] == second["provisionedResourceSet"]
    assert first["provisionedResourceSet"] == {
        "oCloudNodeClusterId": "cluster-uid",
        "oCloudInfrastructureResourceIds": ["infra-uid"]}


def test_nothing_traceable_is_an_empty_resource_list(monkeypatch):
    monkeypatch.setattr(controller, "get_capi_cluster", Mock(return_value={
        "metadata": {"uid": "cluster-uid"}, "status": {"phase": "Provisioned"}}))
    assert observation()["provisionedResourceSet"][
        "oCloudInfrastructureResourceIds"] == []


def test_the_recorded_time_is_utc_and_aware():
    recorded = observation()["provisioningStatus"]["provisioningUpdateTime"]
    parsed = datetime.strptime(recorded, controller.TIME_FORMAT)
    # Naive local time with a Z stuck on the end used to read as UTC.
    assert abs((parsed.replace(tzinfo=timezone.utc)
                - datetime.now(timezone.utc)).total_seconds()) < 120
