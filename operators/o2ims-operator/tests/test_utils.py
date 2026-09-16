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

import http.server
import json
import logging
import os
import random
import string
import threading

import certifi
import pytest
import requests
import responses

import utils
from utils import (
    API_TIMEOUT,
    ApiError,
    KUBERNETES_BASE_URL,
    LABEL,
    OWNER_UID_ANNOTATION,
    check_o2ims_provisioning_request,
    ensure_package_variant,
    get_capi_cluster,
    get_package_variant,
    )

# Constants used for testing
NAME = "test_name"
NAMESPACE = "test_ns"
TEST_JSON = {"status": {"conditions": [{"message": "test"}]}, "message": "message"}
PV_PARAM = {
    "name": "name",
    "repo_location": "location",
    "template_name": "template",
    "template_version": "version",
    "cluster_name": "cluster",
    "mutators": "mutators",
    "namespace": "namespace",
    "create": False,
}
PV_REV = {
    "items": [
        {
            "metadata": {"name": "name"},
            "spec": {"lifecycle": "lifecycle", "packageName": NAME},
        }
    ]
}
PR_PARAMS = {
    "status": {
        "provisioningStatus": "provisioningStatus",
        "provisionedResourceSet": "provisionedResourceSet",
    }
}
PACKAGE_VARIANTS_URI = f"{KUBERNETES_BASE_URL}/apis/config.porch.kpt.dev/v1alpha1/namespaces/{NAMESPACE}/packagevariants"
PACKAGE_REVISIONS_URI = f"{KUBERNETES_BASE_URL}/apis/porch.kpt.dev/v1alpha1/namespaces/{NAMESPACE}/packagerevisions"
PROVISIONING_REQUEST_URI = f"{KUBERNETES_BASE_URL}/apis/o2ims.provisioning.oran.org/v1alpha1/provisioningrequests"
CAPI_URI = f"{KUBERNETES_BASE_URL}/apis/cluster.x-k8s.io/v1beta1/namespaces/{NAMESPACE}/clusters/{NAME}"

REQUEST_UID = "11111111-2222-3333-4444-555555555555"
PV_CREATE = {**PV_PARAM, "create": True}
# What the operator sends, and so what it has to accept back as its own.
OWNED_PV = {
    "apiVersion": "config.porch.kpt.dev/v1alpha1",
    "kind": "PackageVariant",
    "metadata": {
        "name": NAME,
        "labels": dict(LABEL),
        "annotations": {
            OWNER_UID_ANNOTATION: REQUEST_UID,
            "o2ims.provisioning.oran.org/downstream-package": PV_CREATE["cluster_name"],
        },
    },
    "spec": {"downstream": {"package": PV_CREATE["cluster_name"]}},
}
PR_OBJECT = {
    "apiVersion": "o2ims.provisioning.oran.org/v1alpha1",
    "kind": "ProvisioningRequest",
    "metadata": {"name": NAME},
    "status": {"provisioningStatus": {"provisioningState": "progressing"}},
}


TEST_TOKEN = "test-token"


@pytest.fixture(autouse=True)
def setup_and_teardown(monkeypatch, tmp_path):
    """Point TOKEN at a real token for the duration of one test.

    monkeypatch restores the environment afterwards; the previous version
    left TOKEN naming a deleted file, and wrote an empty token, so every
    request test silently exercised an empty Authorization header.
    """
    token_file = tmp_path / "token"
    token_file.write_text(TEST_TOKEN)
    monkeypatch.setenv("TOKEN", str(token_file))
    yield


@responses.activate
@pytest.mark.parametrize(
    "code, reason, retryable",
    [
        (401, "unauthorized", False),
        (403, "unauthorized", False),
        (404, "notFound", False),
        (409, "conflict", False),
        (400, "invalid", False),
        (422, "invalid", False),
        (429, "unavailable", True),
        (500, "unavailable", True),
        (503, "unavailable", True),
        (302, "protocol", False),
        (299, "protocol", False),
    ],
)
def test_an_api_answer_is_classified(code, reason, retryable):
    """One status, one classification. A caller that cannot tell a missing
    resource from an expired credential reports both as progress."""
    responses.get(
        f"{PACKAGE_VARIANTS_URI}/{NAME}",
        json={"kind": "Status", "status": "Failure", "message": "the server said no"},
        status=code,
    )
    with pytest.raises(ApiError) as raised:
        get_package_variant(NAME, NAMESPACE)
    assert raised.value.reason == reason
    assert raised.value.retryable is retryable
    assert raised.value.status_code == code


@responses.activate
def test_a_resource_is_returned_as_it_stands():
    responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}", json=TEST_JSON, status=200)
    assert get_package_variant(NAME, NAMESPACE) == TEST_JSON


@responses.activate
def test_a_transport_failure_is_retryable_and_names_no_write():
    responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}",
                  body=requests.exceptions.ConnectionError("no route"))
    with pytest.raises(ApiError) as raised:
        get_package_variant(NAME, NAMESPACE)
    assert raised.value.reason == "transport"
    assert raised.value.retryable is True
    assert raised.value.write_outcome_unknown is False


@responses.activate
def test_a_write_whose_answer_was_lost_is_settled_by_reading_it_back():
    """The answer never arrived, but the write did land."""
    responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}", json={"kind": "Status"}, status=404)
    responses.post(PACKAGE_VARIANTS_URI,
                   body=requests.exceptions.ConnectionError("connection reset"))
    responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}", json=OWNED_PV, status=200)

    assert ensure_package_variant(NAME, NAMESPACE, PV_CREATE, REQUEST_UID) == OWNED_PV


@responses.activate
def test_a_write_that_did_not_land_is_worth_another_attempt():
    """The read-back settles that nothing was written, which the read-back's
    own 404 would otherwise report as the resource simply not existing."""
    responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}", json={"kind": "Status"}, status=404)
    responses.post(PACKAGE_VARIANTS_URI,
                   body=requests.exceptions.ConnectionError("connection reset"))
    responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}", json={"kind": "Status"}, status=404)

    with pytest.raises(ApiError) as raised:
        ensure_package_variant(NAME, NAMESPACE, PV_CREATE, REQUEST_UID)
    assert raised.value.retryable is True
    assert "did not land" in str(raised.value)


@responses.activate
def test_a_body_that_is_not_an_object_is_a_protocol_error():
    responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}", body="not json", status=200)
    with pytest.raises(ApiError) as raised:
        get_package_variant(NAME, NAMESPACE)
    assert raised.value.reason == "protocol"


@responses.activate
def test_a_logger_does_not_change_the_outcome():
    """Reading the body to log it used to raise before the status was read."""
    for _ in range(2):
        responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}", body="<html>Forbidden</html>", status=403)
    with pytest.raises(ApiError) as quiet:
        get_package_variant(NAME, NAMESPACE)
    with pytest.raises(ApiError) as noisy:
        get_package_variant(NAME, NAMESPACE, logger=logging.getLogger("test"))
    assert quiet.value.reason == noisy.value.reason == "unauthorized"


@responses.activate
def test_every_call_is_bounded():
    responses.get(CAPI_URI, json=TEST_JSON, status=200)
    get_capi_cluster(NAME, NAMESPACE)
    assert responses.calls[0].request.req_kwargs["timeout"] == API_TIMEOUT


@responses.activate
def test_a_redirect_is_reported_and_not_followed():
    """The API server has no reason to redirect, and following one would hand
    the token to wherever it points."""
    responses.get(CAPI_URI, status=302, headers={"Location": "https://elsewhere.invalid/"})
    with pytest.raises(ApiError) as raised:
        get_capi_cluster(NAME, NAMESPACE)
    assert raised.value.reason == "protocol"
    assert len(responses.calls) == 1


@responses.activate
def test_an_existing_package_variant_of_this_request_is_accepted():
    responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}", json=OWNED_PV, status=200)
    assert ensure_package_variant(NAME, NAMESPACE, PV_CREATE, REQUEST_UID) == OWNED_PV


@responses.activate
@pytest.mark.parametrize("difference", ["uid", "target"])
def test_a_package_variant_of_something_else_is_not_this_request_fulfilled(difference):
    """The name is chosen by the request, so the name alone says nothing about
    who created it."""
    foreign = json.loads(json.dumps(OWNED_PV))
    if difference == "uid":
        foreign["metadata"]["annotations"][OWNER_UID_ANNOTATION] = "another-request"
    else:
        foreign["spec"]["downstream"]["package"] = "another-cluster"
    responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}", json=foreign, status=200)
    with pytest.raises(ApiError) as raised:
        ensure_package_variant(NAME, NAMESPACE, PV_CREATE, REQUEST_UID)
    assert raised.value.reason == "conflict"


@responses.activate
def test_a_created_package_variant_carries_labels_and_ownership():
    responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}", json={"kind": "Status"}, status=404)
    responses.post(PACKAGE_VARIANTS_URI, json=OWNED_PV, status=201)
    ensure_package_variant(NAME, NAMESPACE, PV_CREATE, REQUEST_UID)
    sent = json.loads(responses.calls[1].request.body)
    assert sent["metadata"]["labels"] == LABEL
    assert "label" not in sent["metadata"]
    assert sent["metadata"]["annotations"][OWNER_UID_ANNOTATION] == REQUEST_UID
    # #889 replaced a string revision with workspaceName; this keeps it.
    assert sent["spec"]["upstream"]["workspaceName"] == PV_CREATE["template_version"]


@responses.activate
def test_a_conflicting_create_is_settled_by_reading_the_resource_back():
    responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}", json={"kind": "Status"}, status=404)
    responses.post(PACKAGE_VARIANTS_URI, json={"kind": "Status"}, status=409)
    responses.get(f"{PACKAGE_VARIANTS_URI}/{NAME}", json=OWNED_PV, status=200)
    assert ensure_package_variant(NAME, NAMESPACE, PV_CREATE, REQUEST_UID) == OWNED_PV


@responses.activate
def test_a_provisioning_request_is_read_by_name():
    responses.get(f"{PROVISIONING_REQUEST_URI}/{NAME}", json=PR_OBJECT, status=200)
    assert check_o2ims_provisioning_request(NAME, NAMESPACE) == PR_OBJECT
    assert responses.calls[0].request.url.endswith(f"/{NAME}")
    assert "/namespaces/" not in responses.calls[0].request.url


@responses.activate
@pytest.mark.parametrize("answer", [
    {"kind": "ProvisioningRequestList", "items": []},
    {"kind": "ProvisioningRequest", "metadata": {"name": "somebody-else"}},
])
def test_an_answer_about_something_else_is_a_protocol_error(answer):
    """An empty collection used to read as this request progressing."""
    responses.get(f"{PROVISIONING_REQUEST_URI}/{NAME}", json=answer, status=200)
    with pytest.raises(ApiError) as raised:
        check_o2ims_provisioning_request(NAME, NAMESPACE)
    assert raised.value.reason == "protocol"


@responses.activate
def test_a_cluster_is_returned_with_the_current_token():
    responses.get(CAPI_URI, json=TEST_JSON, status=200)
    assert get_capi_cluster(NAME, NAMESPACE) == TEST_JSON
    assert responses.calls[0].request.headers["Authorization"] == f"Bearer {TEST_TOKEN}"


@responses.activate
def test_a_kubernetes_status_body_is_read_as_one():
    """The error body is a Status, not a Cluster with nested conditions."""
    responses.get(CAPI_URI, json={
        "kind": "Status", "status": "Failure", "reason": "ServiceUnavailable",
        "message": "try later", "code": 503,
    }, status=503)
    with pytest.raises(ApiError) as raised:
        get_capi_cluster(NAME, NAMESPACE)
    assert raised.value.retryable is True
    assert "try later" in str(raised.value)


@pytest.fixture
def no_ambient_tls_config(monkeypatch, tmp_path):
    """Make the tests behave the same on a laptop and inside a pod.

    The service variables matter as much as the CA path: tls_verify refuses
    to fall back to the public roots when it can see it is running in a pod,
    so leaving them set makes every default-behaviour test raise instead.
    """
    monkeypatch.setattr(
        utils, "IN_CLUSTER_CA_FILE", str(tmp_path / "absent-ca.crt")
    )
    for name in (
        "KUBERNETES_CA_FILE",
        "UNSAFE_SKIP_TLS_VERIFY",
        "KUBERNETES_SERVICE_HOST",
        "KUBERNETES_SERVICE_PORT",
        "KUBERNETES_BASE_URL",
        # requests rewrites verify=True to whichever bundle these name, so a
        # test asserting the default would see that path instead.
        "REQUESTS_CA_BUNDLE",
        "CURL_CA_BUNDLE",
    ):
        monkeypatch.delenv(name, raising=False)


# Every spelling worth asking requests about: the ones this validator is
# meant to accept, and the ones that have historically parsed differently
# here than they do on the wire.
URL_CORPUS = [
    "https://10.96.0.1:443",
    "https://10.96.0.1",
    "https://kubernetes.default.svc",
    "https://kubernetes.default.svc:443",
    "https://KUBERNETES.DEFAULT.SVC",
    "https://[fd00::1]:443",
    "https://[fd00:0:0:0:0:0:0:1]:443",
    "https://api.example.com:6443",
    "https://api.example.com:6443/prefix",
    "https://api.example.com:6443/prefix/",
    "http://10.96.0.1:443",
    "https://user:pass@10.96.0.1:443",
    "https://10.96.0.1:443?a=b",
    "https://10.96.0.1:443#frag",
    "https://10.96.0.1:0",
    "https://10.96.0.1:99999",
    "https://10.96.0.1:notaport",
    "https://[unclosed",
    "https://",
    "https://10.96.0.1" + chr(92) + "proxy",
    "https://10.96.0.1" + chr(92) + chr(92) + "evil.example",
    "https://api.example.com" + chr(92) + "path",
    "https://10.96.0.1 /proxy",
    "https://10.96.0.1\t.evil",
    "https://10.96.0.1\n",
    "https://10.96.0.1\x00.evil",
    "https://10.96.0.1\x7f.evil",
    "https:/" + chr(92) + "/" + chr(92) + "evil.example",
]


@pytest.mark.parametrize("base_url", URL_CORPUS)
def test_what_is_validated_is_what_requests_connects_to(base_url):
    """Whatever survives validation has to be an address requests agrees
    about.

    Guessing which characters urlsplit and urllib3 disagree over is how
    the control-character and backslash differentials got in. This asks
    requests instead, so a parser change in a dependency shows up here
    rather than as a token sent to an unchecked host.
    """
    try:
        validated = utils.validate_api_server_url(base_url)
    except RuntimeError:
        return  # refused, so there is no address to disagree about

    prepared = requests.Request("GET", validated).prepare().url
    assert utils.tls_origin(validated) == utils.tls_origin(prepared), (
        f"{validated!r} was accepted but requests prepares {prepared!r}"
    )


@pytest.mark.parametrize("left, right", [
    ("https://10.96.0.1", "https://10.96.0.1:443"),
    ("https://kubernetes.default.svc", "https://KUBERNETES.DEFAULT.SVC:443"),
    ("https://[::ffff:a60:1]:443", "https://[0:0:0:0:0:ffff:a60:1]:443"),
    ("https://host:6443/api", "https://host:6443/healthz"),
])
def test_one_endpoint_has_one_origin(left, right):
    assert utils.tls_origin(left) == utils.tls_origin(right)


@pytest.mark.parametrize("left, right", [
    ("https://10.96.0.1:443", "http://10.96.0.1:443"),
    ("https://10.96.0.1:443", "https://10.96.0.1:6443"),
    ("https://10.96.0.1:443", "https://10.96.0.2:443"),
    ("https://kubernetes.default.svc", "https://kubernetes.default.svc.other"),
])
def test_different_endpoints_have_different_origins(left, right):
    assert utils.tls_origin(left) != utils.tls_origin(right)


def test_tls_is_verified_by_default(no_ambient_tls_config):
    # True is what requests calls "verify against the system trust store"
    assert utils.tls_verify(utils.kubernetes_base_url()) is True


def usable_ca_bundle(tmp_path, name):
    """Return a path holding a bundle OpenSSL will actually load."""
    bundle = tmp_path / name
    bundle.write_bytes(open(certifi.where(), "rb").read())
    return str(bundle)


def test_in_cluster_ca_bundle_is_used_when_present(
    no_ambient_tls_config, monkeypatch, tmp_path
):
    ca_file = usable_ca_bundle(tmp_path, "ca.crt")
    monkeypatch.setattr(utils, "IN_CLUSTER_CA_FILE", ca_file)
    assert utils.tls_verify(utils.kubernetes_base_url()) == ca_file


def test_another_endpoint_keeps_the_default_bundle(
    no_ambient_tls_config, monkeypatch, tmp_path
):
    """The mounted bundle belongs to the cluster's API server, not to
    whatever else this operator is pointed at."""
    monkeypatch.setattr(
        utils, "IN_CLUSTER_CA_FILE", usable_ca_bundle(tmp_path, "ca.crt"))
    monkeypatch.setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
    monkeypatch.setenv("KUBERNETES_SERVICE_PORT", "443")
    monkeypatch.setenv("KUBERNETES_BASE_URL", "https://other.example:6443")
    assert utils.tls_verify(utils.kubernetes_base_url()) is True


def test_the_advertised_address_uses_the_mounted_bundle(
    no_ambient_tls_config, monkeypatch, tmp_path
):
    ca_file = usable_ca_bundle(tmp_path, "ca.crt")
    monkeypatch.setattr(utils, "IN_CLUSTER_CA_FILE", ca_file)
    monkeypatch.setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
    monkeypatch.setenv("KUBERNETES_SERVICE_PORT", "443")
    for base_url in ("https://10.96.0.1:443", utils.DEFAULT_KUBERNETES_BASE_URL):
        monkeypatch.setenv("KUBERNETES_BASE_URL", base_url)
        assert utils.tls_verify(utils.kubernetes_base_url()) == ca_file


@pytest.mark.parametrize("base_url", [
    "https://10.96.0.1:443",
    "https://10.96.0.1",                      # 443 is implicit
    "https://kubernetes.default.svc",
    "https://kubernetes.default.svc:443",
    "https://KUBERNETES.DEFAULT.SVC",
    "https://10.96.0.1:443/",
])
def test_equivalent_cluster_addresses_use_the_mounted_bundle(
    no_ambient_tls_config, monkeypatch, tmp_path, base_url
):
    """The certificate does not change with the notation, so neither
    should the bundle chosen to check it."""
    ca_file = usable_ca_bundle(tmp_path, "ca.crt")
    monkeypatch.setattr(utils, "IN_CLUSTER_CA_FILE", ca_file)
    monkeypatch.setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
    monkeypatch.setenv("KUBERNETES_SERVICE_PORT", "443")
    monkeypatch.setenv("KUBERNETES_BASE_URL", base_url)
    assert utils.tls_verify(utils.kubernetes_base_url()) == ca_file


@pytest.mark.parametrize("host, base_url", [
    ("fd00::1", "https://[fd00:0:0:0:0:0:0:1]:443"),
    ("fd00::1", "https://[FD00::1]:443"),
])
def test_equivalent_ipv6_spellings_use_the_mounted_bundle(
    no_ambient_tls_config, monkeypatch, tmp_path, host, base_url
):
    ca_file = usable_ca_bundle(tmp_path, "ca.crt")
    monkeypatch.setattr(utils, "IN_CLUSTER_CA_FILE", ca_file)
    monkeypatch.setenv("KUBERNETES_SERVICE_HOST", host)
    monkeypatch.setenv("KUBERNETES_SERVICE_PORT", "443")
    monkeypatch.setenv("KUBERNETES_BASE_URL", base_url)
    assert utils.tls_verify(utils.kubernetes_base_url()) == ca_file


@pytest.mark.parametrize("base_url", [
    "https://10.96.0.1:6443",                 # same host, different port
    "https://10.96.0.2:443",
    "https://kubernetes.default.svc.other:443",
])
def test_a_different_origin_keeps_the_default_bundle(
    no_ambient_tls_config, monkeypatch, tmp_path, base_url
):
    monkeypatch.setattr(
        utils, "IN_CLUSTER_CA_FILE", usable_ca_bundle(tmp_path, "ca.crt"))
    monkeypatch.setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
    monkeypatch.setenv("KUBERNETES_SERVICE_PORT", "443")
    monkeypatch.setenv("KUBERNETES_BASE_URL", base_url)
    assert utils.tls_verify(utils.kubernetes_base_url()) is True


def test_kubernetes_ca_file_wins(no_ambient_tls_config, monkeypatch, tmp_path):
    monkeypatch.setattr(
        utils, "IN_CLUSTER_CA_FILE", usable_ca_bundle(tmp_path, "in-cluster.crt"))
    configured = usable_ca_bundle(tmp_path, "configured.crt")
    monkeypatch.setenv("KUBERNETES_CA_FILE", configured)
    assert utils.tls_verify(utils.kubernetes_base_url()) == configured


def test_a_missing_ca_file_is_refused(no_ambient_tls_config, monkeypatch, tmp_path):
    missing = str(tmp_path / "missing.crt")
    monkeypatch.setenv("KUBERNETES_CA_FILE", missing)
    with pytest.raises(RuntimeError, match=missing):
        utils.tls_verify(utils.kubernetes_base_url())


@pytest.mark.parametrize("content", ["", "   ", "not a pem\n", "-----BEGIN-----"])
def test_an_unusable_ca_bundle_is_refused(
    no_ambient_tls_config, monkeypatch, tmp_path, content
):
    """A path check alone would defer this to the first request."""
    bundle = tmp_path / "ca.crt"
    bundle.write_text(content)
    monkeypatch.setenv("KUBERNETES_CA_FILE", str(bundle))
    with pytest.raises(RuntimeError, match="unusable"):
        utils.tls_verify(utils.kubernetes_base_url())


def test_a_ca_bundle_and_skip_verify_together_are_refused(
    no_ambient_tls_config, monkeypatch, tmp_path
):
    monkeypatch.setenv("KUBERNETES_CA_FILE", usable_ca_bundle(tmp_path, "ca.crt"))
    monkeypatch.setenv("UNSAFE_SKIP_TLS_VERIFY", "true")
    with pytest.raises(RuntimeError, match="conflict"):
        utils.tls_verify(utils.kubernetes_base_url())


def test_a_missing_in_cluster_ca_is_refused(
    no_ambient_tls_config, monkeypatch, tmp_path
):
    """In a pod, an absent CA must not fall back to the public roots."""
    monkeypatch.setattr(utils, "IN_CLUSTER_CA_FILE", str(tmp_path / "absent.crt"))
    monkeypatch.setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
    with pytest.raises(RuntimeError, match="in-cluster CA"):
        utils.tls_verify(utils.kubernetes_base_url())


def test_a_plaintext_endpoint_is_rejected_and_never_contacted(monkeypatch):
    """requests ignores verify for http, so the token would go out in clear."""
    contacted = []

    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            contacted.append(self.headers.get("Authorization"))
            self.send_response(200)
            self.end_headers()

        def log_message(self, *args):
            pass

    server = http.server.HTTPServer(("127.0.0.1", 0), Handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    try:
        monkeypatch.setenv(
            "KUBERNETES_BASE_URL", f"http://127.0.0.1:{server.server_address[1]}")
        with pytest.raises(RuntimeError, match="must use https"):
            utils.kubernetes_base_url()
    finally:
        server.shutdown()

    assert contacted == []


@pytest.mark.parametrize("base_url", [
    "kubernetes.default.svc",
    "https://",
    "https://user:pass@api.example.com:6443",
    "https://api.example.com:6443?token=x",
    "https://api.example.com:6443#f",
    "https://api.example.com:not-a-port",
    "https://api.example.com:0",
    # urlsplit and requests disagree about these: the host validated here
    # would not be the host connected to.
    "https://api.example.com\t.evil",
    "https://api.example.com\n",
    "https:/\\/\\evil.example",
])
def test_an_unusable_endpoint_is_rejected(monkeypatch, base_url):
    monkeypatch.setenv("KUBERNETES_BASE_URL", base_url)
    with pytest.raises(RuntimeError):
        utils.kubernetes_base_url()


@pytest.mark.parametrize("base_url, problem", [
    ("https://[unclosed", "cannot be parsed"),
    ("https://host:notaport", "invalid port"),
    ("https://host:99999", "invalid port"),
])
def test_an_unparseable_endpoint_is_rejected_as_such(base_url, problem):
    """urlsplit raises on some of these and .port on others; neither
    should reach the caller as a bare ValueError."""
    with pytest.raises(RuntimeError, match=problem):
        utils.validate_api_server_url(base_url)


@pytest.mark.parametrize("base_url", [
    "https://api.example.com\x00.evil",
    "https://api.example.com\x7f.evil",
    "https://api.example.com\x1f.evil",
])
def test_a_control_character_endpoint_is_rejected(base_url):
    """os.environ refuses a NUL, so these go straight to the validator.

    urlsplit drops some of these and keeps others while requests
    percent-encodes them, so the host validated is not the host connected to.
    """
    with pytest.raises(RuntimeError, match="control characters"):
        utils.validate_api_server_url(base_url)


@pytest.mark.parametrize("base_url", [
    "https://kubernetes.default.svc",
    "https://kubernetes.default.svc:443",
    "https://kubernetes.default.svc/",
])
def test_the_legacy_catalog_address_resolves_to_the_advertised_one(
    no_ambient_tls_config, monkeypatch, base_url
):
    """catalog#146 removes this value, but it cannot land in the same
    commit, so a pod running the new image with the old package still
    has to reach a name the certificate covers."""
    monkeypatch.setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
    monkeypatch.setenv("KUBERNETES_SERVICE_PORT", "443")
    monkeypatch.setenv("KUBERNETES_BASE_URL", base_url)
    assert utils.kubernetes_base_url() == "https://10.96.0.1:443"


@pytest.mark.parametrize("name, value", [
    ("KUBERNETES_CA_FILE", "/etc/o2ims/proxy-ca.crt"),
    ("UNSAFE_SKIP_TLS_VERIFY", "true"),
])
def test_the_legacy_address_is_kept_when_trust_was_configured(
    no_ambient_tls_config, monkeypatch, name, value
):
    """A bundle names the endpoint it certifies. Substituting the address
    under it turns a working proxy into a hostname mismatch."""
    monkeypatch.setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
    monkeypatch.setenv("KUBERNETES_SERVICE_PORT", "443")
    monkeypatch.setenv("KUBERNETES_BASE_URL", "https://kubernetes.default.svc")
    monkeypatch.setenv(name, value)
    assert utils.kubernetes_base_url() == "https://kubernetes.default.svc"


def test_the_legacy_address_is_kept_outside_a_pod(no_ambient_tls_config,
                                                  monkeypatch):
    monkeypatch.setenv("KUBERNETES_BASE_URL", "https://kubernetes.default.svc")
    assert utils.kubernetes_base_url() == "https://kubernetes.default.svc"


@pytest.mark.parametrize("base_url", [
    "https://api.example.com:6443",
    "https://kubernetes.default.svc/proxy-prefix",
])
def test_any_other_endpoint_is_respected(no_ambient_tls_config, monkeypatch,
                                         base_url):
    monkeypatch.setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
    monkeypatch.setenv("KUBERNETES_SERVICE_PORT", "443")
    monkeypatch.setenv("KUBERNETES_BASE_URL", base_url)
    assert utils.kubernetes_base_url() == base_url


def test_a_trailing_slash_is_normalised(monkeypatch):
    monkeypatch.setenv("KUBERNETES_BASE_URL", "https://api.example.com:6443/")
    assert utils.kubernetes_base_url() == "https://api.example.com:6443"


@pytest.mark.parametrize(
    "value, verification_skipped",
    [
        ("true", True),
        ("True", True),
        ("1", True),
        ("yes", True),
        ("on", True),
        ("false", False),
        ("False", False),
        ("0", False),
        ("no", False),
        ("", False),
        ("  ", False),
        ("maybe", False),
    ],
)
def test_only_an_explicit_opt_in_skips_verification(
    no_ambient_tls_config, monkeypatch, value, verification_skipped
):
    monkeypatch.setenv("UNSAFE_SKIP_TLS_VERIFY", value)
    assert (utils.tls_verify(utils.kubernetes_base_url()) is False) == verification_skipped


@responses.activate
@pytest.mark.parametrize("verify", [True, "/etc/ssl/certs/test-ca.crt"])
def test_requests_carry_the_verify_setting(monkeypatch, verify):
    monkeypatch.delenv("REQUESTS_CA_BUNDLE", raising=False)
    monkeypatch.delenv("CURL_CA_BUNDLE", raising=False)
    monkeypatch.setattr(utils, "TLS_VERIFY", verify)
    responses.get(CAPI_URI, json=TEST_JSON, status=200)
    get_capi_cluster(NAME, NAMESPACE)
    assert responses.calls[0].request.req_kwargs["verify"] == verify


def exercise_ensure_package_variant():
    """Two requests: the lookup that misses, then the creation."""
    responses.get(PACKAGE_VARIANTS_URI + f"/{NAME}", json={"kind": "Status"}, status=404)
    responses.post(PACKAGE_VARIANTS_URI, json=OWNED_PV, status=201)
    ensure_package_variant(NAME, NAMESPACE, PV_CREATE, REQUEST_UID)


def exercise_get_package_variant():
    responses.get(PACKAGE_VARIANTS_URI + f"/{NAME}", json=TEST_JSON, status=200)
    get_package_variant(NAME, NAMESPACE)


def exercise_check_o2ims_provisioning_request():
    responses.get(f"{PROVISIONING_REQUEST_URI}/{NAME}", json=PR_OBJECT, status=200)
    check_o2ims_provisioning_request(NAME, NAMESPACE)


def exercise_get_capi_cluster():
    responses.get(CAPI_URI, json=TEST_JSON, status=200)
    get_capi_cluster(NAME, NAMESPACE)


@responses.activate
@pytest.mark.parametrize("exercise", [
    exercise_ensure_package_variant,
    exercise_get_package_variant,
    exercise_check_o2ims_provisioning_request,
    exercise_get_capi_cluster,
])
def test_every_api_call_verifies_and_authenticates(monkeypatch, exercise):
    """Every request that carries the token has to check the certificate.

    Asserting this on one path leaves the others free to lose either,
    and both are one keyword argument each.
    """
    monkeypatch.delenv("REQUESTS_CA_BUNDLE", raising=False)
    monkeypatch.delenv("CURL_CA_BUNDLE", raising=False)
    monkeypatch.setattr(utils, "TLS_VERIFY", "/etc/pki/expected.crt")

    exercise()

    assert responses.calls, "the exercise made no request"
    for call in responses.calls:
        assert call.request.req_kwargs["verify"] == "/etc/pki/expected.crt"
        assert call.request.headers["Authorization"] == f"Bearer {TEST_TOKEN}"


@pytest.mark.parametrize(
    "environment, expected",
    [
        (
            {"KUBERNETES_BASE_URL": "https://api.example.com:6443"},
            "https://api.example.com:6443",
        ),
        (
            {"KUBERNETES_SERVICE_HOST": "10.96.0.1",
             "KUBERNETES_SERVICE_PORT": "443"},
            "https://10.96.0.1:443",
        ),
        ({"KUBERNETES_SERVICE_HOST": "fd00::1"}, "https://[fd00::1]:443"),
        ({"KUBERNETES_SERVICE_HOST": "[fd00::1]"}, "https://[fd00::1]:443"),
        ({}, "https://kubernetes.default.svc"),
    ],
)
def test_kubernetes_base_url(monkeypatch, environment, expected):
    for name in (
        "KUBERNETES_BASE_URL",
        "KUBERNETES_SERVICE_HOST",
        "KUBERNETES_SERVICE_PORT",
    ):
        monkeypatch.delenv(name, raising=False)
    for name, value in environment.items():
        monkeypatch.setenv(name, value)
    assert utils.kubernetes_base_url() == expected

# Service account token tests


def test_a_token_path_is_never_passed_to_a_shell(monkeypatch, tmp_path):
    """A path is opened, not executed, however it is spelled."""
    marker = tmp_path / "pwned"
    monkeypatch.setenv("TOKEN", f"{tmp_path}/absent; touch {marker}")
    with pytest.raises(OSError):
        utils.read_token()
    assert not marker.exists()


def test_a_missing_token_file_is_not_silently_empty(monkeypatch, tmp_path):
    """An unreadable token must not degrade into an empty Bearer header."""
    monkeypatch.setenv("TOKEN", str(tmp_path / "absent"))
    with pytest.raises(OSError):
        utils.read_token()


@pytest.mark.parametrize("content", ["tok", "tok\n", "  tok\r\n"])
def test_token_surroundings_are_stripped(monkeypatch, tmp_path, content):
    """requests rejects a header value with a trailing newline."""
    token_file = tmp_path / "token"
    token_file.write_text(content)
    monkeypatch.setenv("TOKEN", str(token_file))
    assert utils.read_token() == "tok"


@responses.activate
def test_each_request_carries_the_current_token(monkeypatch, tmp_path):
    """Kubernetes rotates the projected token while the operator runs."""
    token_file = tmp_path / "token"
    token_file.write_text("first")
    monkeypatch.setenv("TOKEN", str(token_file))
    responses.get(CAPI_URI, json=TEST_JSON, status=200)
    responses.get(CAPI_URI, json=TEST_JSON, status=200)

    get_capi_cluster(NAME, NAMESPACE)
    token_file.write_text("second")
    get_capi_cluster(NAME, NAMESPACE)

    sent = [call.request.headers["Authorization"] for call in responses.calls]
    assert sent == ["Bearer first", "Bearer second"]


@pytest.mark.parametrize("content", ["", " ", "\n", "\r\n", " \t \r\n"])
def test_an_empty_or_blank_token_is_refused(monkeypatch, tmp_path, content):
    """An empty file used to become `Authorization: Bearer `."""
    token_file = tmp_path / "token"
    token_file.write_text(content)
    monkeypatch.setenv("TOKEN", str(token_file))
    with pytest.raises(RuntimeError, match="empty"):
        utils.read_token()


@pytest.mark.parametrize("content", ["tok en", "tok\nen", "tok\ten"])
def test_a_token_with_internal_whitespace_is_refused(
    monkeypatch, tmp_path, content
):
    """requests accepts such a header value; http.client rejects it on the wire."""
    token_file = tmp_path / "token"
    token_file.write_text(content)
    monkeypatch.setenv("TOKEN", str(token_file))
    with pytest.raises(RuntimeError, match="not a token"):
        utils.read_token()


@responses.activate
def test_an_invalid_token_stops_before_any_request_is_made(monkeypatch, tmp_path):
    token_file = tmp_path / "token"
    token_file.write_text("")
    monkeypatch.setenv("TOKEN", str(token_file))
    responses.get(CAPI_URI, json=TEST_JSON, status=200)

    with pytest.raises(ApiError) as raised:
        get_capi_cluster(NAME, NAMESPACE)

    # A configuration failure of its own, not something to wait out.
    assert raised.value.reason == "config"
    assert len(responses.calls) == 0


@responses.activate
def test_ensure_package_variant_refreshes_the_token_between_get_and_post(
    monkeypatch, tmp_path
):
    """The GET and the POST in one call must not share a stale token."""
    token_file = tmp_path / "token"
    token_file.write_text("first")
    monkeypatch.setenv("TOKEN", str(token_file))

    def rotate(request):
        token_file.write_text("second")
        return (404, {}, json.dumps({"kind": "Status"}))

    responses.add_callback(
        responses.GET, f"{PACKAGE_VARIANTS_URI}/{NAME}", callback=rotate,
        content_type="application/json")
    responses.post(PACKAGE_VARIANTS_URI, json=OWNED_PV, status=201)

    ensure_package_variant(NAME, NAMESPACE, PV_CREATE, REQUEST_UID)

    sent = [call.request.headers["Authorization"] for call in responses.calls]
    assert sent == ["Bearer first", "Bearer second"]
