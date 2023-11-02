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
    value: YXBpVmVyc2lvbjogdjEKY2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiBMUzB0TFMxQ1JVZEpUaUJEUlZKVVNVWkpRMEZVUlMwdExTMHRDazFKU1VNMmFrTkRRV1JMWjBGM1NVSkJaMGxDUVVSQlRrSm5hM0ZvYTJsSE9YY3dRa0ZSYzBaQlJFRldUVkpOZDBWUldVUldVVkZFUlhkd2NtUlhTbXdLWTIwMWJHUkhWbnBOUWpSWVJGUkplazFVUVhoUFZFRTBUWHBCZDA5V2IxaEVWRTE2VFZSQmVFNXFRVFJOZWxWM1QxWnZkMFpVUlZSTlFrVkhRVEZWUlFwQmVFMUxZVE5XYVZwWVNuVmFXRkpzWTNwRFEwRlRTWGRFVVZsS1MyOWFTV2gyWTA1QlVVVkNRbEZCUkdkblJWQkJSRU5EUVZGdlEyZG5SVUpCVEZWTENtUmxOMDlxWWpZeVdrOVJSV1EyYTFOSFZuQm5PV3gzYlVOcGNIcElNRkpXVTBwb1ZFVjBOV3BLY1RseVpHazVWRlJNTVZaYVNGQXZjRVJKZEdrMVdXd0tTbEp2Vkc1S1R6TnpNMjF3V0hwb2QwcFRTWFF2V1VjM1lXRTVjakJ0UWtoMVZWWm5NVFZ6TVcxV2JIUk1UVTFrZVc1emNXUlVWa28wV2s1SVNrTXdkQXBPYm1JNFIxWkxja3d2V0VkT2FubEhWMjlMVG04MU1uWlZZV1YzYkZkVldXcHZSbTFuYXpKR2FtbFVkbVF5ZUZSRVlXMWxhV0ZtYUVOTFdUSjVZV3RNQ2xCdk0xSk5RWEl6TWxKYWVtcGlVbWswTlRVNVlrSnNaa3haYlhONFoyeEdMMFpEVjI4MllsUm5SaTlRYUZCcVFqSmpRemhRU1VjeU4zRm5aWEZEYW5BS1lVSlZaRWRQVlU5Q1owVk1SR2hEZUhWbll6aEdkMHB0VVhaWVpFZFlSR0pVVHpaNWNsRnBSVVV4ZFhKVllqY3lWRVJ6ZHprNVFWSm5Ua0ZtY1RWaFF3cHRabk5CZFdNemJXOXBSRFpLTWpka1YyVnpRMEYzUlVGQllVNUdUVVZOZDBSbldVUldVakJRUVZGSUwwSkJVVVJCWjB0clRVSkpSMEV4VldSRmQwVkNDaTkzVVVsTlFWbENRV1k0UTBGUlFYZElVVmxFVmxJd1QwSkNXVVZHVGtsTEx6RkhZbTUwTTBGYVRHSkRUSFF6TmxwbE5FeGtXVlZVVFVFd1IwTlRjVWNLVTBsaU0wUlJSVUpEZDFWQlFUUkpRa0ZSUWt0eFEyTnhhbXBYZEdaRGNsQnhjMnd2WTNwcllqWkZOVEF5VEdsNGMybGtjMFpNY2pKa1JHRXhlSFpJU1Fwd1EzbDRURlZwYVc0d1RUUnBVMnBRYlZsR2EySlZNRnB0V0hodGExWlNMMXBPWTFGdFRXVXJaR3BFUlZCc09EZEdOVWxvWVhWd2FqRmxPRFZ3U1hWWkNtRlhTRmRRZVc5blZIRjNjMUozTXpsR1RGQkVORlZHVDBNNVFXeG9jSGhhY1RkRFF6TnlVelZaUWtsNGExa3dZMWxWYWxaaWNIUllLMnBXU0hvMmVISUtWMFJhV21saWJUaGtWVEYxYWt4S05HRmtUMnBSU2xsNFNrTkhVR2xDY1cxM1ozbHZjRk5rTlRjMlIzWmhVa0Z1YWxGYVRtRXZVeXRKTkhKeFREZEhOd281V1ROTlNscDJWVGRMTW5vck1UTXdaVEY1WjFwRFZ6UkhjSFpRZUZOM05VMTZOR3hzWmpKWk1UaGlhR3RDTTIxcmJ5OUNabFJWUjNodWFsWTNZM293Q2xwNU1VWlBURmgzU0dka1JFRktkbEl3YlZrclYyVXJSMFV2YkVSUGVIcFljelpQVldNNGFYVUtMUzB0TFMxRlRrUWdRMFZTVkVsR1NVTkJWRVV0TFMwdExRbz0KICAgIHNlcnZlcjogaHR0cHM6Ly8xNzIuMTguMC42OjY0NDMKICBuYW1lOiB3Yy1hcmdvY2QKY29udGV4dHM6Ci0gY29udGV4dDoKICAgIGNsdXN0ZXI6IHdjLWFyZ29jZAogICAgdXNlcjogd2MtYXJnb2NkLWFkbWluCiAgbmFtZTogd2MtYXJnb2NkLWFkbWluQHdjLWFyZ29jZApjdXJyZW50LWNvbnRleHQ6IHdjLWFyZ29jZC1hZG1pbkB3Yy1hcmdvY2QKa2luZDogQ29uZmlnCnByZWZlcmVuY2VzOiB7fQp1c2VyczoKLSBuYW1lOiB3Yy1hcmdvY2QtYWRtaW4KICB1c2VyOgogICAgY2xpZW50LWNlcnRpZmljYXRlLWRhdGE6IExTMHRMUzFDUlVkSlRpQkRSVkpVU1VaSlEwRlVSUzB0TFMwdENrMUpTVVJGZWtORFFXWjFaMEYzU1VKQlowbEpXVmxDT1RkcGNHbGtZamgzUkZGWlNrdHZXa2xvZG1OT1FWRkZURUpSUVhkR1ZFVlVUVUpGUjBFeFZVVUtRWGhOUzJFelZtbGFXRXAxV2xoU2JHTjZRV1ZHZHpCNVRYcEZkMDFVYTNkUFJFMTNUVVJzWVVaM01IbE9SRVYzVFZSbmQwOUVUVEZOUkd4aFRVUlJlQXBHZWtGV1FtZE9Wa0pCYjFSRWJrNDFZek5TYkdKVWNIUlpXRTR3V2xoS2VrMVNhM2RHZDFsRVZsRlJSRVY0UW5Ka1YwcHNZMjAxYkdSSFZucE1WMFpyQ21KWGJIVk5TVWxDU1dwQlRrSm5hM0ZvYTJsSE9YY3dRa0ZSUlVaQlFVOURRVkU0UVUxSlNVSkRaMHREUVZGRlFYbExlWEI0YlVGVk5uTnZOV1ZEVURJS1prcEdka0l4ZUVRMWJHOHJlbFpOVUVaVU16UmthVnAxVFRoWWNsZzJOSE5YWlVaWGRrOUxPRmRuY1VKcU4yWXZlV053VFZSd1dFbFJWV1J1ZUZsUFZncHdaVUpYYUZWTE5Ua3JOekZhTkd0dlRqSjNWRVZQVkhwak9GRmhTVnB2UjNKRU9HOVNUbUZWYldKU2RsbFlXbEJJYmtSMU5GSkJLMmRyYVdFellrWmxDa2RZV1ZaUFJVMW5kSFJEVjNWTGFWZDFhR2REZVhwb1ptZEZPRUp3THl0RlVXMWpSbEZVVDNock1YTnlkRU5UVjI5d01YZzBXWEJsV25sbU4yOXZNRVlLY3l0VE0yRllVMkpMUmsxcVVqaFlaV3hCTkRCbFZXaEplWGwwVDFKTWJEVm9WR2xTTVcxQlRTdFVjRGxpVGtKRmNESmFhRzV3ZVhOM1oyZ3pUMDFSV2dvNVNWUkRWV3RJYmtOelNISlplVXBqYkZnMWFISlViRXRMTVZwNldFMTZORVZWVkdadlVVdGxWR2R2UW5aaksyRm5Sa1puUm1SUGMyVjJhSGx4TUdsWENtaEZZbmd3ZDBsRVFWRkJRbTh3WjNkU2FrRlBRbWRPVmtoUk9FSkJaamhGUWtGTlEwSmhRWGRGZDFsRVZsSXdiRUpCZDNkRFoxbEpTM2RaUWtKUlZVZ0tRWGRKZDBoM1dVUldVakJxUWtKbmQwWnZRVlV3WjNJdlZWcDFaVE5qUW10MGMwbDFNMlp3YkRkbmRERm9VazEzUkZGWlNrdHZXa2xvZG1OT1FWRkZUQXBDVVVGRVoyZEZRa0ZLWlRrM1QyVkRTbFJEVWt4VlpqRXdUekJ0YVVac0sxZEVRVGR1UlVKdU9URnhNM0k0ZERsTWQyaGlXSEJxUmtobmEzQkRPRVU0Q25CT2FqaHpVRTVIVVZFd1RFdHhNRVJpT1hZNWNtSXZWbUpqUTFsTFEzZFhUR2N5TmpoUE1WbEpWM1ZsWld0V1dXUmFLMUJOTjJVeGVFRnpVVlp4ZDFBS1FucHlOMFZMVkRGNmFtdDZabmhqTTBOaVVsUkdXakJ3Y21Gb2Ntd3pkVWxFY0dVek9FaGhiakpGUVZwa1VUQnVSSEJqTDJob2FEVTBaV3hLY1RsTlR3cFlRV0ozVUZCd1YybFZSbTByZWxkWE0wTmtOVU0wVVc1aVFsbzJhWHBQVXpVeldUUlRTbXc1YnpZNE4xRklWMnN5S3pOUFIyaDFSSFJES3pOMWRXRTNDbmRyZDJ0NFYwRlhRakppUjJOSFYyYzJhVzVxZVVsc2NpdHJNV1kyVUcxU05rSk5SVkkyZFc1MmNVUjBOMDFrVG05M2JIcEdPUzl5YTI5WWEwWTFNaThLUTFkVFoxRmhaVGNyU1hkUVRpdGlVVkJJUlVoek9IZzFNbFowY0M5amF6MEtMUzB0TFMxRlRrUWdRMFZTVkVsR1NVTkJWRVV0TFMwdExRbz0KICAgIGNsaWVudC1rZXktZGF0YTogTFMwdExTMUNSVWRKVGlCU1UwRWdVRkpKVmtGVVJTQkxSVmt0TFMwdExRcE5TVWxGY0ZGSlFrRkJTME5CVVVWQmVVdDVjSGh0UVZVMmMyODFaVU5RTW1aS1JuWkNNWGhFTld4dkszcFdUVkJHVkRNMFpHbGFkVTA0V0hKWU5qUnpDbGRsUmxkMlQwczRWMmR4UW1vM1ppOTVZM0JOVkhCWVNWRlZaRzU0V1U5V2NHVkNWMmhWU3pVNUt6Y3hXalJyYjA0eWQxUkZUMVI2WXpoUllVbGFiMGNLY2tRNGIxSk9ZVlZ0WWxKMldWaGFVRWh1UkhVMFVrRXJaMnRwWVROaVJtVkhXRmxXVDBWTlozUjBRMWQxUzJsWGRXaG5RM2w2YUdablJUaENjQzhyUlFwUmJXTkdVVlJQZUdzeGMzSjBRMU5YYjNBeGVEUlpjR1ZhZVdZM2IyOHdSbk1yVXpOaFdGTmlTMFpOYWxJNFdHVnNRVFF3WlZWb1NYbDVkRTlTVEd3MUNtaFVhVkl4YlVGTksxUndPV0pPUWtWd01scG9ibkI1YzNkbmFETlBUVkZhT1VsVVExVnJTRzVEYzBoeVdYbEtZMnhZTldoeVZHeExTekZhZWxoTmVqUUtSVlZVWm05UlMyVlVaMjlDZG1NcllXZEdSbWRHWkU5elpYWm9lWEV3YVZkb1JXSjRNSGRKUkVGUlFVSkJiMGxDUVZGRFNYQXpVVkozUW5CbFlsRnVTQXB6TkROR09VY3ZUV0pzZEdGM1dHWnJPRXgyYW5ScmFUbG9TbmRzU3pWSFIyNWhRbU5OUzFOV2FXb3pSR1l3V1U1aVluRkdWM1p4ZEhoVVoxTndRMHRhQ210WGNsZEJOMmhJYkZaSU5EaGpVWGxHYldaS2JtaEpOWFptS3k5b1VIQmhhVmcwYW5rdmNVeE1hV3RTY0hCeU1tNXpPWGRLYlRJeE0wRTRSbEExV1dVS2FFRjBWbWRYUldZMlF6WmtOemwyUkhOcFN6WmFWMFkxY2taRU1HMVRTMEZPUVdReGMzWTJOR2xFT1dkdU1GVmhXUzl4U2xkb1VYbDRUUzluVTB4NFZncEJhMnhVZUdSd2RXUlROQzlSTTJrM2RHRmtXVloxVW05cVowTlhMMDUxYVhWamJWUTViR0phTkV4UFFuSm9XVU5DTnpkdVMybEZkbEZpT1Vock9YaFNDbkZhVVhvMmVWbEtiMm92WWl0eFJsRkJTMlI2ZWpSMmRFeHZaWEpLSzA0M1NGbzFlVnBKUVhoRU0zbDFiRzU2TlZWcVFtOWpXamRDTkhsNVJsZE1OM1FLVVZsYUwxSllZVUpCYjBkQ1FWQnhhRlppYWt0VGRtdzBlVGd2UjFoVVRqQkpSV2d5ZW5ONlMzSmhlRVJvUWxReEwyWnpRVWN6TlROU1NVTXZTR05LUkFwM1JFeDVTR2xKVEhsYWVVWmlZVnBvY3pFM0t6RnJXR3hwVVZWRVNFTndjekEwZVVaRWRqRXhSVXBrTUdST1NGZzBhekZYSzJOaE5VVjVNaTgwVGpkYUNsRjRkVEEzT1U0NU1rSmtja1V5YzIxMWIzVTJWVlpKVUZKd2IzQkJZbWRtTTFkS01rRnpiak12UTNaUVlXNDJOV05tTVROVFZXdFVRVzlIUWtGTmVqVUtWa1U1TkhkemFITTRjMUpZUzFaWk5sWlFRbnBzYUZCSGFVWllPSEF4Vm1kNVZHUmhUbGhQV0ZObVR6aE1lR3RPTldOMFQzazVWVTUzZFdSeVJXTkJXUXBwTjFGSlVrMDNXbmhDUTNCT2VpOTBUR1kxVTJzMlVtZGhaMXBDVlhNMFNXdDJhekJ2ZW5KaFVFTklWbEpJVFZoc1JFWm9hbk15TWk4eVJrazBORkJ3Q21SQlltUXhiMVJUSzFVNFMyOXBlWGxRVjFsaGJWQndUM1ZrWVc1RFpVZHlWVXRxVTBZMGVFSkJiMGRCU3paT05YWkpTV3RwWTJ3MWJUQlVUSEExUkZvS2NXbFNaRzQxVkU5d2IySmxPRnBPWmxaS2RsSk1ZbFIzYWxsWk5WWlZSR055TUVsMU9IWnBaMk4zZEdOaGVreHdkVEp0WkhZeUwxYzJjazRyTjFCbmFnbzBXSGwwWm00MGNXeEhVbTVxY2pSTVNHcEpPWEpXWXl0UGNVbEdUVzFuWWpScVFXWTNaa0ZyWkhaa2FUSjVOVlZJY3pScVMwdHRWMUJ1VGxSYVJUZFJDa05xZGtsQmIybFhUa3cyV25GV1ZEWjJkM0JPUkZaalEyZFpSVUZwWmpoTFZHWTNSbFpHTlZaRGRUWk1aU3QzUm5BeVUwdEZORXBFTkROQldEQkNhemNLU0hKTWVUZFpSbUpZYldKQ1duRkxLM0JMVUVkQmQyZHBkMlEyU25OelVsUjFZamhtV2tGMWExQnVkRTlKVWtkNlRqRmxjelJ4ZFhWa1kzVm5lbVZzYWdveVltTm5aR042WWxWM1VHSTROR2hoVlhWNVZsVmtXREJHVkVWWVV6UllkV2RZYWpCd1lsQk1LMUJVTVhab1Z6VXhTRmgyY2tGVk5HaEdWWFJDSzNSdkNtZEhRVUpVYjBWRFoxbEZRWE00UTBadFpEUjJORlZZTjJaa2N6ZFZWWGc0VGt0QlZHaGhWaXRZVTB0dmVUSmFhRms1Tkcxd1lreFpZVmhoVDFSUFRFZ0tSelkyZDJWMk5tOW5OR2hCY0cxdk5WSmpUbkZGU0dObGVuUkxUbE5YUjNoaVNTdGtPWEl6ZW0wNVRGVjRZbU1yUVcxYU1sY3dZalZtTjBkemNHRjRPQXBPZDNKNkwydFdZbnB6WVc5NkwzWTFTR3hyVjBSU1dVTlBhVkpCVUhJek1tVmlPRW94Ums1dldIbEZOSE5RUjJzeFZGSmxTVzVuUFFvdExTMHRMVVZPUkNCU1UwRWdVRkpKVmtGVVJTQkxSVmt0TFMwdExRbz0K
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
    password: NTJlZGJhNzY4MTY3OGNkY2M3YmY1ODRiYjcxY2RkZDJjNzY1ZWEwMg==
    token: NTJlZGJhNzY4MTY3OGNkY2M3YmY1ODRiYjcxY2RkZDJjNzY1ZWEwMg==
    username: bmVwaGlv
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
