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

from utils import LOG_LEVEL, CLUSTER_PROVISIONER, CREATION_TIMEOUT
from provisioning_request_controller import *
from provisioning_request_validation_controller import *
from datetime import datetime
import logging
import kopf
import os


@kopf.on.startup()
def configure(settings: kopf.OperatorSettings, memo: kopf.Memo, **_):
    # OwnerReference
    if LOG_LEVEL == "INFO":
        settings.posting.level = logging.INFO
    if LOG_LEVEL == "ERROR":
        settings.posting.level = logging.ERROR
    if LOG_LEVEL == "WARNING":
        settings.posting.level = logging.WARNING
    settings.persistence.finalizer = f"provisioningrequests.o2ims.provisioning.oran.org"
    settings.persistence.progress_storage = kopf.AnnotationsProgressStorage(
        prefix=f"provisioningrequests.o2ims.provisioning.oran.org"
    )
    settings.persistence.diffbase_storage = kopf.AnnotationsDiffBaseStorage(
        prefix=f"provisioningrequests.o2ims.provisioning.oran.org",
        key="last-handled-configuration",
    )
    memo.cluster_provisioner = CLUSTER_PROVISIONER
    memo.creation_timeout = CREATION_TIMEOUT


## kopf.event is designed to show events in kubectl get events. For clusterscope resources currently it is not possible to show events
@kopf.on.resume(f"o2ims.provisioning.oran.org", "provisioningrequests")
@kopf.on.create(f"o2ims.provisioning.oran.org", "provisioningrequests")
async def create_fn(spec, logger, status, patch: kopf.Patch, memo: kopf.Memo, **kwargs):
    metadata_name = kwargs["body"]["metadata"]["name"]
    # Template name will be treated as package name
    template_name = spec.get("templateName")
    # Template version will be treated as repository branch/tag/commit
    template_version = spec.get("templateVersion")
    template_parameters = spec.get("templateParameters")
    kopf.event(
        kwargs["body"],
        type="Info",
        reason="Logging",
        message="Provisioning request validation ongoing",
    )
    # Check in-case the package variant was manually created
    _status = check_creation_request_status(request_name=metadata_name, logger=logger)
    if (
        not _status["status"]
        and _status["reason"] == "notFound"
        and _status["pv"]["status"]
    ):
        patch.status["provisioningStatus"] = {
            "provisioningMessage": "Provisioning request creation failed, package variant already exist",
            "provisioningState": "failed",
            "provisioningUpdateTime": datetime.now().strftime(TIME_FORMAT),
        }
        kopf.event(
            kwargs["body"],
            type="Error",
            reason="Logging",
            message="Provisioning request creation failed, package variant already exist",
        )
        return

    # TODO: This should be done via on.validate handler (admissionwebhooks)
    request_validation = validate_cluster_creation_request(params=template_parameters)

    if not request_validation["status"]:
        patch.status["provisioningStatus"] = {
            "provisioningMessage": "Provisioning request validation failed; reason: "
            + request_validation["reason"],
            "provisioningState": "failed",
            "provisioningUpdateTime": datetime.now().strftime(TIME_FORMAT),
        }
        kopf.PermanentError(
            kwargs["body"],
            type="Error",
            reason="Logging",
            message="Provisioning request validation failed; reason: {request_validation['reason']}",
        )
        return

    @kopf.subhandler()
    def sub_validations(*, patch, **kwargs):
        if request_validation["status"]:
            patch.status["provisioningStatus"] = {
                "provisioningMessage": "Provisioning request validation done",
                "provisioningState": "progressing",
                "provisioningUpdateTime": datetime.now().strftime(TIME_FORMAT),
            }
            kopf.event(
                kwargs["body"],
                type="Info",
                reason="Logging",
                message="Provisioning request validation done",
            )

    creation_request_output = cluster_creation_request(
        request_name=metadata_name,
        template_name=template_name,
        template_version=template_version,
        params=template_parameters.copy(),
        logger=logger,
    )

    if creation_request_output["provisioningState"] == "failed":
        raise kopf.PermanentError("Cluster creation permanently failed")

    @kopf.subhandler()
    def sub_creation(*, patch, **kwargs):
        patch.status["provisioningStatus"] = {
            "provisioningMessage": "Cluster instance rendering completed",
            "provisioningState": "progressing",
            "provisioningUpdateTime": datetime.now().strftime(TIME_FORMAT),
        }

    @kopf.subhandler(timeout=memo.creation_timeout)
    def check_c_status(*, spec, patch, logger, memo: kopf.Memo, **kwargs):
        creation_state_output = cluster_creation_status(
            cluster_name=template_parameters["clusterName"],
            timeout=memo.creation_timeout,
            cluster_provisioner=memo.cluster_provisioner,
            logger=logger,
        )
        patch.status["provisioningStatus"] = creation_state_output["provisioningStatus"]
        if "provisionedResourceSet" in creation_state_output.keys():
            patch.status["provisionedResourceSet"] = creation_state_output[
                "provisionedResourceSet"
            ]
        if creation_request_output["provisioningState"] == "failed":
            raise kopf.PermanentError("Cluster creation permanently failed")


##health check
@kopf.on.probe(id="now")
def get_current_timestamp(**kwargs):
    return datetime.datetime.now(datetime.timezone.utc).isoformat()

