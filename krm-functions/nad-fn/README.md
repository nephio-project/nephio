# nad-fn

Purpose of the function
-----------------------

The `NAD` KRM function is designed to be a part of the pipeline of a Nephio NF kpt package.

Its primary purpose is to find all the `IP allocations`, `VLAN allocation`, `WorkloadCluster` and `Interfaces` in the package and expand it to a set of network attachment definition resources, i.e. `NetworkAttachmentDefinition` then to be used by container network interface (CNI) plugin e.g. Multus.  

The functions and controllers are also designed to be part of the "`Condition` choreography" that is meant to synchronize the effects of multiple KRM functions and controllers acting on the same kpt package. It adds `Conditions` to the `Status` of the `Kptfile` object indicating that the `NetworkAttachmentDefinition` requests are not fulfilled yet. 

Child resources generated for an `NAD` resource
-------------------------------------------------------

For each `NAD` the function will generate one `NetworkAttachmentDefinition` CR. In order to generate the `Spec` of this child resource it uses information also from the `WorkloadCluster`, `VLANAllocation`, `IPAllocation`, `Interface` resources that the kpt package assumed to also contain. 

Let's see an example! Assuming that a kpt package contains the following three resources (among others):

    apiVersion: req.nephio.org/v1alpha1
    kind: Interface
    metadata:
        name: n3
        annotations:
            config.kubernetes.io/local-config: "true"
    spec:
        networkInstance:
            name: vpc-ran
        cniType: sriov
        attachmentType: vlan
    status:
        ...
    ---
    apiVersion: infra.nephio.io/v1alpha1
    kind: WorkloadCluster
    metadata:
        name: cluster-context-1
        annotations:
            config.kubernetes.io/local-config: "true"
    spec:
        cniConfig:
            cniType: "macvlan"
            masterInterface: "bond0"
        siteCode: edge1
        region: us-central1
    ---
    apiVersion: ipam.nephio.org/v1alpha1
    kind: IPAllocation
    metadata:
        creationTimestamp: null
        labels:
            nephio.org/interface: n3
        name: upf-us-central1-n3
    spec:
        kind: network
        selector:
            matchLabels:
            nephio.org/network-instance: sample-vpc
            nephio.org/network-name: sample-n3-net
    status:
        prefix: 13.0.0.3/24
        gateway: "13.0.0.1"
    ---
    apiVersion: vlan.alloc.nephio.org/v1alpha1
    kind: VLANAllocation
    metadata:
        annotations:
            specializer.nephio.org/owner: req.nephio.org/v1alpha1.Interface.n3
        name: n3
    spec:
        selector:
            matchLabels:
                nephio.org/site: edge1
    status:
        vlanID: 100
    ---


the following list of CRs will be added to package:

    apiVersion: "k8s.cni.cncf.io/v1"
    kind: NetworkAttachmentDefinition
    metadata:
        name: upf-us-central1-n3
    spec:
        config: '{"cniVersion":"0.3.1","vlan":100,"plugins":[{"type":"sriov","capabilities":{"ips":true},"master":"eth1","mode":"bridge","ipam":{"type":"static","addresses":[{"address":"13.0.0.3/24","gateway":"13.0.0.1"}]}}]}'
    ---

The function also adds the following conditions to the `Status` of the `Kptfile` object:

    apiVersion: kpt.dev/v1
    kind: Kptfile
    metadata:
        name: pkg-upf
        annotations:
            config.kubernetes.io/local-config: "true"
    pipeline:
      mutators:
        - image: docker.io/nephio/interface-fn:v1
        - image: docker.io/nephio/dnn-fn:v1
        - image: docker.io/nephio/nad-fn:v1
        - .....
      status:
          conditions:
            - message: update done
              reason: req.nephio.org/v1alpha1.Interface.n3
              status: "True"
              type: k8s.cni.cncf.io/v1.NetworkAttachmentDefinition.n3


Implementation details
----------------------

The function ensures that the resources of an `NetworkAttachmentDefinition` (NAD) are always in sync with the `Spec` of the input resources: the `IP allocations`, `VLAN allocation`, `WorkloadCluster` and `Interfaces`. In addition, it also checks for any previous `NetworkAttachmentDefinition` that exist part of kpt package and act upon them to produce the latest on every run. 

The NAD fn support for various scenarios:
----------------------
Please note: `github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1` defines the library for NetworkAttachmentDefinition. On the same the Config is defined as a String.  

``` 
type NetworkAttachmentDefinitionSpec struct {
	Config string `json:"config"`
} 
```
This would mean there can be a large number of acceptable combination to NAD config spec. For this reason in the scope of Nephio R1, following NAD types generation are only supported.
In practice this Config is a JSON object and it can be configured as several types such as with a VLAN, with IP Address information and with a static/dynamic IP address. Specific to Nephio.v1 the following scenarios are considered:

Case-1: Only VLAN and no IPAM
```
spec:
  config: '{
      "cniVersion": "0.3.0",
      "type": "sriov",
      "vlan": 2000
    }'
```

Case-2: IPAM with CNFtype sriov

```
  spec:
    config: '{
  "cniVersion": "0.3.1",
  "plugins": [
    {
      "type": "sriov",
      "capabilities": {
        "ips": true
      },
      "master": "eth1",
      "mode": "bridge",
      "ipam": {
        "type": "static",
        "addresses": [
          {
            "address": "14.0.0.2/24",
            "gateway": "14.0.0.1"
          }
        ]
      }
    }
  ]
}'
 ```


Case-3: IPAM with CNFtype ipvlan

```
  spec:
    config: '{
  "cniVersion": "0.3.1",
  "plugins": [
    {
      "type": "ipvlan",
      "capabilities": {
        "ips": true
      },
      "master": "eth1",
      "mode": "l2",
      "ipam": {
        "type": "static",
        "addresses": [
          {
            "address": "14.0.0.2/24",
            "gateway": "14.0.0.1"
          }
        ]
      }
    }
  ]
}'
 ```

Case-4: IPAM with CNFtype macvlan

```
  spec:
    config: '{
  "cniVersion": "0.3.1",
  "plugins": [
    {
      "type": "macvlan",
      "capabilities": {
        "ips": true
      },
      "master": "eth1",
      "mode": "bridge",
      "ipam": {
        "type": "static",
        "addresses": [
          {
            "address": "14.0.0.2/24",
            "gateway": "14.0.0.1"
          }
        ]
      }
    }, {
          "capabilities": { "mac": true },
          "type": "tuning"
       }
  ]
}'
 ```

Case-5: Other IPAM (any CNIType) and VLAN is present

```
  spec:
    config: '{
  "cniVersion": "0.3.1",
  "vlan": 2000
  "plugins": [
    {
      "type": "sriov",
      "capabilities": {
        "ips": true
      },
      "master": "eth1",
      "mode": "bridge",
      "ipam": {
        "type": "static",
        "addresses": [
          {
            "address": "14.0.0.2/24",
            "gateway": "14.0.0.1"
          }
        ]
      }
    }
  ]
}'
 ```

Above is based on reference from Nephio-Release1 discussion and free5GC helm: https://github.com/Orange-OpenSource/towards5gs-helm/tree/main/charts/free5gc/charts/

## usage

```bash
kpt fn source data | go run main.go
```

## run with the built docker image

```bash
kpt fn eval --type mutator ./data   -i <nad-container-image> 
# e.g.
kpt fn eval --type mutator ./data  -i docker.io/nephio/nad-fn:v1 
```
