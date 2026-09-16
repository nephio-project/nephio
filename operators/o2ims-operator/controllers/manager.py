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

import logging
import os
import threading
from datetime import datetime, timezone

import kopf
from waitress.server import create_server

from northbound_restapi import app
from utils import (
    CLUSTER_PROVISIONER,
    CREATION_TIMEOUT,
    LOG_LEVEL,
    TIME_FORMAT,
    provisioning_status,
    validate_cluster_creation_request,
)
from provisioning_request_controller import (
    cluster_creation_request,
    cluster_creation_status,
)

LOGGER = logging.getLogger(__name__)

NBI_HOST = os.getenv("NBI_HOST", "0.0.0.0")
NBI_PORT = int(os.getenv("NBI_PORT", "5000"))
# How long a request waits before it is looked at again. One observation per
# pass, so the handler never holds a worker for the whole provisioning budget.
OBSERVE_DELAY = 10
# How long shutdown waits for the northbound API to stop serving. Under the
# five seconds the deployment gives the pod, so the wait finishes rather than
# being cut off by SIGKILL with nothing said about it.
SHUTDOWN_TIMEOUT = float(os.getenv("NBI_SHUTDOWN_TIMEOUT", "3"))

_server = None
_server_thread = None


def now() -> str:
    """Return an RFC3339 UTC timestamp."""
    return datetime.now(timezone.utc).strftime(TIME_FORMAT)


def publish(patch: kopf.Patch, state: str, message: str) -> None:
    """Record where this request has got to.

    One writer, so a later step cannot be overwritten by an earlier one that
    happened to run in another subhandler.
    """
    patch.status["provisioningStatus"] = {
        "provisioningState": state,
        "provisioningMessage": message,
        "provisioningUpdateTime": now(),
    }


@kopf.on.startup()
def configure(settings: kopf.OperatorSettings, memo: kopf.Memo, **_):
    """Configure the operator from settings it can honour."""
    level = {
        "INFO": logging.INFO,
        "ERROR": logging.ERROR,
        "WARNING": logging.WARNING,
        "DEBUG": logging.DEBUG,
    }.get(LOG_LEVEL)
    if level is not None:
        settings.posting.level = level

    settings.persistence.finalizer = (
        "provisioningrequests.o2ims.provisioning.oran.org")
    settings.persistence.progress_storage = kopf.AnnotationsProgressStorage(
        prefix="provisioningrequests.o2ims.provisioning.oran.org"
    )
    settings.persistence.diffbase_storage = kopf.AnnotationsDiffBaseStorage(
        prefix="provisioningrequests.o2ims.provisioning.oran.org",
        key="last-handled-configuration",
    )
    memo.cluster_provisioner = CLUSTER_PROVISIONER
    memo.creation_timeout = CREATION_TIMEOUT


@kopf.on.startup()
def start_northbound(logger, **_):
    """Serve the northbound API, and fail startup if the port cannot be bound.

    create_server binds before it returns, so a port already in use stops the
    operator here instead of leaving it running with no API. This used to be a
    thread expression evaluated at import, which neither started the server nor
    reported that it had not.
    """
    global _server, _server_thread
    _server = create_server(app, host=NBI_HOST, port=NBI_PORT)
    _server_thread = threading.Thread(
        target=_server.run, name="northbound", daemon=True)
    _server_thread.start()
    logger.info("northbound API listening on %s:%s", NBI_HOST, NBI_PORT)


@kopf.on.cleanup()
def stop_northbound(logger, **_):
    """Stop serving, and say so if the server outlives the budget."""
    global _server, _server_thread
    if _server is not None:
        _server.close()
    if _server_thread is not None:
        _server_thread.join(timeout=SHUTDOWN_TIMEOUT)
        if _server_thread.is_alive():
            logger.warning("the northbound API did not stop within %ss",
                           SHUTDOWN_TIMEOUT)
        else:
            logger.info("the northbound API stopped")
    _server, _server_thread = None, None


@kopf.on.resume("o2ims.provisioning.oran.org", "provisioningrequests")
@kopf.on.create("o2ims.provisioning.oran.org", "provisioningrequests")
def create_fn(spec, logger, patch: kopf.Patch, memo: kopf.Memo, body, **_):
    """Move this request one step and record where it got to.

    Synchronous, because the work underneath is synchronous HTTP: an async
    handler running it blocks the event loop Kopf watches every resource with.
    Each pass makes one observation and hands the retry back to Kopf rather
    than sleeping through the provisioning budget inside the handler.
    """
    metadata = body["metadata"]
    request_name = metadata["name"]

    # Fulfilled is where a provisioning request stops. Resume runs for every
    # existing request when the operator starts, and driving a finished one
    # from the rendering step again re-creates its PackageVariant and reports
    # it as progressing after it had been reported done.
    if provisioning_status(body).get("provisioningState") == "fulfilled":
        logger.info("provisioning request %s is already fulfilled",
                    request_name)
        return

    validation = validate_cluster_creation_request(spec)
    if not validation["status"]:
        message = ("Provisioning request validation failed; reason: "
                   f"{validation['reason']}")
        # The business status is written before the handler stops: a Kopf
        # failure is recorded in an annotation, not in the resource's status.
        publish(patch, "failed", message)
        raise kopf.PermanentError(message)

    rendering = cluster_creation_request(
        request_name=request_name,
        template_name=spec["templateName"],
        template_version=spec["templateVersion"],
        params=spec["templateParameters"],
        request_uid=metadata.get("uid"),
        logger=logger,
    )
    if rendering["provisioningState"] == "failed":
        publish(patch, "failed", rendering["provisioningMessage"])
        raise kopf.PermanentError(rendering["provisioningMessage"])
    if not rendering["rendered"]:
        publish(patch, "progressing", rendering["provisioningMessage"])
        raise kopf.TemporaryError(rendering["provisioningMessage"],
                                  delay=OBSERVE_DELAY)

    observation = cluster_creation_status(
        cluster_name=spec["templateParameters"]["clusterName"],
        # The budget runs from when the request was made, so a restart carries
        # on with what is left of it rather than being given a fresh one.
        started=metadata.get("creationTimestamp"),
        timeout=memo.creation_timeout,
        cluster_provisioner=memo.cluster_provisioner,
        logger=logger,
    )
    patch.status["provisioningStatus"] = observation["provisioningStatus"]
    if "provisionedResourceSet" in observation:
        patch.status["provisionedResourceSet"] = observation[
            "provisionedResourceSet"]

    state = observation["provisioningStatus"]["provisioningState"]
    message = observation["provisioningStatus"]["provisioningMessage"]
    if state == "failed":
        raise kopf.PermanentError(message)
    if state != "fulfilled":
        raise kopf.TemporaryError(message, delay=OBSERVE_DELAY)
    logger.info("provisioning request %s fulfilled", request_name)


@kopf.on.probe(id="now")
def get_current_timestamp(**_):
    """Answer the liveness probe."""
    return datetime.now(timezone.utc).isoformat()
