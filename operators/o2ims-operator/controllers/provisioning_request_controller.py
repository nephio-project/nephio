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

from utils import *
import time
import uuid

def check_creation_request_status(
    request_name: str = None,
    namespace: str = "default",
    logger=None,
):
    """
    :param request_name: Name of the provisioning request
    :type request_name: str
    :param namespace: Namespace in which PV will be created
    :type namespace: str
    :param logger: logger
    :type logger: <class 'kopf._core.actions.loggers.ObjectLogger'>
    :return: output
    :rtype: dict
    """
    output = check_o2ims_provisioning_request(
        name=request_name,
        namespace=namespace,
        logger=logger,
    )

    return output

# Creating a package variant 
def cluster_creation_request(
    request_name: str = None,
    template_name: str = None,
    template_version: str = None,
    params: dict = None,
    namespace: str = "default",
    logger=None,
):
    """
    :param request_name: Name of the provisioning request
    :type request_name: str
    :param template_name: Git repository name which contains the template
    :type template_name: str
    :param template_version: Branch of the repository to use for the template
    :type template_version: str
    :param params: Parameters to provide to the template
    :type params: dict
    :param namespace: Namespace in which PV will be created
    :type namespace: str
    :param logger: logger
    :type logger: <class 'kopf._core.actions.loggers.ObjectLogger'>
    :return: output
    :rtype: dict
    """

    # Git repository location
    repo_location = GIT_REPOSITORY
    # Add validation for clusterName
    cluster_name = params["clusterName"]
    params.pop("clusterName")
    # Generate mutators from template parameters (params)
    mutators = []
    for param in params:
        if "labels" in param:
            mutators.append(
                {
                    "image": "gcr.io/kpt-fn/set-labels:v0.2.0",
                    "configMap": params["labels"],
                }
            )

    # Generate package variant body
    package_variant_body = {
        "name": request_name,
        "repo_location": repo_location,
        "template_name": template_name,
        "template_version": template_version,
        "cluster_name": cluster_name,
        "mutators": mutators,
        "namespace": namespace,
        "create": True,
    }
    reason = "Started"
    _status = "False"
    provisioning_message = "Cluster instance rendering ongoing"
    provisioning_state = "progressing"
    timer = 0
    # Short timeouts are fine to see if package variant has problems
    timeout = 30
    try:
        status = create_package_variant(
            name=package_variant_body["name"],
            namespace=namespace,
            pv_param=package_variant_body,
            logger=logger,
        )
        while status["status"]:
            creation_status = get_package_variant(
                name=request_name, namespace=namespace, logger=logger
            )
            if creation_status["status"] and "status" in creation_status["body"].keys():
                if (
                    creation_status["body"]["status"]["conditions"] is not None
                    and len(creation_status["body"]["status"]["conditions"]) > 0
                ):
                    # Checking the status of the latest entry of the list
                    _status = creation_status["body"]["status"]["conditions"][-1][
                        "status"
                    ]
                    reason = creation_status["body"]["status"]["conditions"][-1][
                        "reason"
                    ]
                    if _status == "True":
                        provisioning_message = "Cluster instance rendering completed"
                        provisioning_state = "progressing"
                        break
                    elif _status == "False":
                        provisioning_message = (
                            f"Cluster instance rendering failed {reason}"
                        )
                        provisioning_state = "failed"
                        break
            elif not creation_status["status"]:
                _status = "False"
                provisioning_message = "Cluster instance rendering failed"
                provisioning_state = "failed"
                break
            if timer >= timeout:
                provisioning_message = (
                    "Cluster resource creation failed reached timeout"
                )
                provisioning_state = "failed"
                break
            time.sleep(1)
            timer += 1
    except Exception as e:
        logger.error(
            f"Exception {e} in creating package variant {package_variant_body['name']} in namespace {namespace}"
        )
        provisioning_message = "Cluster instance rendering failed"
        provisioning_state = "failed"

    output = {
        "provisioningMessage": provisioning_message,
        "provisioningState": provisioning_state,
    }
    return output

# Checking the status of cluster creation
# TODO check the status of package revision
def cluster_creation_status(
    cluster_name: str,
    namespace: str = "default",
    timeout=1800,
    cluster_provisioner="capi",
    logger=None,
):
    """
    :param cluster_name: Name of the provisioning request
    :type cluster_name: str
    :param namespace: Namespace in which PV will be created
    :type namespace: str
    :param timeout: Timeout after which cluster creation will be declared failed
    :type timeout: int
    :param cluster_provisioner: name of the cluster provisioner
    :type cluster_provisioner: int
    :param logger: logger
    :type logger: <class 'kopf._core.actions.loggers.ObjectLogger'>
    :return: output
    :rtype: dict
    """

    provisioning_message = "Cluster resource creation ongoing"
    provisioning_state = "progressing"
    # Timer to check for timeout
    timer = 0

    if cluster_provisioner == "capi" and timer <= timeout:
        while True:
            cluster_status = get_capi_cluster(
                name=cluster_name, namespace=namespace, logger=logger
            )
            logger.debug(cluster_status)
            if cluster_status["status"]:
                if "status" in cluster_status["body"].keys():
                    if cluster_status["body"]["status"]["phase"] == "Provisioned":
                        provisioning_message = "Cluster resource created"
                        provisioning_state = "fulfilled"
                        break
            if timer >= timeout:
                provisioning_message = (
                    "Cluster resource creation failed reached timeout"
                )
                provisioning_state = "failed"
                break
            timer += 1
            time.sleep(1)

    output = {
        "provisioningStatus": {
            "provisioningUpdateTime": datetime.now().strftime(TIME_FORMAT),
            "provisioningMessage": provisioning_message,
            "provisioningState": provisioning_state,
        }
    }

    if provisioning_state == "fulfilled":
        output.update(
            {
                "provisionedResources": {
                    "oCloudNodeClusterId": str(uuid.uuid4()),
                    "oCloudInfrastructureResourceIds": [str(uuid.uuid4())],
                }
            }
        )
    return output
