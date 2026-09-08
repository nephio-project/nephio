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
import os
import random
import string
import threading

import certifi
import pytest
import requests
import responses

from controllers import utils
from controllers.utils import (
    KUBERNETES_BASE_URL,
    create_package_variant,
    get_package_variant,
    check_o2ims_provisioning_request,
    get_capi_cluster,
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
    "get_code, post_code, status, create, response_2, response_2_value, exception",
    [
        (200, None, True, False, "name", NAME, False),
        (401, None, False, False, "reason", "unauthorized", False),
        (403, None, False, False, "reason", "unauthorized", False),
        (404, 200, True, True, "name", NAME, False),
        (404, 201, True, True, "name", NAME, False),
        (404, 401, False, True, "reason", "unauthorized", False),
        (404, 403, False, True, "reason", "unauthorized", False),
        (404, 404, False, True, "reason", "notFound", False),
        (404, 400, False, True, "reason", TEST_JSON["message"], False),
        (404, 1234, False, True, "reason", TEST_JSON, False),
        (404, None, False, True, "reason", "NotAbleToCommunicateWithTheCluster ", True),
        (404, 200, False, False, "reason", "notFound", False),
        (500, None, False, False, "reason", "k8sApi server is not reachable", False),
        (1234, None, False, False, "reason", TEST_JSON, False),
        (None, None, False, False, "reason", "NotAbleToCommunicateWithTheCluster ", True),
    ],
)
def test_create_package_variant(get_code, post_code, status, create, response_2, response_2_value, exception):
    if not exception:
        responses.get(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            json=TEST_JSON,
            status=get_code,
        )
    else:
        responses.get(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            body=Exception(""),
        )

    pv_params = PV_PARAM.copy()
    if get_code == 404 and create:
        responses.post(
            PACKAGE_VARIANTS_URI,
            json=TEST_JSON,
            status=post_code,
        )
        pv_params.update({"create": True})

    response = create_package_variant(NAME, NAMESPACE, pv_params)
    assert response["status"] == status and response[response_2] == response_2_value


@responses.activate
@pytest.mark.parametrize(
    "http_code, status, response_2, response_2_value, response_3, response_3_value, exception",
    [
        (200, True, "name", NAME, "body", TEST_JSON, False),
        (401, False, "reason", "unauthorized", None, None, False),
        (403, False, "reason", "unauthorized", None, None, False),
        (404, False, "reason", "notFound", None, None, False),
        (1234, False, "reason", TEST_JSON, None, None, False),
        (None, False, "reason", "NotAbleToCommunicateWithTheCluster ", None, None, True),
    ],
)
def test_get_package_variant(
    http_code,
    status,
    response_2,
    response_2_value,
    response_3,
    response_3_value,
    exception,
):
    if not exception:
        responses.get(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            json=TEST_JSON,
            status=http_code,
        )
    else:
        responses.get(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            body=Exception(""),
        )
    response = get_package_variant(NAME, NAMESPACE)
    assert response["status"] == status and response[response_2] == response_2_value
    if response_3:
        assert response[response_3] == response_3_value


@responses.activate
@pytest.mark.parametrize(
    "pr_code, status, status_response, pv_code, response_2, response_2_value, response_3, response_3_value, response_3_exception, exception",
    [
        (200, True, True, None, "provisioningStatus", PR_PARAMS["status"]["provisioningStatus"], None, None, None, False),
        (
            200,
            True,
            False,
            None,
            "provisioningStatus",
            {
                "provisioningMessage": "Cluster provisioning request received",
                "provisioningState": "progressing",
            },
            None,
            None,
            None,
            False,
        ),
        (401, False, False, None, "reason", "unauthorized", None, None, None, False),
        (403, False, False, None, "reason", "unauthorized", None, None, None, False),
        (404, False, False, 200, "reason", "notFound", "pv", True, None, False),
        (404, False, False, 401, "reason", "notFound", "pv", False, None, False),
        (404, False, False, 403, "reason", "notFound", "pv", False, None, False),
        (404, False, False, 404, "reason", "notFound", "pv", False, None, False),
        (404, False, False, 1234, "reason", "notFound", "pv", False, None, False),
        (404, False, False, None, "reason", "notFound", "pv", False, True, False),
        (1234, False, False, None, "reason", PR_PARAMS, None, None, None, False),
        (None, False, False, None, "reason", "NotAbleToCommunicateWithTheCluster ", None, None, None, True),
    ],
)
def test_check_o2ims_provisioning_request(
    pr_code,
    status,
    status_response,
    pv_code,
    response_2,
    response_2_value,
    response_3,
    response_3_value,
    response_3_exception,
    exception,
):
    if not exception:
        pr_params = PR_PARAMS.copy()
        if pr_code == 200 and not status_response:
            pr_params.pop("status")

        responses.get(
            PROVISIONING_REQUEST_URI,
            json=pr_params,
            status=pr_code,
        )

    else:
        responses.get(
            PROVISIONING_REQUEST_URI,
            body=Exception(""),
        )

    if pv_code and not response_3_exception:
        responses.get(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            json=TEST_JSON,
            status=pv_code,
        )
    elif pv_code and response_3_exception:
        responses.get(
            f"{PACKAGE_VARIANTS_URI}/{NAME}",
            body=Exception(""),
        )
    response = check_o2ims_provisioning_request(NAME, NAMESPACE)
    print(response)
    assert response["status"] == status and response[response_2] == response_2_value

    if pv_code:
        assert response[response_3] == response_3_value


@responses.activate
@pytest.mark.parametrize(
    "http_code, status, response_2, response_2_value, exception",
    [
        (200, True, "body", TEST_JSON, False),
        (401, False, "reason", "unauthorized", False),
        (403, False, "reason", "unauthorized", False),
        (404, False, "reason", "notFound", False),
        (1234, False, "reason", TEST_JSON["status"]["conditions"][0]["message"], False),
        (None, False, "reason", "NotAbleToCommunicateWithTheCluster ", True),
    ],
)
def test_get_capi_cluster(http_code, status, response_2, response_2_value, exception):
    if not exception:
        responses.get(
            CAPI_URI,
            json=TEST_JSON,
            status=http_code,
        )
    else:
        responses.get(
            CAPI_URI,
            body=Exception(""),
        )
    response = get_capi_cluster(NAME, NAMESPACE)
    assert response["status"] == status and response[response_2] == response_2_value
    if not exception:
        sent = responses.calls[0].request.headers["Authorization"]
        assert sent == f"Bearer {TEST_TOKEN}"


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


def exercise_create_package_variant():
    """Two requests: the lookup that misses, then the creation."""
    responses.get(PACKAGE_VARIANTS_URI + f"/{NAME}", json=TEST_JSON, status=404)
    responses.post(PACKAGE_VARIANTS_URI, json=TEST_JSON, status=201)
    params = PV_PARAM.copy()
    params.update({"create": True})
    create_package_variant(NAME, NAMESPACE, params)


def exercise_get_package_variant():
    responses.get(PACKAGE_VARIANTS_URI + f"/{NAME}", json=TEST_JSON, status=200)
    get_package_variant(NAME, NAMESPACE)


def exercise_check_o2ims_provisioning_request():
    responses.get(
        PROVISIONING_REQUEST_URI,
        json={"status": {"provisioningStatus": "fulfilled"}},
        status=200,
    )
    check_o2ims_provisioning_request(NAME, NAMESPACE)


def exercise_get_capi_cluster():
    responses.get(CAPI_URI, json=TEST_JSON, status=200)
    get_capi_cluster(NAME, NAMESPACE)


@responses.activate
@pytest.mark.parametrize("exercise", [
    exercise_create_package_variant,
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

    get_capi_cluster(NAME, NAMESPACE)

    assert len(responses.calls) == 0


@responses.activate
def test_create_package_variant_refreshes_the_token_between_get_and_post(
    monkeypatch, tmp_path
):
    """The GET and the POST in one call must not share a stale token."""
    token_file = tmp_path / "token"
    token_file.write_text("first")
    monkeypatch.setenv("TOKEN", str(token_file))

    def rotate(request):
        token_file.write_text("second")
        return (404, {}, json.dumps(TEST_JSON))

    responses.add_callback(
        responses.GET, f"{PACKAGE_VARIANTS_URI}/{NAME}", callback=rotate,
        content_type="application/json")
    responses.post(PACKAGE_VARIANTS_URI, json=TEST_JSON, status=201)

    params = PV_PARAM.copy()
    params.update({"create": True})
    create_package_variant(NAME, NAMESPACE, params)

    sent = [call.request.headers["Authorization"] for call in responses.calls]
    assert sent == ["Bearer first", "Bearer second"]
