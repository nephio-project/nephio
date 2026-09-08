# Nephio O-RAN O2 IMS Operator

This operator implements O-RAN O2 IMS for K8s based cloud management. 

## How to start

### Development Requirements:

- >= Python3.11
- requirements.txt installed in development environment

### Nephio Management Cluster Requirements:

- 6 vCPU
- 10Gi RAM

## Create Development Environment


### Including Nephio mgmt Cluster

The following will create a kind cluster and install required components such as:
- Porch
- ConfigSync
- Gitea (available at `172.18.0.200:3000`)
- MetalLB and MetalLB Sandbox Environment
- CAPI
- ConfigSync and RootSync objects to create clusters

It will also configure a secret which the operator can use for development purposes (when running the operator in non-containerize environments). It creates a pod and appends the `porch-controllers` service account token and redirects it from `/var/run/secrets/kubernetes.io/serviceaccount/token` to `/tmp/porch-token`.


```bash
# Get the repository
git clone https://github.com/nephio-project/nephio.git
cd operators/o2ims-operator
# Create a virtual environment
virtualenv venv -p python3
source venv/bin/activate
# Install requirements
pip install -r requirements.txt
# Set kernel parameters (run these commands after system restart or when new VM/system is created)
sudo sysctl -w fs.inotify.max_user_watches=524288
sudo sysctl -w fs.inotify.max_user_instances=512
sudo sysctl -w kernel.keys.maxkeys=500000
sudo sysctl -w kernel.keys.maxbytes=1000000
# Run the create-cluster.sh script to create the mgmt cluster and development environment
./tests/create-cluster.sh
```

Operator CRD is can be fetched via below command, though the above cluster creation script automatically fetches and apply this CRD.

```bash
curl --create-dirs -O --output-dir ./config/crd/bases/ https://raw.githubusercontent.com/nephio-project/api/refs/heads/main/config/crd/bases/o2ims.provisioning.oran.org_provisioningrequests.yaml
```

### Existing Nephio mgmt Cluster

#### Non-containerized Development Environment

Already setup by the test/create-cluster.sh go to To Start the Operator.

#### Containerized Development Environment

Build a Docker image: 

```bash
cd operators/o2ims-operator/
make docker-build
```

Push this image in your cluster, here we are using a `kind` cluster so we will push using the below command:

```bash
kind load docker-image nephio/o2ims-operator:latest -n o2ims-mgmt
```

`NOTE`: `o2ims-mgmt` is the name of the kind cluster. It is good to mention cluster name if you have multiple clusters.

Deploy the O2 IMS operator:

```bash
kpt pkg get --for-deployment https://github.com/nephio-project/catalog.git/nephio/optional/o2ims@origin/main /tmp/o2ims
kpt fn render /tmp/o2ims
kpt live init /tmp/o2ims
kpt live apply /tmp/o2ims --reconcile-timeout=15m --output=table
```

### To Start the Operator: 

Note that there are some constants in manager.py that can be tuned before running the operator.

```bash
## To run in debug mode use the "--debug" flag or "-v --log-format=full"
kopf run controllers/manager.py
```

Open another terminal to provision a cluster:

```bash
kpt pkg get --for-deployment https://github.com/nephio-project/catalog.git
/nephio/optional/o2ims@origin/main /tmp/o2ims
kpt fn render /tmp/o2ims
kpt live init /tmp/o2ims
kpt live apply /tmp/o2ims --reconcile-timeout=15m --output=table
```

### Connecting to the Kubernetes API

The operator verifies the API server's certificate on every request. Inside a
pod there is nothing to configure: the CA bundle mounted at
`/var/run/secrets/kubernetes.io/serviceaccount/ca.crt` is used, and the address
comes from `KUBERNETES_SERVICE_HOST`, which is the one Kubernetes expects that
certificate to be valid for.

| Variable | Default | Meaning |
|----------|---------|---------|
| `KUBERNETES_BASE_URL` | in-cluster address | API server to talk to |
| `KUBERNETES_CA_FILE` | in-cluster CA bundle | PEM bundle used to verify the API server |
| `UNSAFE_SKIP_TLS_VERIFY` | `false` | Development only: skip certificate verification |

Running the operator outside a cluster, point it at the API server and at the
CA that signs its certificate:

```bash
# A function, so that a failure stops the setup without closing an
# interactive shell, and so the token refresh can be re-run on its own.
refresh_o2ims_token() {
  tmp=$(mktemp /tmp/o2ims-token.XXXXXX) || return 1
  chmod 0600 "$tmp" || { rm -f "$tmp"; return 1; }

  if ! kubectl -n o2ims create token o2ims-operator --duration=1h > "$tmp" ||
     ! test -s "$tmp"; then
    rm -f "$tmp"
    echo "token refresh failed; the previous token is unchanged" >&2
    return 1
  fi

  mv -f "$tmp" /tmp/o2ims-token
}

o2ims_dev_env() {
  # Assigned before exporting: export always succeeds, so it would hide a
  # kubectl that failed, and an empty value falls back to the in-cluster name.
  KUBERNETES_BASE_URL=$(kubectl config view --minify \
    -o jsonpath='{.clusters[0].cluster.server}') || return 1
  test -n "$KUBERNETES_BASE_URL" || {
    echo "kubeconfig names no API server" >&2
    return 1
  }

  # --flatten embeds the CA whether the kubeconfig holds it inline or as a
  # path, and a path is relative to the kubeconfig rather than to this shell.
  # The pipeline's status is base64's, and base64 succeeds on the empty input
  # a failed kubectl leaves, so the size check is what catches that.
  ca=$(mktemp /tmp/o2ims-ca.XXXXXX) || return 1
  if ! kubectl config view --raw --minify --flatten \
        -o jsonpath='{.clusters[0].cluster.certificate-authority-data}' \
        | base64 -d > "$ca" || ! test -s "$ca"; then
    rm -f "$ca"
    echo "could not extract a non-empty cluster CA from the kubeconfig" >&2
    return 1
  fi
  mv -f "$ca" /tmp/cluster-ca.crt

  # TOKEN is a path, not a token. Outside a pod there is no mounted one, so
  # request a short-lived credential and point at the file. A refresh that
  # fails is only fatal when there is no usable token from a previous one.
  refresh_o2ims_token || test -s /tmp/o2ims-token || return 1

  export KUBERNETES_BASE_URL
  export KUBERNETES_CA_FILE=/tmp/cluster-ca.crt
  export TOKEN=/tmp/o2ims-token
}

o2ims_dev_env
```

The rename matters. The operator opens this file on every request, and `>`
truncates before `kubectl` finishes writing, so a request landing in that window
would read an empty token. A rename on the same filesystem is atomic: a reader
sees either the whole old file or the whole new one.

So does keeping the failure off that path. The redirection creates the
temporary file before `kubectl` runs, so an expired kubeconfig or an
unreachable API server leaves it empty; renaming that over a token that is
still valid causes the outage the rename exists to prevent, and the operator
reports `Kubernetes token file ... is empty`. The function above discards the
temporary file instead and leaves the old one in place.

Re-run `refresh_o2ims_token` when the token expires. Inside a pod none of this
applies, because the kubelet rotates the mounted one.

`UNSAFE_SKIP_TLS_VERIFY=true` turns verification off altogether. It hands the
service account token to an endpoint whose identity has not been checked, warns
about it at startup, and must never be used outside development.

### Redeploying

To redeploy the cluster, or to recreate the development environment, one must delete the created cluster. The Nephio mgmt cluster will be deleted automatically when running `create-cluster.sh`, but the cluster deployed by this operator has a name in the `clusterName` field. For example, it may be `edge`, thus:

```bash
kind delete cluster -n edge
```

## Operator logic

O2IMS operator listens for ProvisioningRequest CR and once it is created it goes through different stages 

Following are the Provisioning Request Phases:


| Status   | Description |
| ---      | ---         |
| `PENDING`  | The ProvisioningRequest is waiting to be processed by the O-Cloud (IMS). |
| `PROGRESSING` | The O-Cloud (IMS) is processing the ProvisioningRequest and executing the actions to fulfill it. |
| `FULFILLED` | The ProvisioningRequest has been successfully processed and completed by the O-Cloud (IMS). |
| `FAILED` | The ProvisioningRequest could not be fully processed by the O-Cloud (IMS). |
| `DELETING` |  	The ProvisioningRequest is in the process of being deleted by the O-Cloud (IMS). |

1. `ProvisioningRequest validation`: The controller [provisioning_request_validation_controller.py](./controllers/provisioning_request_validation_controller.py) validates the provisioning requests. Currently it checks if the field `clusterName` and `clusterProvisioner`. At the moment only `capi` handled clusters are support
2. `ProvisioningRequest creation`: The controller [provisioning_request_controller.py](./controllers/provisioning_request_controller.py) takes care of creating the a package variant for Porch which can be applied to the cluster where porch is running. After applying package variant it waits for the cluster to be created and it follows the creation via querying `clusters.cluster.x-k8s.io` endpoint. Later we will add querying of packageRevisions also but at the moment their is a problem with querying packageRevisions because sometimes Porch is not able to process the request

Output of a **Successful workflow**:

<details>
<summary>The output is similar to:</summary>

```yaml
apiVersion: o2ims.provisioning.oran.org/v1alpha1
kind: ProvisioningRequest
metadata:
  annotations:
    provisioningrequests.o2ims.provisioning.oran.org/kopf-managed: "yes"
    provisioningrequests.o2ims.provisioning.oran.org/last-ha-a.A3qw: |
      {"spec":{"description":"Provisioning request for setting up a test kind cluster.","name":"test-env-Provisioning","templateName":"nephio-workload-cluster","templateParameters":{"clusterName":"edge","labels":{"nephio.org/region":"europe-paris-west","nephio.org/site-type":"edge"},"templateVersion":"v3.0.0"}}
    provisioningrequests.o2ims.provisioning.oran.org/last-handled-configuration: |
      {"spec":{"description":"Provisioning request for setting up a test kind cluster.","name":"test-env-Provisioning","templateName":"nephio-workload-cluster","templateParameters":{"clusterName":"edge","labels":{"nephio.org/region":"europe-paris-west","nephio.org/site-type":"edge"},"templateVersion":"v3.0.0"}}
  creationTimestamp: "2025-01-31T13:50:46Z"
  generation: 1
  name: provisioning-request-sample
  resourceVersion: "12122"
  uid: e8377db2-5652-4bc6-9632-8ce0836c6afd
spec:
  description: Provisioning request for setting up a test kind cluster.
  name: test-env-Provisioning
  templateName: nephio-workload-cluster
  templateParameters:
    clusterName: edge
      labels:
        nephio.org/site-type: edge
        nephio.org/region: europe-paris-west
        nephio.org/owner: nephio-o2ims
  templateVersion: v3.0.0
status:
  provisionedResourceSet:
    oCloudInfrastructureResourceIds:
    - cb92ece1-7272-4e01-9d5c-11e47b2e2473
    oCloudNodeClusterId: 09470fe4-cff6-4362-a7d6-badc77dbf059
  provisioningStatus:
    provisioningMessage: Cluster resource created
    provisioningState: fulfilled
    provisioningUpdateTime: "2025-01-31T14:52:21Z"
```

</details>

## Unit Testing

Unit tests are contained in the `tests` directory, and are intended to test pieces of the O2IMS Operator in the `controllers` directory. Currently unit tests are not comprehensive, but provide expected coverage of core utility components.

Prior to running the tests, install tox in your environment:
```bash
pip install tox
```

To run all tests in `test_utils.py` targeting a specific python version:
 ```bash
tox -e py312
```

`tox` exits non-zero if any test fails, which is what makes this a gate rather
than a report. #1175 runs the same command in CI.

## Known issues

### Porch Endpoints and Stuck Deployments

One may notice that the edge cluster is not provisioned, the provisioning request times out, or the package variant claims to be stalled (examples below). This is believed to be a bug in Porch, and so will be fixed upstream. For now a workaround has been identified.

#### O2IMS Cluster Not Present

You created the provisioning request but the cluster is not created

```bash
kind get clusters
mgmt
```

#### ProvisioningRequest Timeout

```bash
kubectl get provisioningrequest provisioning-request-sample -o yaml | grep provisioningStatus: -A 2
  provisioningStatus:
    provisioningMessage: Cluster resource creation failed reached timeout
    provisioningState: failed
```

#### PackageVariant Stalled

The package variant created by O2IMS is stalled

```bash
$ kubectl get packagevariant provisioning-request-sample -o yaml | grep conditions: -A 5
  conditions:
  - lastTransitionTime: "2025-01-29T22:25:08Z"
    message: all validation checks passed
    reason: Valid
    status: "False"
    type: Stalled
```

#### Potential Solution

One may attempt to delete the PackageVariant, ProvisioningRequest, and the Porch Server. After the Porch Server is re-deployed, re-deploy the ProvisioningRequest:

```bash
## Delete the sample provisioning resource
kubectl delete packagevariant provisioning-request-sample
kubectl delete provisioningrequest provisioning-request-sample
kubectl delete pod porch-server-7c5485b96b-tk7sr -n porch-system # Get the pod name from kubectl
# Once deleted and new Porch Server is up
kubectl create -f tests/sample_provisioning_request.yaml
```

### Deletion request O2IMS cluster

This is not supported so you have to delete the cluster manually

First delete the provisioning request:

```bash
kubectl delete -f tests/sample_provisioning_request.yaml
```

Then delete the resources, replace **edge** with your cluster name and change **mgmt** cluster repository name with your cluster management cluster repository name. 

```bash
kubectl delete packagevariants -l nephio.org/site-type=edge
kubectl delete packagevariants provisioning-request-sample
pkgList=$(kpt alpha rpkg get| grep edge | grep mgmt| awk '{print $1;}')
for pkg in $pkgList
do
 kpt alpha rpkg propose-delete $pkg -ndefault
 kpt alpha rpkg delete $pkg -ndefault
done
```

