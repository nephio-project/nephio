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

import signal
import socket
import types
from datetime import datetime, timezone
from unittest.mock import Mock

import kopf
import pytest

import manager
from provisioning_request_controller import outcome

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


def just_now():
    """A creation time that leaves the whole budget."""
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def reconcile(patch, memo, spec=None, created=None):
    """Call the handler with what kopf passes, not with literals.

    kopf hands over a Body and a Spec, which are mappings over the resource
    and not dicts. Passing dictionaries here once hid a validation check that
    rejected every real request.
    """
    metadata = {**BODY["metadata"]}
    if created is not None:
        metadata["creationTimestamp"] = created
    body = kopf.Body({**BODY, "metadata": metadata, "spec": spec or SPEC})
    return manager.create_fn(spec=kopf.Spec(body), logger=Mock(), patch=patch,
                             memo=memo, body=body)


def test_the_handler_takes_what_kopf_hands_it():
    """Neither Body nor Spec is a dict, and a check for one turns every real
    request into a permanent validation failure."""
    body = kopf.Body(BODY)
    assert not isinstance(body, dict)
    assert not isinstance(kopf.Spec(body), dict)

    patch = kopf.Patch()
    memo = types.SimpleNamespace(cluster_provisioner="capi", creation_timeout=1800)
    manager.create_fn(spec=kopf.Spec(body), logger=Mock(), patch=patch,
                      memo=memo, body=body)

    # It got past validation and ran to the end. Rejecting a Spec would have
    # left this "failed" with a validation message instead.
    assert patch.status["provisioningStatus"]["provisioningState"] == "fulfilled"


def test_configure_and_reconcile_with_nothing_but_real_kopf_objects(steps):
    """Settings, Memo, Body, Spec and Patch as kopf builds them.

    The reconciler reads its budget and provisioner off the memo configure
    filled in, so the two are worth running together rather than with a stand-in
    in between.
    """
    memo = kopf.Memo()
    manager.configure(settings=kopf.OperatorSettings(), memo=memo)
    assert memo.cluster_provisioner == "capi"
    assert memo.creation_timeout > 0

    body = kopf.Body(BODY)
    patch = kopf.Patch()
    manager.create_fn(spec=kopf.Spec(body), logger=Mock(), patch=patch,
                      memo=memo, body=body)

    assert patch.status["provisioningStatus"]["provisioningState"] == "fulfilled"
    assert steps.observing.call_args.kwargs["cluster_provisioner"] == "capi"


def test_a_fulfilled_request_is_not_driven_again(steps):
    """Resume runs for every existing request when the operator starts. One
    that had been fulfilled used to be re-rendered, which re-creates its
    PackageVariant and reports it as progressing after it was reported done."""
    fulfilled = {**BODY, "status": {"provisioningStatus": {
        "provisioningState": "fulfilled", "provisioningMessage": "done",
        "provisioningUpdateTime": "2026-09-16T00:00:01Z"}}}
    body = kopf.Body(fulfilled)
    patch = kopf.Patch()
    memo = types.SimpleNamespace(cluster_provisioner="capi",
                                 creation_timeout=1800)

    manager.create_fn(spec=kopf.Spec(body), logger=Mock(), patch=patch,
                      memo=memo, body=body)

    steps.rendering.assert_not_called()
    steps.observing.assert_not_called()
    assert "provisioningStatus" not in patch.status


def test_the_api_listens_on_loopback_by_default(monkeypatch):
    """It has no authentication, so it is not put on the pod network unless
    NBI_HOST says so."""
    assert manager.NBI_HOST == "127.0.0.1"

    monkeypatch.setattr(manager, "NBI_PORT", 0)
    logger = Mock()
    manager.start_northbound(logger=logger)
    try:
        assert manager._server.socket.getsockname()[0] == "127.0.0.1"
    finally:
        manager.stop_northbound(logger=logger)


class Server:
    """Stands in for the waitress server: run returns, or raises."""

    def __init__(self, error=None):
        self.error = error

    def run(self):
        if self.error:
            raise self.error


@pytest.mark.parametrize("server", [Server(), Server(OSError("boom"))],
                         ids=["returned", "raised"])
def test_an_api_that_stops_by_itself_stops_the_operator(monkeypatch, server):
    """Exiting is what gets the pod restarted; staying up left the operator
    running with nothing listening, and catalog sets no probe to notice."""
    kill = Mock()
    monkeypatch.setattr(manager.os, "kill", kill)
    manager._stopping.clear()

    manager._serve(server)

    kill.assert_called_once_with(manager.os.getpid(), signal.SIGTERM)


def test_a_real_server_that_stops_by_itself_stops_the_operator(monkeypatch):
    """Closed without stop_northbound, the waitress loop returns."""
    kill = Mock()
    monkeypatch.setattr(manager.os, "kill", kill)
    monkeypatch.setattr(manager, "NBI_PORT", 0)
    logger = Mock()
    manager.start_northbound(logger=logger)
    thread = manager._server_thread
    try:
        manager._server.close()
        thread.join(timeout=5)
        assert not thread.is_alive()
        kill.assert_called_once_with(manager.os.getpid(), signal.SIGTERM)
    finally:
        manager.stop_northbound(logger=logger)


def test_the_probe_connects_to_the_api(monkeypatch):
    monkeypatch.setattr(manager, "NBI_PORT", 0)
    logger = Mock()
    manager.start_northbound(logger=logger)
    try:
        assert manager.northbound_accepts() == "accepting"
    finally:
        manager.stop_northbound(logger=logger)

    with pytest.raises(kopf.PermanentError):
        manager.northbound_accepts()


def test_the_probe_fails_when_nothing_accepts(monkeypatch):
    """A serving thread whose listening socket is gone still looks alive."""
    closed = socket.socket()
    closed.bind(("127.0.0.1", 0))
    address = closed.getsockname()
    closed.close()
    gone = types.SimpleNamespace(socket=types.SimpleNamespace(
        getsockname=lambda: address))
    monkeypatch.setattr(manager, "_server", gone)

    with pytest.raises(kopf.PermanentError, match="does not accept"):
        manager.northbound_accepts()


def test_stopping_the_api_does_not_stop_the_operator(monkeypatch):
    kill = Mock()
    monkeypatch.setattr(manager.os, "kill", kill)
    monkeypatch.setattr(manager, "NBI_PORT", 0)
    logger = Mock()
    manager.start_northbound(logger=logger)
    thread = manager._server_thread
    manager.stop_northbound(logger=logger)

    assert not thread.is_alive()
    kill.assert_not_called()


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
        reconcile(patch, memo, created=just_now())

    assert raised.value.delay == manager.OBSERVE_DELAY
    assert patch.status["provisioningStatus"]["provisioningState"] == "progressing"
    steps.observing.assert_not_called()


# What cluster_creation_request reports for each way rendering can stall.
STALLED = [
    outcome("progressing", "Cluster instance rendering ongoing: "
            "RenderFailed the pipeline failed", retryable=True),
    outcome("progressing", "Cluster instance rendering ongoing",
            retryable=True),
    outcome("progressing", "Cluster instance rendering ongoing: "
            "the API server is unavailable", retryable=True),
]


@pytest.mark.parametrize("rendering", STALLED,
                         ids=["ready-false", "no-ready-condition", "api-error"])
def test_rendering_past_the_budget_fails_the_request(patch, memo, steps, rendering):
    """The budget used to be checked only once rendering had finished, so a
    package that never rendered kept the request progressing for good."""
    steps.rendering.return_value = rendering
    for _ in range(3):
        with pytest.raises(kopf.PermanentError):
            reconcile(patch, memo, created="2020-01-01T00:00:00Z")

        status = patch.status["provisioningStatus"]
        assert status["provisioningState"] == "failed"
        assert status["provisioningMessage"].startswith(
            "Cluster instance rendering timed out")
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
    manager.start_northbound(logger=logger)
    thread = manager._server_thread
    port = manager._server.socket.getsockname()[1]
    try:
        assert thread.is_alive()
        assert port != 0
    finally:
        manager.stop_northbound(logger=logger)

    assert manager._server is None
    # The module variable is cleared either way, so the thread and the port are
    # what say the server actually stopped.
    assert not thread.is_alive()
    released = socket.socket()
    try:
        released.bind(("0.0.0.0", port))
    finally:
        released.close()


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
