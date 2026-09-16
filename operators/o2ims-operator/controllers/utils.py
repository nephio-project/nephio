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
    return all(HOST_LABEL.match(label)
               for label in host.rstrip(".").split("."))


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
# Provisioners this operator knows how to observe. An unknown one used to be
# accepted and then silently skipped, leaving the request reported as
# progressing while nothing was ever done about it.
SUPPORTED_CLUSTER_PROVISIONERS = ("capi",)


def cluster_provisioner() -> str:
    """Return the provisioner, refusing one that cannot be observed."""
    provisioner = str(os.getenv("CLUSTER_PROVISIONER", "capi"))
    if provisioner not in SUPPORTED_CLUSTER_PROVISIONERS:
        raise RuntimeError(
            f"CLUSTER_PROVISIONER {provisioner!r} is not one of "
            f"{', '.join(SUPPORTED_CLUSTER_PROVISIONERS)}"
        )
    return provisioner


def creation_timeout() -> int:
    """Return the provisioning budget in seconds, which has to be positive."""
    raw = os.getenv("CREATION_TIMEOUT", "1800")
    try:
        timeout = int(raw)
    except ValueError as error:
        raise RuntimeError(
            f"CREATION_TIMEOUT {raw!r} is not a number") from error
    if timeout <= 0:
        raise RuntimeError(
            f"CREATION_TIMEOUT {raw!r} is not a positive number")
    return timeout


CLUSTER_PROVISIONER = cluster_provisioner()
CREATION_TIMEOUT = creation_timeout()

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


# Connect and read timeouts, applied per socket operation. Neither bounds the
# request as a whole, so a reconcile keeps its own budget as well.
CONNECT_TIMEOUT = 3.05
READ_TIMEOUT = 10.0
API_TIMEOUT = (CONNECT_TIMEOUT, READ_TIMEOUT)

# Proxies and netrc in the environment are ignored: the API server is reached
# directly with the trust settings resolved above, and a proxy would otherwise
# be handed the service account token. Set O2IMS_TRUST_ENVIRONMENT=true to
# restore the Requests default for a deployment that needs an egress proxy.
TRUST_ENVIRONMENT = env_flag("O2IMS_TRUST_ENVIRONMENT")


class ApiError(Exception):
    """An API call that did not produce a resource.

    Carries what the caller has to decide on: whether another attempt can help,
    and whether a write may have landed even though the answer never arrived.
    """

    def __init__(
        self,
        message: str,
        *,
        operation: str,
        status_code: int = None,
        reason: str = None,
        retryable: bool = False,
        write_outcome_unknown: bool = False,
    ):
        super().__init__(message)
        self.operation = operation
        self.status_code = status_code
        self.reason = reason
        self.retryable = retryable
        self.write_outcome_unknown = write_outcome_unknown


def decode_body(response):
    """Decode a response body once, returning None when it is not an object."""
    try:
        body = response.json()
    except ValueError:
        return None
    return body if isinstance(body, dict) else None


def status_message(body, default: str) -> str:
    """Read the message out of a Kubernetes Status, or fall back."""
    if isinstance(body, dict) and body.get("kind") == "Status":
        return str(body.get("message") or body.get("reason") or default)[:200]
    return default


def read_api_response(response, *, operation: str, writing: bool = False,
                      logger=None) -> dict:
    """Turn one API answer into a resource, or into a classified failure.

    The body is decoded at most once and only after the status is known, so
    enabling debug logging cannot change what this returns.
    """
    body = decode_body(response)
    code = response.status_code
    if logger:
        logger.debug("%s answered %s", operation, code)

    if 300 <= code < 400:
        raise ApiError(
            f"{operation}: the API server redirected to "
            f"{response.headers.get('Location')!r}",
            operation=operation, status_code=code, reason="protocol",
        )
    if code in (200, 201):
        if body is None:
            raise ApiError(
                f"{operation}: the API server answered {code} without a "
                "JSON object",
                operation=operation, status_code=code, reason="protocol",
            )
        return body
    if code in (401, 403):
        raise ApiError(
            f"{operation}: {status_message(body, 'not authorised')}",
            operation=operation, status_code=code, reason="unauthorized",
        )
    if code == 404:
        raise ApiError(
            f"{operation}: {status_message(body, 'not found')}",
            operation=operation, status_code=code, reason="notFound",
        )
    if code == 409:
        raise ApiError(
            f"{operation}: {status_message(body, 'already exists')}",
            operation=operation, status_code=code, reason="conflict",
        )
    if code in (400, 422):
        raise ApiError(
            f"{operation}: "
            f"{status_message(body, 'rejected by the API server')}",
            operation=operation, status_code=code, reason="invalid",
        )
    if code == 429 or code >= 500:
        raise ApiError(
            f"{operation}: "
            f"{status_message(body, 'the API server is unavailable')}",
            operation=operation, status_code=code, reason="unavailable",
            retryable=True, write_outcome_unknown=writing,
        )
    raise ApiError(
        f"{operation}: unexpected status {code}",
        operation=operation, status_code=code, reason="protocol",
    )


def api_call(method: str, url: str, *, operation: str, body: dict = None,
             logger=None) -> dict:
    """Send one request to the API server and return the resource it answered.

    Every call carries a freshly read token, the verified trust settings and a
    bounded timeout, and refuses redirects: the API server has no reason to
    send one, and following it would hand the token to another host.
    """
    writing = method not in ("GET", "HEAD")
    try:
        headers = request_headers()
    except (OSError, RuntimeError) as error:
        # A token that cannot be read is a configuration failure, and saying so
        # is the difference between fixing it and waiting out a timeout. No
        # request is sent.
        raise ApiError(
            f"{operation}: the service account token is not usable ({error})",
            operation=operation, reason="config",
        ) from error

    # The answer is read inside the session: closing it first happens to work
    # while the body is preloaded, and stops working the moment it is not.
    with requests.Session() as session:
        session.trust_env = TRUST_ENVIRONMENT
        try:
            response = session.request(
                method, url,
                headers=headers,
                json=body,
                verify=TLS_VERIFY,
                timeout=API_TIMEOUT,
                allow_redirects=False,
            )
        except requests.exceptions.RequestException as error:
            raise ApiError(
                f"{operation}: cannot reach the API server "
                f"({type(error).__name__})",
                operation=operation, reason="transport", retryable=True,
                write_outcome_unknown=writing,
            ) from error
        return read_api_response(response, operation=operation,
                                 writing=writing, logger=logger)


# Annotations that say which ProvisioningRequest a PackageVariant belongs to.
# A PackageVariant is addressed by a name the request chooses, so the name on
# its own does not establish that this request created it.
OWNER_UID_ANNOTATION = "o2ims.provisioning.oran.org/request-uid"
OWNER_TARGET_ANNOTATION = "o2ims.provisioning.oran.org/downstream-package"


def package_variant_url(namespace: str, name: str = None) -> str:
    """Return the collection URL, or the URL of one named PackageVariant."""
    url = (
        f"{KUBERNETES_BASE_URL}/apis/config.porch.kpt.dev/v1alpha1"
        f"/namespaces/{namespace}/packagevariants"
    )
    return f"{url}/{name}" if name else url


def package_variant_body(pv_param: dict, request_uid: str,
                         label: dict) -> dict:
    """Return the PackageVariant this request asks for."""
    return {
        "apiVersion": "config.porch.kpt.dev/v1alpha1",
        "kind": "PackageVariant",
        "metadata": {
            "name": pv_param["name"],
            "labels": dict(label),
            "annotations": {
                OWNER_UID_ANNOTATION: str(request_uid),
                OWNER_TARGET_ANNOTATION: str(pv_param["cluster_name"]),
            },
        },
        "spec": {
            "upstream": {
                "repo": pv_param["repo_location"],
                "package": pv_param["template_name"],
                "workspaceName": pv_param["template_version"],
            },
            "downstream": {
                # TODO: should the repo be configurable instead of
                # being hardcoded?
                "repo": "mgmt",
                "package": pv_param["cluster_name"],
            },
            "annotations": {"approval.nephio.org/policy": "initial"},
            "pipeline": {"mutators": pv_param["mutators"]},
        },
    }


def owns_package_variant(resource: dict, request_uid: str,
                         pv_param: dict) -> bool:
    """Report whether this PackageVariant was created for this request."""
    annotations = (resource.get("metadata") or {}).get("annotations") or {}
    if annotations.get(OWNER_UID_ANNOTATION) != str(request_uid):
        return False
    downstream = (resource.get("spec") or {}).get("downstream") or {}
    return downstream.get("package") == pv_param["cluster_name"]


def get_package_variant(name: str = None, namespace: str = None,
                        logger=None) -> dict:
    """Return one PackageVariant.

    :raises ApiError: the resource was not returned; ``reason`` says why
    """
    if logger:
        logger.debug("get_package_variant %s", name)
    return api_call(
        "GET", package_variant_url(namespace, name),
        operation=f"get packagevariant {name}", logger=logger,
    )


def ensure_package_variant(
    name: str = None,
    namespace: str = None,
    pv_param: dict = None,
    request_uid: str = None,
    label: dict = LABEL,
    logger=None,
) -> dict:
    """Return the PackageVariant belonging to this request, creating it once.

    An existing PackageVariant of the same name is only accepted when it
    carries this request's ownership annotation and targets the same cluster;
    otherwise it belongs to something else and this request has not been
    fulfilled. A create that answers 409, or whose answer is lost, is settled
    by reading the resource back rather than by assuming either outcome.

    :raises ApiError: the PackageVariant could not be established
    """
    operation = f"ensure packagevariant {name}"
    body = package_variant_body(pv_param, request_uid, label)

    def accept(resource: dict) -> dict:
        if owns_package_variant(resource, request_uid, pv_param):
            return resource
        raise ApiError(
            f"{operation}: a PackageVariant named {name!r} already exists and "
            "was not created for this request",
            operation=operation, reason="conflict",
        )

    try:
        return accept(get_package_variant(name, namespace, logger))
    except ApiError as error:
        if error.reason != "notFound":
            raise
    if not pv_param.get("create"):
        raise ApiError(
            f"{operation}: no PackageVariant named {name!r} and creating "
            "one was not asked for",
            operation=operation, reason="notFound",
        )

    try:
        return accept(api_call(
            "POST", package_variant_url(namespace),
            operation=operation, body=body, logger=logger,
        ))
    except ApiError as error:
        # Either another reconcile won the race, or the write landed and the
        # answer did not. Reading it back is what tells the two apart.
        if error.reason != "conflict" and not error.write_outcome_unknown:
            raise
        try:
            return accept(get_package_variant(name, namespace, logger))
        except ApiError as reread:
            if reread.reason != "notFound":
                raise
            # The read-back settles it: nothing was written, so this is worth
            # another attempt. Reporting the read-back's own "not found" would
            # lose that.
            raise ApiError(
                f"{operation}: the create did not land ({error})",
                operation=operation, reason=error.reason, retryable=True,
            ) from error


def check_o2ims_provisioning_request(
    name: str = None, namespace: str = None, logger=None
) -> dict:
    """Return one ProvisioningRequest, addressed by name.

    ProvisioningRequest is cluster-scoped, so ``namespace`` is accepted for
    call compatibility and not put in the path. A collection answered here
    means the request was not addressed, which is a protocol error rather
    than a request that is still progressing.

    :raises ApiError: the resource was not returned; ``reason`` says why
    """
    operation = f"get provisioningrequest {name}"
    if logger:
        logger.debug("check_o2ims_provisioning_request %s", name)
    resource = api_call(
        "GET",
        f"{KUBERNETES_BASE_URL}/apis/o2ims.provisioning.oran.org/v1alpha1"
        f"/provisioningrequests/{name}",
        operation=operation, logger=logger,
    )
    kind = resource.get("kind")
    if kind != "ProvisioningRequest":
        raise ApiError(
            f"{operation}: the API server answered {kind!r} instead of a "
            "ProvisioningRequest",
            operation=operation, reason="protocol",
        )
    if (resource.get("metadata") or {}).get("name") != name:
        raise ApiError(
            f"{operation}: the API server answered a different object",
            operation=operation, reason="protocol",
        )
    return resource


def provisioning_status(resource: dict) -> dict:
    """Return the recorded provisioning status, or an empty one."""
    status = resource.get("status")
    if not isinstance(status, dict):
        return {}
    recorded = status.get("provisioningStatus")
    return recorded if isinstance(recorded, dict) else {}


def get_capi_cluster(name: str = None, namespace: str = None,
                     logger=None) -> dict:
    """Return one CAPI Cluster.

    :raises ApiError: the resource was not returned; ``reason`` says why
    """
    if logger:
        logger.debug("get_capi_cluster %s", name)
    return api_call(
        "GET",
        f"{KUBERNETES_BASE_URL}/apis/cluster.x-k8s.io/v1beta1"
        f"/namespaces/{namespace}/clusters/{name}",
        operation=f"get cluster {name}", logger=logger,
    )


def validate_cluster_creation_request(params: dict = None) -> dict:
    """Validate the provisioning request envelope and its template parameters.

    One validator for both entry points: the northbound API and the reconciler
    used to disagree about what a valid request is, and about whether an
    invalid one raises or is reported.

    :return: ``{"status": True}``, or ``{"status": False, "reason": ...}``
    """
    if not isinstance(params, dict):
        return {"status": False,
                "reason": "provisioning request must be an object, got "
                          f"{type(params).__name__}"}

    for field in ("templateName", "templateVersion", "templateParameters"):
        if not params.get(field):
            return {"status": False, "reason": f"{field} is empty or missing"}

    template_parameters = params["templateParameters"]
    if not isinstance(template_parameters, dict):
        return {"status": False,
                "reason": "templateParameters must be an object, got "
                          f"{type(template_parameters).__name__}"}

    return validate_template_parameters(template_parameters)


def validate_template_parameters(params: dict = None) -> dict:
    """Validate the template parameters the reconciler renders from.

    :return: ``{"status": True}``, or ``{"status": False, "reason": ...}``
    """
    if not isinstance(params, dict):
        return {"status": False,
                "reason": "templateParameters must be an object, got "
                          f"{type(params).__name__}"}
    cluster_name = params.get("clusterName")
    if not isinstance(cluster_name, str) or not cluster_name.strip():
        return {"status": False,
                "reason": "clusterName is missing in template parameters"}
    labels = params.get("labels")
    if labels is not None and not isinstance(labels, dict):
        return {"status": False,
                "reason": "labels must be an object, got "
                          f"{type(labels).__name__}"}
    return {"status": True}
