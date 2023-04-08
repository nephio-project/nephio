# NAD function Design

Purpose of the function
-----------------------

The `NAD` KRM function is designed to be a part of the pipeline of a Nephio NF kpt package.

Its primary purpose is to find all `Network Function Deployments` custom resources and the `IP allocations` in the package and expand it to a set of network attachement definition resources, i.e. `NetworkAttachmentDefinition` then to be used by container network interface (CNI) plugin e.g. Multus. 

The function is also designed to be part of the "`Condition` choreography" that is meant to synchronize the effects of multiple KRM functions and controllers acting on the same kpt package. It adds `Conditions` to the `Status` of the `Kptfile` object indicating that the `NetworkAttachmentDefinition` requests are not fulfilled yet. 

Child resources generated for an `NAD` resource
-------------------------------------------------------

For each `NAD` the function will generate one `NetworkAttachmentDefinition` CR. In order to generate the `Spec` of this child resource it uses information also from the `ClusterContext`, `Network Function Deployments`, `IPAllocation` resource that the kpt package assumed to also contain. 

Let's see an example! Assuming that a kpt package contains the following three resources (among others):

    apiVersion: nf.nephio.org/v1alpha1
    kind: UPFDeployment
    metadata:
        creationTimestamp: "Fri Mar 31 05:26:10 PM IST 2023"
        name: upf-us-central1
    spec:
        capacity:
            downlinkThroughput: 10G
            uplinkThroughput: 1G
        n3Interfaces:
        - gatewayIPs:
            - ""
            ips:
            - ""
            name: n3
    status:
        computeuptime: "Fri Mar 31 05:26:10 PM IST 2023"
        operationuptime: "Fri Mar 31 05:26:10 PM IST 2023"
    ---
    apiVersion: infra.nephio.io/v1alpha1
    kind: ClusterContext
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
        prefix: 10.0.0.3/24
        gateway: "10.0.0.1"

the following list of CRs will be added to package:

    apiVersion: "k8s.cni.cncf.io/v1"
    kind: NetworkAttachmentDefinition
    metadata:
        name: upf-us-central1-n3
    spec:
        config: '{ "cniVersion": "0.3.1", "plugins": [ { "type": "macvlan", "capabilities": { "ips": true }, "master": "bond0", "mode": "bridge", "ipam": { "type": "static", "addresses": [ { "address": "10.0.0.3/24", "gateway": ""10.0.0.1"" } ] } }, { "capabilities": { "mac": true }, "type": "tuning" } ] }'
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
      - image: nephio.org/interface-manager:v1
      - image: nephio.org/vlan:v1
      - image: nephio.org/nad-fn:v1
      - .....
    status:
        conditions:
            - type: k8s-cni-cncf-io-v1-NetworkAttachmentDefinition-upf-us-central1-n3
              status: true


Implementation details
----------------------

The function ensures that the resources of an `NetworkAttachmentDefinition` are always in sync with the `Spec` of the input resources: the `ClusterContext`, `Network Function Deployments` and `IPAllocation`. In addition, it also checks for any previous `NetworkAttachmentDefinition` that exist part of kpt package and act upon them to produce the latest on every run. 


## dev test
`data` directory contains example of NF blueprint packages.
arguments to run locally with kpt installed. 

```bash
kpt fn source data | go run main.go
```

## run with the built docker image

```bash
kpt fn eval --type mutator ./data  -i europe-docker.pkg.dev/srlinux/eu.gcr.io/nad-fn:latest 
```
