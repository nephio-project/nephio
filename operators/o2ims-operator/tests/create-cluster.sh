#!/bin/bash

## A script to create a kind cluster for testing the operator
set -eo pipefail
NEPHIO_TAG=main
kpt_dir=/tmp
CATALOG_REPO=https://github.com/nephio-project/catalog.git

echo "------Deploying Nephio components from tag $NEPHIO_TAG------"

# Create a kpt package
create_kpt_package() {
  rm -rf "${kpt_dir:?}/$2"
  kpt pkg get --for-deployment "$1"/$NEPHIO_TAG $kpt_dir/"$2"
  kpt fn render "${kpt_dir:?}/$2"
  kpt live init "${kpt_dir:?}/$2"
  kpt live apply "${kpt_dir:?}/$2" --reconcile-timeout=15m --output=table
  rm -rf "${kpt_dir:?}/$2"
}

## Always delete the cluster 
kind delete cluster -n o2ims-mgmt || true
kind create cluster --config="$(dirname "$0")"/mgmt-cluster.yaml --wait 5m
kubectl cluster-info --context kind-o2ims-mgmt

# Gitea
create_kpt_package $CATALOG_REPO/distros/sandbox/gitea@origin gitea
# Porch
create_kpt_package $CATALOG_REPO/nephio/core/porch@origin porch
# MetalLB
create_kpt_package $CATALOG_REPO/distros/sandbox/metallb@origin metallb
# MetalLB Configuration
create_kpt_package $CATALOG_REPO/distros/sandbox/metallb-sandbox-config@origin metallb-sandbox-config
# Gitea IP Address
kubectl get svc -n gitea gitea
# Cluster Provisioning Cert Manager
create_kpt_package $CATALOG_REPO/distros/sandbox/cert-manager@origin cert-manager
# CAPI
create_kpt_package $CATALOG_REPO/infra/capi/cluster-capi@origin cluster-capi
# CAPI Infra
create_kpt_package $CATALOG_REPO/infra/capi/cluster-capi-infrastructure-docker@origin cluster-capi-infrastructure-docker
# CAPI Cluster Templates
create_kpt_package $CATALOG_REPO/infra/capi/cluster-capi-kind-docker-templates@origin cluster-capi-kind-docker-templates
# ConfigSync
create_kpt_package $CATALOG_REPO/nephio/core/configsync@origin configsync
# Resource Backend Operator
create_kpt_package $CATALOG_REPO/nephio/optional/resource-backend@origin resource-backend
# Nephio Core Opertaor
create_kpt_package $CATALOG_REPO/nephio/core/nephio-operator@origin nephio-operator

# Create Gitea secret 
kubectl apply -f  - <<EOF
apiVersion: v1
kind: Secret
metadata:
    name: git-user-secret
    namespace: nephio-system
type: kubernetes.io/basic-auth
stringData:
    username: nephio
    password: secret
EOF

# Gitea Repository for Management Cluster
create_kpt_package $CATALOG_REPO/distros/sandbox/repository@origin mgmt

# RootSync Object (special case)
rm -rf $kpt_dir/rootsync
kpt pkg get --for-deployment $CATALOG_REPO/nephio/optional/rootsync@origin/$NEPHIO_TAG $kpt_dir/rootsync
kpt fn eval --image "gcr.io/kpt-fn/search-replace:v0.2" $kpt_dir/rootsync/package-context.yaml -- 'by-path=data.name' "put-value=mgmt"
kpt fn render $kpt_dir/rootsync
kpt live init $kpt_dir/rootsync
kpt live apply $kpt_dir/rootsync --reconcile-timeout=15m --output=table
rm -rf $kpt_dir/rootsync

# Stock Repositories
create_kpt_package $CATALOG_REPO/nephio/optional/stock-repos@origin stock-repos

# Check Token for SA
kubectl create -f "$(dirname "$0")"/sa-test-pod.yaml
kubectl wait --for=condition=ready pod -l app=testo2ims -n porch-system --timeout=3m
rm -rf /tmp/porch-token
kubectl exec -it -n porch-system porch-sa-test -- cat /var/run/secrets/kubernetes.io/serviceaccount/token &> /tmp/porch-token

# Create CRD
kubectl create -f https://raw.githubusercontent.com/nephio-project/api/refs/heads/main/config/crd/bases/o2ims.provisioning.oran.org_provisioningrequests.yaml
export TOKEN=/tmp/porch-token ## important for development environment

# Exposing the kube proxy for development, killing previous proxy sessions if they exist
pkill kubectl
nohup kubectl proxy --port 8080 &>/dev/null &
echo "Cluster is properly configured and proxy is running at 8080"
