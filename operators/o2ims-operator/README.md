# Nephio O-RAN O2 IMS Operator

This operator implements O-RAN O2 IMS for K8s based cloud management. 

## How to start

### Development Requirements:

- Python3.10
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
cd o2ims-operator
# Create a virtual environment
virtualenv venv -p python3
source venv
# Install requirements
pip install -r requirements.txt
# Set kernel parameters (run these commands after system restart or when new VM/system is created)
sudo sysctl -w fs.inotify.max_user_watches=524288
sudo sysctl -w fs.inotify.max_user_instances=512
sudo sysctl -w kernel.keys.maxkeys=500000
sudo sysctl -w kernel.keys.maxbytes=1000000
# Get the CRD from the Nephio API repo and place it in o2ims-operator/config/crd/bases/
curl --create-dirs -O --output-dir ./config/crd/bases/ https://raw.githubusercontent.com/nephio-project/api/refs/heads/main/config/crd/bases/o2ims.provisioning.oran.org_provisioningrequests.yaml
# Run the create-cluster.sh script to create the mgmt cluster and development environment
./tests/create-cluster.sh
```

### Existing Nephio mgmt Cluster

#### Non-containerized Development Environment

```bash
kubectl exec -it -n porch-system porch-sa-test -- cat /var/run/secrets/kubernetes.io/serviceaccount/token &> /tmp/porch-token
# Get the CRD from the Nephio API repo and place it in o2ims-operator/config/crd/bases/
curl --create-dirs -O --output-dir ./config/crd/bases/ https://raw.githubusercontent.com/nephio-project/api/refs/heads/main/config/crd/bases/o2ims.provisioning.oran.org_provisioningrequests.yaml
## Create CRD
kubectl create -f ./config/crd/bases
export TOKEN=/tmp/porch-token 
# Exposing the Kube proxy for development after killing previous proxy sessions
pkill kubectl
nohup kubectl proxy --port 8080 &>/dev/null &
```

#### Containerized Development Environment

Build a Docker image: 

```bash
docker build -t o2ims:latest -f Dockerfile .
```

Push this image in your cluster, here we are using a `kind` cluster so we will push using the below command:

```bash
kind load docker-image o2ims:latest -n mgmt
```

Deploy the O2 IMS operator:

```bash
kubectl -f tests/deployment/operator.yaml
```

### To Start the Operator: 

Note that there are some constants in manager.py that can be tuned before running the operator.

```bash
## To run in debug mode use the "--debug" flag or "-v --log-format=full"
kopf run controllers/manager.py
```

Open another terminal to provision a cluster:

```bash
kubectl create -f tests/sample_provisioning_request.yaml
```

### Redeploying

To redeploy the cluster, or to recreate the development environment, one must delete the created cluster. The Nephio mgmt cluster will be deleted automatically when running `create-cluster.sh`, but the cluster deployed by this operator has a name in the `clusterName` field. For example, it may be `edge`, thus:

```bash
kind delete cluster -n edge
```

## Operator logic

O2IMS operator listens for ProvisioningRequest CR and once it is created it goes through different stages 

1. `ProvisioningRequest validation`: The controller [provisioning_request_validation_controller.py](./controllers/provisioning_request_validation_controller.py) validates the provisioning requests. Currently it checks if the field `clusterName` and `clusterProvisioner`. At the moment only `capi` handled clusters are support
2. `ProvisioningRequest creation`: The controller [provisioning_request_controller.py](./controllers/provisioning_request_controller.py) takes care of creating the a package variant for Porch which can be applied to the cluster where porch is running. After applying package variant it waits for the cluster to be created and it follows the creation via querying `clusters.cluster.x-k8s.io` endpoint. Later we will add querying of packageRevisions also but at the moment their is a problem with querying packageRevisions because sometimes Porch is not able to process the request

Output of a **Success workflow**:

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

## Unit Testing

Unit tests are contained in the `tests` directory, and are intended to test pieces of the O2IMS Operator in the `controllers` directory. Currently unit tests are not comprehensive, but provide expected coverage of core utility components. 

Prior to running the tests, install the requirements:
```bash
pip3 install -r ./tests/unit_test_requirements.txt
```

To run all tests in `test_utils.py` with abridged output:
 ```bash
pytest ./tests/test_utils.py
```

Output:
```bash
==================================================================== test session starts ====================================================================
platform linux -- Python 3.13.0, pytest-8.3.4, pluggy-1.5.0
rootdir: /home/dkosteck/Documents/nephio/operators/o2ims-operator
collected 61 items                                                                                                                                          

tests/test_utils.py .............................................................                                                                     [100%]

==================================================================== 61 passed in 0.14s =====================================================================

```

To run with verbose output (showing individual test results):
 ```bash
pytest -v ./tests/test_utils.py
```