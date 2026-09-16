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
from datetime import datetime, timedelta, timezone

from utils import (
    ApiError,
    CREATION_TIMEOUT,
    TIME_FORMAT,
    UPSTREAM_PKG_REPO,
    ensure_package_variant,
    get_capi_cluster,
    validate_template_parameters,
)

LOGGER = logging.getLogger(__name__)

# How long rendering is given before it is called failed. Rendering either
# reports a condition quickly or something is wrong with the package.
RENDERING_TIMEOUT = 30


def utc_now() -> datetime:
    """Return the current time, aware, so arithmetic and formatting agree."""
    return datetime.now(timezone.utc)


def timestamp(moment: datetime = None) -> str:
    """Return an RFC3339 UTC timestamp."""
    return (moment or utc_now()).strftime(TIME_FORMAT)


def outcome(state: str, message: str, retryable: bool = False,
            rendered: bool = False) -> dict:
    """Return one observation of a provisioning step.

    ``rendered`` says the package has been rendered, which ``progressing`` on
    its own does not: the caller has to know whether to move on to the cluster.
    """
    return {
        "provisioningState": state,
        "provisioningMessage": message,
        "retryable": retryable,
        "rendered": rendered,
    }


def deadline_from(started: str = None, budget: int = None,
                  logger=None) -> datetime:
    """Return the moment this request runs out of time.

    The deadline is derived from when the request started, not from how many
    times it has been looked at, so a restart continues the same budget
    instead of being granted a fresh one. A timestamp that cannot be read
    falls back to now, which does grant a fresh one, so it says so rather
    than quietly undoing that.
    """
    budget = CREATION_TIMEOUT if budget is None else budget
    begin = utc_now()
    if started:
        try:
            # fromisoformat, not strptime: it takes the trailing Z and the
            # fractional seconds a timestamp may carry, which the operator's
            # own format string does not.
            begin = datetime.fromisoformat(started)
            if begin.tzinfo is None:
                begin = begin.replace(tzinfo=timezone.utc)
        except (TypeError, ValueError):
            (logger or LOGGER).warning(
                "cannot read %r as a start time, so this request is being "
                "given a fresh budget", started)
    return begin + timedelta(seconds=budget)


def package_variant_params(
    request_name: str, template_name: str, template_version: str, params: dict
) -> dict:
    """Return the PackageVariant parameters for this request.

    The caller's parameters are copied: this used to pop ``clusterName`` out of
    the dict it was handed.

    :raises ValueError: the template parameters are not usable
    """
    validation = validate_template_parameters(params)
    if not validation["status"]:
        raise ValueError(validation["reason"])

    # dict(): the caller may hand a mapping that is not one, and a copy is
    # what keeps clusterName from being popped out of the caller's own.
    template_parameters = dict(params)
    cluster_name = template_parameters.pop("clusterName")

    mutators = []
    # An exact key: "labels" in param used to be a substring test, so a
    # parameter named node-labels matched and then read params["labels"].
    if isinstance(template_parameters.get("labels"), dict):
        mutators.append({
            "image": "gcr.io/kpt-fn/set-labels:v0.2.0",
            "configMap": template_parameters["labels"],
        })

    return {
        "name": request_name,
        "repo_location": UPSTREAM_PKG_REPO,
        "template_name": template_name,
        "template_version": template_version,
        "cluster_name": cluster_name,
        "mutators": mutators,
        "namespace": None,
        "create": True,
    }


def ready_condition(resource: dict) -> dict:
    """Return the Ready condition, by type rather than by position."""
    status = resource.get("status")
    if not isinstance(status, dict):
        return {}
    conditions = status.get("conditions")
    if not isinstance(conditions, list):
        return {}
    for condition in conditions:
        if isinstance(condition, dict) and condition.get("type") == "Ready":
            return condition
    return {}


def cluster_creation_request(
    request_name: str = None,
    template_name: str = None,
    template_version: str = None,
    params: dict = None,
    request_uid: str = None,
    namespace: str = "default",
    logger=None,
):
    """Ensure the PackageVariant for this request and report what it says now.

    One observation. Whether to look again is the caller's decision, so a
    rendering that has not finished is reported as progressing and retryable
    rather than being waited out inside a handler.

    :return: ``provisioningState``, ``provisioningMessage`` and ``retryable``
    :rtype: dict
    """
    log = logger or LOGGER

    try:
        pv_param = package_variant_params(
            request_name, template_name, template_version, params
        )
    except ValueError as error:
        return outcome("failed", f"Cluster instance rendering failed: {error}")

    try:
        resource = ensure_package_variant(
            name=request_name,
            namespace=namespace,
            pv_param=pv_param,
            request_uid=request_uid,
            logger=logger,
        )
    except ApiError as error:
        log.error("ensuring the package variant for %s failed: %s",
                  request_name, error)
        if error.retryable:
            return outcome("progressing",
                           f"Cluster instance rendering ongoing: {error}",
                           retryable=True)
        return outcome("failed", f"Cluster instance rendering failed: {error}")

    condition = ready_condition(resource)
    state = condition.get("status")
    reason = condition.get("reason", "")
    message = condition.get("message", "")

    if state == "True":
        return outcome("progressing", "Cluster instance rendering completed",
                       rendered=True)
    if state == "False":
        # Porch reports a transient rendering error the same way it reports a
        # package that cannot be rendered at all, so this is not terminal on
        # its own; the caller's budget decides.
        return outcome("progressing",
                       f"Cluster instance rendering ongoing: {reason} "
                       f"{message}".strip(),
                       retryable=True)
    return outcome("progressing", "Cluster instance rendering ongoing",
                   retryable=True)


def cluster_creation_status(
    cluster_name: str,
    namespace: str = "default",
    started: str = None,
    timeout: int = None,
    cluster_provisioner: str = "capi",
    logger=None,
):
    """Observe the cluster once and report where provisioning has got to.

    :param started: when the request began, so a restart keeps its budget
    :return: ``provisioningStatus``, ``retryable`` and, once provisioned,
             ``provisionedResourceSet``
    :rtype: dict
    """
    log = logger or LOGGER

    def report(state: str, message: str, retryable: bool = False) -> dict:
        return {
            "provisioningStatus": {
                "provisioningUpdateTime": timestamp(),
                "provisioningMessage": message,
                "provisioningState": state,
            },
            "retryable": retryable,
        }

    if cluster_provisioner != "capi":
        return report("failed",
                      f"Cluster provisioner {cluster_provisioner!r} is not "
                      "supported")

    deadline = deadline_from(started, timeout, logger=log)

    try:
        cluster = get_capi_cluster(name=cluster_name, namespace=namespace,
                                   logger=logger)
    except ApiError as error:
        log.error("observing cluster %s failed: %s", cluster_name, error)
        if not error.retryable and error.reason not in ("notFound",):
            return report("failed",
                          f"Cluster resource creation failed: {error}")
        if utc_now() >= deadline:
            return report("failed",
                          f"Cluster resource creation timed out: {error}")
        return report("progressing", "Cluster resource creation ongoing",
                      retryable=True)

    phase = (cluster.get("status") or {}).get("phase")
    if phase == "Provisioned":
        result = report("fulfilled", "Cluster resource created")
        result["provisionedResourceSet"] = provisioned_resource_set(cluster)
        return result
    if utc_now() >= deadline:
        return report("failed",
                      "Cluster resource creation failed reached timeout")
    return report("progressing",
                  "Cluster resource creation ongoing"
                  f"{f' ({phase})' if phase else ''}",
                  retryable=True)


def provisioned_resource_set(cluster: dict) -> dict:
    """Return the resources this cluster actually stands for.

    The identifiers are read off the provisioned objects. An O-Cloud inventory
    identifier this operator has not been given is left out rather than
    invented: a random one changes on every observation and traces to nothing.
    """
    metadata = cluster.get("metadata") or {}
    infrastructure = (cluster.get("spec") or {}).get("infrastructureRef") or {}
    resource_ids = [infrastructure["uid"]] if infrastructure.get("uid") else []
    return {
        "oCloudNodeClusterId": metadata.get("uid", ""),
        "oCloudInfrastructureResourceIds": resource_ids,
    }
