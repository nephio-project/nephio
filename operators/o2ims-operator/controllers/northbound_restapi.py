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
import re
from datetime import datetime, timezone

from flask import Flask, jsonify, request
from kubernetes import client, config

from utils import (
    TIME_FORMAT,
    provisioning_status,
    validate_cluster_creation_request,
)

LOGGER = logging.getLogger(__name__)

app = Flask(__name__)

COLLECTION = "/O2ims_infrastructureProvisioning/v1/provisioningRequests"
GROUP = "o2ims.provisioning.oran.org"
VERSION = "v1alpha1"
PLURAL = "provisioningrequests"

# Bounded so a stalled API server does not hold a northbound request open.
API_TIMEOUT = float(os.getenv("NBI_API_TIMEOUT", "10"))

# The CR state values, and the phases the northbound API answers with. A
# request the reconciler has not recorded anything about yet is PENDING; a
# state nothing maps to is reported as such rather than as progress.
PHASES = {
    "progressing": "PROGRESSING",
    "fulfilled": "FULFILLED",
    "failed": "FAILED",
    "deleting": "DELETING",
}

# metadata.name carries the SMO's provisioning request id, so the id has to be
# one. Rewriting it to fit would hand back an id the caller cannot use again.
LABEL = r"[a-z0-9]([-a-z0-9]*[a-z0-9])?"
KUBERNETES_NAME = re.compile(rf"\A{LABEL}(\.{LABEL})*\Z")
KUBERNETES_NAME_LIMIT = 253

_api = None


def custom_objects_api():
    """Return the shared API client, configured once."""
    global _api
    if _api is None:
        config.load_incluster_config()
        _api = client.CustomObjectsApi()
    return _api


def now() -> str:
    """Return an RFC3339 UTC timestamp."""
    return datetime.now(timezone.utc).strftime(TIME_FORMAT)


def failure(message: str, code: int, phase: str = "FAILED"):
    """Return one error answer in the shape the API describes."""
    return jsonify({"status": {
        "updateTime": now(), "message": message, "provisioningPhase": phase,
    }}), code


def answer_for(error, doing: str):
    """Map what the API server said onto what the caller should be told.

    A rejected object is the caller's to fix and a missing permission is not,
    so they cannot share one status.
    """
    status = getattr(error, "status", None)
    reason = getattr(error, "reason", "") or ""
    if status in (400, 422):
        return failure(f"the provisioning request was rejected: {reason}", 400)
    if status == 409:
        return failure(f"{doing}: it already exists", 409)
    if status == 404:
        return failure(f"{doing}: it was not found", 404)
    if status == 429:
        return failure(f"{doing}: the API server is rate limiting", 503)
    LOGGER.error("%s failed: %s %s", doing, status, reason)
    return failure(f"{doing} failed: {reason}", 502)


def provisioned_resource_set(resource: dict) -> dict:
    """Return the resource set under the names the northbound API uses.

    The CR records these as oCloudNodeClusterId and
    oCloudInfrastructureResourceIds; the API asks for nodeClusterId and
    infrastructureResourceIds, and both are required of it.
    """
    status = resource.get("status")
    recorded = (status.get("provisionedResourceSet")
                if isinstance(status, dict) else None)
    recorded = recorded if isinstance(recorded, dict) else {}
    ids = recorded.get("oCloudInfrastructureResourceIds")
    return {
        "nodeClusterId": recorded.get("oCloudNodeClusterId", ""),
        "infrastructureResourceIds": ids if isinstance(ids, list) else [],
    }


def provisioning_request_info(resource: dict) -> dict:
    """Return one ProvisioningRequest as the northbound API describes it.

    Plain dictionaries throughout: the collection serialises once, at the top,
    and a Response object nested inside another answer is not JSON.
    """
    metadata = resource.get("metadata") or {}
    recorded = provisioning_status(resource)
    state = recorded.get("provisioningState")
    phase = PHASES.get(state)
    if phase is None:
        if state is not None:
            LOGGER.warning(
                "provisioning request %s records a state this API has no "
                "phase for: %r", metadata.get("name"), state)
        # Nothing this operator has produced a phase for yet.
        phase = "PENDING"

    spec = resource.get("spec") or {}
    data = {
        # The id belongs inside the request data, which is where the API
        # declares it; metadata.name is where the CR keeps the SMO's.
        "provisioningRequestId": metadata.get("name"),
        # Required of every answer, and a request written straight to the API
        # server rather than through here may carry neither.
        "name": spec.get("name") or "",
        "description": spec.get("description") or "",
        "templateName": spec.get("templateName") or "",
        "templateVersion": spec.get("templateVersion") or "",
        "templateParameters": spec.get("templateParameters") or {},
    }

    return {
        "provisioningRequestData": data,
        # "assigned by the service producer at the time of request creation":
        # the uid the API server assigned, not something made up here.
        "provisioningRequestReference": metadata.get("uid", ""),
        "status": {
            # The time the reconciler recorded, not the time this was read.
            "updateTime": recorded.get("provisioningUpdateTime", ""),
            "message": recorded.get("provisioningMessage", ""),
            "provisioningPhase": phase,
        },
        # Only what provisioning actually produced. An empty set says nothing
        # has been provisioned yet, which placeholder ids used to hide.
        "provisionedResourceSet": provisioned_resource_set(resource),
    }


@app.route(COLLECTION, methods=["POST"])
def trigger_action():
    """Create a ProvisioningRequest from an SMO provisioning request."""
    data = request.get_json(silent=True)
    if not isinstance(data, dict):
        return failure("the request body is not a JSON object", 400)

    validation = validate_cluster_creation_request(data)
    if not validation["status"]:
        return failure(validation["reason"], 400)

    request_id = data.get("provisioningRequestId")
    if (not isinstance(request_id, str)
            or len(request_id) > KUBERNETES_NAME_LIMIT
            or not KUBERNETES_NAME.match(request_id)):
        return failure(
            "provisioningRequestId must be a lowercase RFC 1123 subdomain "
            "to be used as the provisioning request identity",
            400,
        )

    LOGGER.info("creating provisioning request %s", request_id)
    # The two specifications disagree about these. The CRD types them as
    # strings and rejects a null; the API reference requires them of every
    # answer. An empty string is what both take, and it says the SMO gave
    # none rather than inventing one.
    spec = {
        "name": data.get("name") or "",
        "description": data.get("description") or "",
        "templateName": data["templateName"],
        "templateParameters": data["templateParameters"],
        "templateVersion": data["templateVersion"],
    }

    o2ims_cr = {
        "apiVersion": f"{GROUP}/{VERSION}",
        "kind": "ProvisioningRequest",
        "metadata": {"name": request_id},
        "spec": spec,
    }

    try:
        created = custom_objects_api().create_cluster_custom_object(
            group=GROUP, version=VERSION, plural=PLURAL, body=o2ims_cr,
            _request_timeout=API_TIMEOUT,
        )
    except client.exceptions.ApiException as error:
        return answer_for(error, f"creating provisioning request {request_id}")

    return jsonify(provisioning_request_info(created)), 201


@app.route(COLLECTION, methods=["GET"])
def fetch_status():
    """Return every ProvisioningRequest."""
    try:
        answer = custom_objects_api().list_cluster_custom_object(
            group=GROUP, version=VERSION, plural=PLURAL,
            _request_timeout=API_TIMEOUT,
        )
    except client.exceptions.ApiException as error:
        return answer_for(error, "listing provisioning requests")

    items = answer.get("items") if isinstance(answer, dict) else None
    if not isinstance(items, list):
        return failure("the API server did not answer with a list", 502)

    # Every item, not the first: this used to return from inside the loop.
    return jsonify(
        {"items": [provisioning_request_info(item) for item in items]}), 200


@app.route(f"{COLLECTION}/<request_id>", methods=["GET"])
def fetch_one_status(request_id: str):
    """Return one ProvisioningRequest, by the id the SMO asked with."""
    try:
        resource = custom_objects_api().get_cluster_custom_object(
            group=GROUP, version=VERSION, plural=PLURAL, name=request_id,
            _request_timeout=API_TIMEOUT,
        )
    except client.exceptions.ApiException as error:
        return answer_for(error, f"reading provisioning request {request_id}")

    return jsonify(provisioning_request_info(resource)), 200
