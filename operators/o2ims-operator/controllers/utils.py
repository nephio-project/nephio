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

import os
from datetime import datetime
from dateutil.tz import tzutc
import requests

requests.packages.urllib3.disable_warnings()

TIME_FORMAT = "%Y-%m-%dT%H:%M:%SZ"
# Allowed values vanilla/Openshift
KUBERNETES_TYPE = str(os.getenv("KUBERNETES_TYPE", "vanilla")).lower()
# Labels to put inside the owned resources
LABEL = {"owner": "o2ims.provisioning.oran.org.provisioningrequests"}
# Log level of the controller
LOG_LEVEL = str(os.getenv("LOG_LEVEL", "INFO"))
# To verify HTTPs certificates when communicating with cluster
HTTPS_VERIFY = bool(os.getenv("HTTPS_VERIFY", False))
# Token used to communicate with Kube cluster
TOKEN = os.getenv("TOKEN", "/var/run/secrets/kubernetes.io/serviceaccount/token")
TOKEN = os.popen(f"cat {TOKEN}").read()
KUBERNETES_BASE_URL = str(os.getenv("KUBERNETES_BASE_URL", "http://127.0.0.1:8080"))
UPSTREAM_PKG_REPO = os.getenv("UPSTREAM_PKG_REPO", "catalog-infra-capi")

HEADERS_DICT = {
    "Content-type": "application/json",
    "Accept": "application/json",
    "User-Agent": "kopf_o2ims_operator/python",
    "Authorization": "Bearer {}".format(TOKEN),
}
CLUSTER_PROVISIONER = str(os.getenv("CLUSTER_PROVISIONER", "capi"))
CREATION_TIMEOUT = int(os.getenv("CREATION_TIMEOUT", 1800))


def create_package_variant(
    name: str = None,
    namespace: str = None,
    pv_param: dict = None,
    label: dict = LABEL,
    logger=None,
):
    """
    :param name: name of the package variant
    :type name: str
    :param namespace: Namespace name
    :type namespace: str
    :param pv_param: parameters of package variant
    :type pv_param: dict
    :param label: label for pv resource
    :type label: dict
    :param logger: logger
    :type logger: <class 'kopf._core.actions.loggers.ObjectLogger'>
    :return: response
    :rtype: dict
    """
    if logger:
        logger.debug("create_package_variant")
    r = get_package_variant(name, namespace, logger)
    if "reason" in r and r["reason"] == "notFound" and pv_param["create"]:
        pv_body = {
            "apiVersion": "config.porch.kpt.dev/v1alpha1",
            "kind": "PackageVariant",
            "metadata": {"name": f"{pv_param['name']}", "label": f"{label}"},
            "spec": {
                "upstream": {
                    "repo": f"{pv_param['repo_location']}",
                    "package": f"{pv_param['template_name']}",
                    "workspaceName": f"{pv_param['template_version']}",
                },
                "downstream": {
                    # TODO: should the repo be configurable instead of being hardcoded?
                    "repo": "mgmt",
                    "package": f"{pv_param['cluster_name']}",
                },
                "annotations": {"approval.nephio.org/policy": "initial"},
                "pipeline": {"mutators": pv_param["mutators"]},
            },
        }
        if logger:
            logger.debug(
                f"package-variant {name} does not exist in namespace {namespace}, o2ims operator is creating it now"
            )
        r = requests.post(
            f"{KUBERNETES_BASE_URL}/apis/config.porch.kpt.dev/v1alpha1/namespaces/{namespace}/packagevariants",
            headers=HEADERS_DICT,
            json=pv_body,
            verify=HTTPS_VERIFY,
        )
        if logger:
            logger.debug(
                "response of the request to create package variant %s is %s"
                % (r.request.url, r.json())
            )
        if r.status_code in [200, 201]:
            response = {"status": True, "name": name}
        elif r.status_code in [401, 403]:
            response = {"status": False, "reason": "unauthorized"}
        elif r.status_code == 404:
            response = {"status": False, "reason": "notFound"}
        elif r.status_code == 400:
            response = {"status": False, "reason": r.json()["message"]}
        elif r.status_code == 500:
            response = {"status": False, "reason": "k8sApi server is not reachable"}
        else:
            response = {"status": False, "reason": r.json()}
    elif r["status"] == True and "name" in r:
        response = {"status": r["status"], "name": r["name"]}
    else:
        response = {"status": r["status"], "reason": r["reason"]}
    if logger:
        logger.debug(response)
    return response


def get_package_variant(name: str = None, namespace: str = None, logger=None):
    """
    :param name: name of the package variant
    :type name: str
    :param namespace: Namespace name
    :type namespace: str
    :param logger: logger
    :type logger: <class 'kopf._core.actions.loggers.ObjectLogger'>
    :return: response
    :rtype: dict
    """
    if logger:
        logger.debug("get package variant")
    try:
        r = requests.get(
            f"{KUBERNETES_BASE_URL}/apis/config.porch.kpt.dev/v1alpha1/namespaces/{namespace}/packagevariants/{name}",
            headers=HEADERS_DICT,
            verify=HTTPS_VERIFY,
        )
    except Exception as e:
        if logger:
            logger.debug("get_package_variant error: %s" % (e))
        return {"status": False, "reason": f"NotAbleToCommunicateWithTheCluster {e}"}
    if logger:
        logger.debug(
            "response of the request to get package variant %s is %s"
            % (r.request.url, r.json())
        )
    if r.status_code in [200]:
        response = {"status": True, "name": name, "body": r.json()}
    elif r.status_code in [401, 403]:
        response = {"status": False, "reason": "unauthorized"}
    elif r.status_code == 404:
        response = {"status": False, "reason": f"notFound"}
    elif r.status_code == 500:
        response = {"status": False, "reason": "k8sApi server is not reachable"}
    else:
        response = {"status": False, "reason": r.json()}
    if logger:
        logger.debug("Status %s" % (response))
    return response


def check_o2ims_provisioning_request(
    name: str = None, namespace: str = None, logger=None
):
    """
    :param name: cluster name
    :type name: str
    :param namespace: Namespace name
    :type namespace: str
    :param logger: logger
    :type logger: <class 'kopf._core.actions.loggers.ObjectLogger'>
    :return: response
    :rtype: dict
    """
    if logger:
        logger.debug("get_capi_cluster")

    try:
        r = requests.get(
            f"{KUBERNETES_BASE_URL}/apis/o2ims.provisioning.oran.org/v1alpha1/provisioningrequests",
            headers=HEADERS_DICT,
            verify=HTTPS_VERIFY,
        )
    except Exception as e:
        if logger:
            logger.debug("check_o2ims_provisioning_request error: %s" % (e))
        return {"status": False, "reason": f"NotAbleToCommunicateWithTheCluster {e}"}
    if r.status_code in [200] and "status" in r.json().keys():
        response = {
            "status": True,
            "provisioningStatus": r.json()["status"]["provisioningStatus"],
        }
        if "provisionedResourceSet" in r.json()["status"]:
            response.update(
                {"provisionedResourceSet": r.json()["status"]["provisionedResourceSet"]}
            )
    elif r.status_code in [200] and "status" not in r.json().keys():
        response = {
            "status": True,
            "provisioningStatus": {
                "provisioningMessage": "Cluster provisioning request received",
                "provisioningState": "progressing",
            },
        }
    elif r.status_code in [401, 403]:
        response = {"status": False, "reason": "unauthorized"}
    elif r.status_code == 404:
        response = {"status": False, "reason": "notFound"}
        creation_status = get_package_variant(
            name=name, namespace=namespace, logger=logger
        )
        response.update({"pv": creation_status["status"]})
    elif r.status_code == 500:
        response = {"status": False, "reason": "k8sApi server is not reachable"}
    else:
        response = {
            "status": False,
            "reason": r.json(),
        }
    if logger:
        logger.debug(f"check_o2ims_provisioning_request response: {r.json()}")
    return response


def get_capi_cluster(name: str = None, namespace: str = None, logger=None):
    """
    :param name: cluster name
    :type name: str
    :param namespace: Namespace name
    :type namespace: str
    :param logger: logger
    :type logger: <class 'kopf._core.actions.loggers.ObjectLogger'>
    :return: response
    :rtype: dict
    """
    if logger:
        logger.debug("get_capi_cluster")

    try:
        r = requests.get(
            f"{KUBERNETES_BASE_URL}/apis/cluster.x-k8s.io/v1beta1/namespaces/{namespace}/clusters/{name}",
            headers=HEADERS_DICT,
            verify=HTTPS_VERIFY,
        )
    except Exception as e:
        if logger:
            logger.debug("get_capi_cluster error: %s" % (e))
        return {"status": False, "reason": f"NotAbleToCommunicateWithTheCluster {e}"}
    if r.status_code in [200]:
        response = {"status": True, "body": r.json()}
    elif r.status_code in [401, 403]:
        response = {"status": False, "reason": "unauthorized"}
    elif r.status_code == 404:
        response = {"status": False, "reason": "notFound"}
    elif r.status_code == 500:
        response = {"status": False, "reason": "k8sApi server is not reachable"}
    else:
        response = {
            "status": False,
            "reason": r.json()["status"]["conditions"][0]["message"],
        }
    if logger:
        logger.debug(f"get_capi_cluster response: {r.json()}")
    return response
