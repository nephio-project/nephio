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

"""Operator tests: what reaches the resource, and what serves the API.

The handler runs with real kopf error types and a real kopf.Patch, and the
listener is bound on a real socket, because "the thread was constructed" is
what the previous version was able to prove.
"""

import socket
import types
from datetime import datetime
from unittest.mock import Mock

import kopf
import pytest

import manager

SPEC = {
    "templateName": "cluster-template",
    "templateVersion": "main",
    "templateParameters": {"clusterName": "cluster-a"},
}
BODY = {"metadata": {"name": "edge-01", "uid": "uid-1",
                     "creationTimestamp": "2026-09-16T00:00:00Z"},
        "spec": SPEC}


@pytest.fixture
def patch():
    return kopf.Patch()


@pytest.fixture
def memo():
    return types.SimpleNamespace(cluster_provisioner="capi", creation_timeout=1800)


@pytest.fixture(autouse=True)
def steps(monkeypatch):
    """Both steps succeed unless a test says otherwise."""
    rendering = Mock(return_value={"provisioningState": "progressing",
                                   "provisioningMessage": "rendered",
                                   "retryable": False, "rendered": True})
    observing = Mock(return_value={"provisioningStatus": {
        "provisioningState": "fulfilled", "provisioningMessage": "done",
        "provisioningUpdateTime": "2026-09-16T00:00:01Z"}})
    monkeypatch.setattr(manager, "cluster_creation_request", rendering)
    monkeypatch.setattr(manager, "cluster_creation_status", observing)
    return types.SimpleNamespace(rendering=rendering, observing=observing)


def reconcile(patch, memo, spec=None):
    return manager.create_fn(spec=spec or SPEC, logger=Mock(), patch=patch,
                             memo=memo, body=BODY)


def test_the_probe_answers_with_a_timestamp():
    """datetime.datetime.now on a "from datetime import datetime" import
    raised AttributeError, so the probe never answered."""
    datetime.fromisoformat(manager.get_current_timestamp())


def test_a_validation_failure_is_recorded_before_the_handler_stops(patch, memo):
    """A kopf failure is an annotation. The resource's own status is what an
    operator reads, and it used to be left saying nothing."""
    with pytest.raises(kopf.PermanentError):
        reconcile(patch, memo, spec={**SPEC, "templateParameters": {}})

    assert patch.status["provisioningStatus"]["provisioningState"] == "failed"
    assert "templateParameters" in patch.status["provisioningStatus"][
        "provisioningMessage"]


def test_a_rendering_failure_is_recorded_before_the_handler_stops(patch, memo, steps):
    steps.rendering.return_value = {"provisioningState": "failed",
                                    "provisioningMessage": "the package is wrong",
                                    "retryable": False, "rendered": False}
    with pytest.raises(kopf.PermanentError):
        reconcile(patch, memo)

    assert patch.status["provisioningStatus"]["provisioningState"] == "failed"
    assert patch.status["provisioningStatus"][
        "provisioningMessage"] == "the package is wrong"
    steps.observing.assert_not_called()


def test_rendering_that_has_not_finished_is_retried_not_slept_through(patch, memo, steps):
    steps.rendering.return_value = {"provisioningState": "progressing",
                                    "provisioningMessage": "still rendering",
                                    "retryable": True, "rendered": False}
    with pytest.raises(kopf.TemporaryError) as raised:
        reconcile(patch, memo)

    assert raised.value.delay == manager.OBSERVE_DELAY
    assert patch.status["provisioningStatus"]["provisioningState"] == "progressing"
    steps.observing.assert_not_called()


def test_a_cluster_that_failed_is_what_the_handler_acts_on(patch, memo, steps):
    """The subhandler used to check the previous step's result, so a cluster
    that failed was recorded as failed and then not raised for."""
    steps.observing.return_value = {"provisioningStatus": {
        "provisioningState": "failed", "provisioningMessage": "the cluster failed",
        "provisioningUpdateTime": "2026-09-16T00:00:01Z"}}

    with pytest.raises(kopf.PermanentError, match="the cluster failed"):
        reconcile(patch, memo)

    assert patch.status["provisioningStatus"]["provisioningState"] == "failed"


def test_a_cluster_still_being_built_comes_back_later(patch, memo, steps):
    steps.observing.return_value = {"provisioningStatus": {
        "provisioningState": "progressing", "provisioningMessage": "on it",
        "provisioningUpdateTime": "2026-09-16T00:00:01Z"}}

    with pytest.raises(kopf.TemporaryError):
        reconcile(patch, memo)


def test_a_fulfilled_request_is_recorded_with_its_resources(patch, memo, steps):
    steps.observing.return_value = {
        "provisioningStatus": {"provisioningState": "fulfilled",
                               "provisioningMessage": "done",
                               "provisioningUpdateTime": "2026-09-16T00:00:01Z"},
        "provisionedResourceSet": {"oCloudNodeClusterId": "cluster-uid",
                                   "oCloudInfrastructureResourceIds": []},
    }

    reconcile(patch, memo)

    assert patch.status["provisioningStatus"]["provisioningState"] == "fulfilled"
    assert patch.status["provisionedResourceSet"]["oCloudNodeClusterId"] == "cluster-uid"


def test_the_budget_starts_when_the_request_did(patch, memo, steps):
    """A restart continues the budget rather than being given a fresh one."""
    steps.observing.return_value = {"provisioningStatus": {
        "provisioningState": "progressing", "provisioningMessage": "on it",
        "provisioningUpdateTime": "2026-09-16T00:00:01Z"}}
    with pytest.raises(kopf.TemporaryError):
        reconcile(patch, memo)
    assert steps.observing.call_args.kwargs["started"] == BODY["metadata"][
        "creationTimestamp"]


def test_the_listener_is_bound_and_can_be_stopped(monkeypatch):
    """The previous version evaluated threading.Thread(...).start without
    calling it, so nothing listened and nothing said so."""
    monkeypatch.setattr(manager, "NBI_PORT", 0)
    logger = Mock()
    try:
        manager.start_northbound(logger=logger)
        assert manager._server_thread.is_alive()
        assert manager._server.socket.getsockname()[1] != 0
    finally:
        manager.stop_northbound(logger=logger)
    assert manager._server is None


def test_a_port_that_is_taken_stops_startup(monkeypatch):
    """Failing here is what keeps the operator from running with no API."""
    taken = socket.socket()
    taken.bind(("127.0.0.1", 0))
    taken.listen(1)
    monkeypatch.setattr(manager, "NBI_HOST", "127.0.0.1")
    monkeypatch.setattr(manager, "NBI_PORT", taken.getsockname()[1])
    try:
        with pytest.raises(OSError):
            manager.start_northbound(logger=Mock())
    finally:
        taken.close()
