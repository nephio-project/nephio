// Copyright 2022 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bootstrappackages

import (
    "context"
    "fmt"

    //porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
    pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
    porchconfig "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
    corev1 "k8s.io/api/core/v1"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/yaml"
)

type fakeClient struct {
    objects []client.Object
    client.Client
}

var _ client.Client = &fakeClient{}

func (f *fakeClient) List(_ context.Context, obj client.ObjectList, _ ...client.ListOption) error {
    repoList := `apiVersion: config.porch.kpt.dev/v1alpha1
kind: RepositoryList
metadata:
  name: my-repo-list
items:
- apiVersion: config.porch.kpt.dev/v1alpha1
  kind: Repository
  metadata:
    annotations:
      nephio.org/staging: "true"
    name: mgmt-staging
- apiVersion: config.porch.kpt.dev/v1alpha1
  kind: Repository
  metadata:
    name: dummy-repo`

    secretList := `apiVersion: config.porch.kpt.dev
kind: List
metadata:
  name: my-secret-list
items:
- apiVersion: v1
  data:
    value: blah
  kind: Secret
  metadata:
    creationTimestamp: "2023-10-19T08:35:09Z"
    labels:
      cluster.x-k8s.io/cluster-name: wc-argocd
    name: wc-argocd-kubeconfig
    namespace: default
    ownerReferences:
    - apiVersion: controlplane.cluster.x-k8s.io/v1beta1
      blockOwnerDeletion: true
      controller: true
      kind: KubeadmControlPlane
      name: wc-argocd-w4xs6
      uid: 6e311b22-6102-420f-becc-4c633d9f9274
    resourceVersion: "345490"
    uid: df5fba18-c330-4e19-84ec-3ea39827c8f8
  type: cluster.x-k8s.io/secret
- apiVersion: v1
  data:
    password: password
    token: password
    username: user
  kind: Secret
  metadata:
    annotations:
      config.k8s.io/owning-inventory: efc7c07f6ee6a608264120c82ad317c76669a618-1697640998951302581
      internal.kpt.dev/upstream-identifier: infra.nephio.org|Token|default|example-site-name-access-token-porch
      kubectl.kubernetes.io/last-applied-configuration: |
        {"apiVersion":"infra.nephio.org/v1alpha1","kind":"Token","metadata":{"annotations":{"config.k8s.io/owning-inventory":"efc7c07f6ee6a608264120c82ad317c76669a618-1697640998951302581","internal.kpt.dev/upstream-identifier":"infra.nephio.org|Token|default|example-site-name-access-token-porch"},"name":"mgmt-access-token-porch","namespace":"default"},"spec":{}}
    creationTimestamp: "2023-10-18T14:56:40Z"
    name: mgmt-access-token-porch
    namespace: default
    ownerReferences:
    - apiVersion: infra.nephio.org/v1alpha1
      controller: true
      kind: Token
      name: mgmt-access-token-porch
      uid: dc85205e-43e6-42f6-8bb7-a5bafb398aa5
    resourceVersion: "3530"
    uid: 045c19c7-c93d-4eb4-b7c6-6fc62045a2ab
  type: kubernetes.io/basic-auth`
  

    pvList := `apiVersion: config.porch.kpt.dev
kind: PackageVariantList
metadata:
  name: my-pv-list
items:
- apiVersion: config.porch.kpt.dev
  kind: PackageVariant
  metadata:
    name: my-pv-1
  spec:
    upstream:
      repo: up
      package: up
      revision: up
    downstream:
      repo: dn-1
      package: dn-1
- apiVersion: config.porch.kpt.dev
  kind: PackageVariant
  metadata:
    name: my-pv-2
  spec:
    upstream:
      repo: up
      package: up
      revision: up
    downstream:
      repo: dn-2
      package: dn-2`

    var err error
    switch v := obj.(type) {
    case *porchconfig.RepositoryList:
        err = yaml.Unmarshal([]byte(repoList), v)
        for _, o := range v.Items {
            f.objects = append(f.objects, o.DeepCopy())
        }
    case *corev1.SecretList:
        err = yaml.Unmarshal([]byte(secretList), v)
        for _, o := range v.Items {
            f.objects = append(f.objects, o.DeepCopy())
        }
    case *pkgvarapi.PackageVariantList:
        err = yaml.Unmarshal([]byte(pvList), v)
        for _, o := range v.Items {
            f.objects = append(f.objects, o.DeepCopy())
        }
    default:
        return fmt.Errorf("unsupported type")
    }
    return err
}
