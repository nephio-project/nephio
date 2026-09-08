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

import ipaddress
import logging
import re
import os
import ssl
from urllib.parse import urlsplit

import requests

TIME_FORMAT = "%Y-%m-%dT%H:%M:%SZ"
# Allowed values vanilla/Openshift
KUBERNETES_TYPE = str(os.getenv("KUBERNETES_TYPE", "vanilla")).lower()
# Labels to put inside the owned resources
LABEL = {"owner": "o2ims.provisioning.oran.org.provisioningrequests"}
# Log level of the controller
LOG_LEVEL = str(os.getenv("LOG_LEVEL", "DEBUG"))

LOGGER = logging.getLogger(__name__)

# CA bundle and token Kubernetes mounts into every container
IN_CLUSTER_CA_FILE = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
IN_CLUSTER_TOKEN_FILE = "/var/run/secrets/kubernetes.io/serviceaccount/token"
# API server address used outside a cluster
DEFAULT_KUBERNETES_BASE_URL = "https://kubernetes.default.svc"


def env_flag(name: str) -> bool:
    """Return True only for an explicit true value; anything else is False."""
    return os.getenv(name, "").strip().lower() in ("1", "true", "yes", "on")


# Letter-digit-hyphen labels, which is all a hostname may be. Anything a
# second parser could read differently — percent escapes, backslashes,
# spaces, anything non-ASCII — is simply not in here.
HOST_LABEL = re.compile(r"\A[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?\Z")


def is_unambiguous_host(host: str) -> bool:
    """Whether every url parser will read this as the same host.

    Named characters were refused one at a time until fuzzing the
    validator against requests turned up two more classes at once, so
    this allows a shape instead: an IP literal, or a DNS name.

    :param host: the host as urlsplit reports it
    :rtype: bool
    """
    try:
        ipaddress.ip_address(host)
        return True
    except ValueError:
        pass
    if not host or len(host) > 253:
        return False
    return all(HOST_LABEL.match(label) for label in host.rstrip(".").split("."))


def validate_api_server_url(base_url: str) -> str:
    """Return base_url once it is safe to send a bearer token to.

    requests applies ``verify`` only to https, so a plaintext endpoint is
    refused outright rather than merely unverified.

    :return: the url, without a trailing slash
    :rtype: str
    """
    # urlsplit rejects an unclosed IPv6 bracket by raising, and so does reading
    # .port on one that is not a number in range. Neither should reach the
    # caller as a bare ValueError when every other rejection here is a
    # RuntimeError naming what is wrong.
    try:
        parsed = urlsplit(base_url)
    except ValueError:
        raise RuntimeError(
            f"Kubernetes API url {base_url!r} cannot be parsed"
        ) from None
    try:
        port = parsed.port
        problem = ""
        if any(ch < " " or ch == "\x7f" for ch in base_url):
            # urlsplit drops some of these and keeps others, while requests
            # percent-encodes them, so the host checked here is not always the
            # host connected to.
            problem = "must not contain control characters"
        elif parsed.scheme != "https":
            problem = "must use https"
        elif not parsed.hostname:
            problem = "has no host"
        elif not is_unambiguous_host(parsed.hostname):
            # urlsplit keeps a backslash and a percent escape in the host;
            # requests hands the authority to urllib3, which reads the first as
            # a path separator and decodes the second. Either way the address
            # checked here would not be the one connected to.
            problem = f"has an unusable host {parsed.hostname!r}"
        elif parsed.username or parsed.password:
            problem = "must not embed credentials"
        elif parsed.query or parsed.fragment:
            problem = "must not carry a query or fragment"
        elif port == 0:
            problem = "has an invalid port"
    except ValueError:
        problem = "has an invalid port"
    if problem:
        raise RuntimeError(f"Kubernetes API url {base_url!r} {problem}")
    return base_url.rstrip("/")


def validate_ca_bundle(path: str) -> str:
    """Return path once it actually loads as a CA bundle.

    Checking the path alone would accept an empty or malformed PEM and
    defer the failure to the first request.

    :return: the path
    :rtype: str
    """
    try:
        ssl.create_default_context(cafile=path)
    except (OSError, ssl.SSLError) as exc:
        raise RuntimeError(f"CA bundle {path!r} is unusable: {exc}") from exc
    return path


def kubernetes_base_url() -> str:
    """
    :return: KUBERNETES_BASE_URL, else the address advertised to the pod,
             the only one its certificate is expected to be valid for
    :rtype: str
    """
    base_url = os.getenv("KUBERNETES_BASE_URL")
    advertised = in_cluster_base_url()
    if not base_url:
        return advertised or DEFAULT_KUBERNETES_BASE_URL

    base_url = validate_api_server_url(base_url)
    # Catalog releases before nephio-project/catalog#146 pin the in-cluster DNS
    # name, which Kubernetes does not promise a certificate for. That value
    # cannot be dropped in the same commit as this one, because it lives in
    # another repository, so in a pod it resolves to the address Kubernetes
    # does promise: the same server, named the way it is certified.
    #
    # Only for that deployment, which sets the address and nothing else. A CA
    # bundle or a skip flag says the endpoint was chosen deliberately, and the
    # bundle may only certify the name that was asked for: substituting an
    # address there would break a working setup rather than rescue a stale one.
    chose_a_trust_policy = bool(os.getenv("KUBERNETES_CA_FILE")) or env_flag(
        "UNSAFE_SKIP_TLS_VERIFY"
    )
    if (
        advertised
        and not chose_a_trust_policy
        and urlsplit(base_url).path in ("", "/")
        and tls_origin(base_url) == tls_origin(DEFAULT_KUBERNETES_BASE_URL)
    ):
        LOGGER.warning(
            "KUBERNETES_BASE_URL is %s, which Kubernetes does not guarantee a "
            "serving certificate for; using %s instead. Remove the variable "
            "to silence this.",
            base_url,
            advertised,
        )
        return advertised
    return base_url


def in_cluster_base_url() -> str:
    """
    :return: the address the kubelet advertises, or "" outside a pod
    :rtype: str
    """
    host = os.getenv("KUBERNETES_SERVICE_HOST")
    if not host:
        return ""
    if ":" in host and not host.startswith("["):  # IPv6 literal
        host = f"[{host}]"
    port = os.getenv("KUBERNETES_SERVICE_PORT", "443")
    return validate_api_server_url(f"https://{host}:{port}")


def tls_origin(base_url: str) -> tuple:
    """Reduce a URL to the endpoint its certificate is issued for.

    https://host and https://host:443 are one endpoint, as are the short
    and long spellings of an IPv6 address, so comparing URLs as written
    would send equivalent addresses down different trust paths.

    :param base_url: an API server URL
    :return: scheme, canonical host and effective port
    :rtype: tuple
    """
    parsed = urlsplit(base_url)
    host = parsed.hostname or ""
    try:
        host = ipaddress.ip_address(host).compressed
    except ValueError:
        host = host.lower()
    return parsed.scheme.lower(), host, parsed.port or 443


def tls_verify(base_url: str):
    """
    Only UNSAFE_SKIP_TLS_VERIFY disables verification; the bearer token
    below is attached to every request. In a pod the mounted CA has to
    load, because the fallback is not a failed handshake but the public
    roots requests ships with.

    Returning True means the bundle requests defaults to, which is certifi
    and which REQUESTS_CA_BUNDLE or CURL_CA_BUNDLE can redirect. That only
    happens with no CA configured for this endpoint; a path is never
    overridden.

    :param base_url: the endpoint the token will be sent to
    :return: CA bundle path, True for the requests default bundle, or False
    :rtype: str or bool
    """
    unsafe = env_flag("UNSAFE_SKIP_TLS_VERIFY")
    ca_file = os.getenv("KUBERNETES_CA_FILE")
    if unsafe and ca_file:
        raise RuntimeError(
            "KUBERNETES_CA_FILE and UNSAFE_SKIP_TLS_VERIFY conflict, "
            "refusing to guess which one was meant"
        )
    if unsafe:
        return False
    if ca_file:
        return validate_ca_bundle(ca_file)
    # The mounted bundle is issued for the cluster's own API server, reachable
    # both at the advertised address and at the in-cluster name. Pinning it to
    # some other endpoint would reject every connection to it.
    origin = tls_origin(base_url)
    in_cluster = in_cluster_base_url()
    if origin not in (
        tls_origin(in_cluster) if in_cluster else (),
        tls_origin(DEFAULT_KUBERNETES_BASE_URL),
    ):
        return True
    if os.path.exists(IN_CLUSTER_CA_FILE):
        return validate_ca_bundle(IN_CLUSTER_CA_FILE)
    if os.getenv("KUBERNETES_SERVICE_HOST"):
        raise RuntimeError(
            f"the in-cluster CA {IN_CLUSTER_CA_FILE} is missing; supply "
            "KUBERNETES_CA_FILE or set UNSAFE_SKIP_TLS_VERIFY"
        )
    return True


KUBERNETES_BASE_URL = kubernetes_base_url()

# Verify the API server certificate on every request
TLS_VERIFY = tls_verify(KUBERNETES_BASE_URL)
if TLS_VERIFY is False:
    LOGGER.warning(
        "UNSAFE_SKIP_TLS_VERIFY is set: the Kubernetes API server "
        "certificate will NOT be verified and the service account token "
        "can be intercepted. Never enable this outside development."
    )
if os.getenv("HTTPS_VERIFY") is not None:
    LOGGER.warning(
        "HTTPS_VERIFY is no longer supported and is ignored; certificates "
        "are always verified unless UNSAFE_SKIP_TLS_VERIFY=true"
    )
UPSTREAM_PKG_REPO = os.getenv("UPSTREAM_PKG_REPO", "catalog-infra-capi")
CLUSTER_PROVISIONER = str(os.getenv("CLUSTER_PROVISIONER", "capi"))
CREATION_TIMEOUT = int(os.getenv("CREATION_TIMEOUT", 1800))

BASE_HEADERS = {
    "Content-type": "application/json",
    "Accept": "application/json",
    "User-Agent": "kopf_o2ims_operator/python",
}


def read_token() -> str:
    """Return the token, read fresh so that rotation is picked up.

    An empty or blank file is refused rather than turned into a bare
    ``Bearer``, which the API server answers with an opaque 401.

    :return: the token
    :rtype: str
    """
    path = os.getenv("TOKEN", IN_CLUSTER_TOKEN_FILE)
    with open(path, encoding="utf-8") as token_file:
        token = token_file.read().strip()
    if not token:
        raise RuntimeError(f"Kubernetes token file {path!r} is empty")
    if any(character.isspace() for character in token):
        raise RuntimeError(f"Kubernetes token file {path!r} is not a token")
    return token


def request_headers() -> dict:
    """Return the API server request headers, carrying the current token."""
    return {**BASE_HEADERS, "Authorization": f"Bearer {read_token()}"}


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
            headers=request_headers(),
            json=pv_body,
            verify=TLS_VERIFY,
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
    elif r["status"] and "name" in r:
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
            headers=request_headers(),
            verify=TLS_VERIFY,
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
        response = {"status": False, "reason": "notFound"}
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
            headers=request_headers(),
            verify=TLS_VERIFY,
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
            headers=request_headers(),
            verify=TLS_VERIFY,
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

def validate_cluster_creation_request(params: dict = None):
    """
    :param params: parameters of cluster creation request
    :type params: dict
    :return: None
    :rtype: None
    """
  
    # Validate templateName
    if not params.get('templateName'):
        raise ValueError("Parameter 'templateName' is empty or missing.")

    # Validate templateVersion
    if not params.get('templateVersion'):
        raise ValueError("Parameter 'templateVersion' is empty or missing.")

    # Validate templateParameters
    if not params.get('templateParameters'):
        raise ValueError("Parameter 'templateParameters' is empty or missing.")
