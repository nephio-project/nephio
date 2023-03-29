Interface management KRM function design
========================================

Purpose of the function
-----------------------

The `Interface` KRM function is designed to be a part of the pipeline of a Nephio NF kpt package.

Its primary purpose is to find all `Interface` custom resources in the package and expand it to a set of more specific (child) KRM resources, i.e. `IPAllocation` and `VLANAllocation` resources. 

The function is also designed to be part of the "`Condition` choreography" that is meant to synchronize the effects of multiple KRM functions acting on the same kpt package. It adds `Conditions` to the `Status` of the `Kptfile` object indicating that the `IPAllocation` and `VLANAllocation` requests are not fulfilled yet. The function also triggers the addition of a `NetworkAttachmentDefinition` later by a "NAD generator" KRM function by setting the corresponding `Condition` to false.


Child resources generated for an `Interface` resource
-------------------------------------------------------

For each `Interface` the function will generate one `IPAllocation` and one `VLANAllocation` CR. In order to generate the `Spec` of those child resources it uses information also from the `ClusterContext` resource that the kpt package assumed to also contain.

Let's see an example! Assuming that a kpt package contains the following two resources (among others):

    apiVersion: req.nephio.org/v1alpha1
    kind: Interface
    metadata:
        name: n6
        annotations:
            config.kubernetes.io/local-config: "true"
    spec:
        networkInstance:
            name: vpc-internet
        cniType: sriov
        attachementType: vlan
    ---
    apiVersion: infra.nephio.org/v1alpha1
    kind: ClusterContext
    metadata: 
        name: cluster-context
        annotations:
            config.kubernetes.io/local-config: "true"
    spec:
        cniConfig:
            cniType: macvlan
            masterInterface: eth1
        siteCode: edge1
        region: us-central1

the following list of CRs will be added to package:

    apiVersion: ipam.nephio.org/v1alpha1
    kind: IPAllocation
    metadata:
      name: n6-ip-abcd
    spec:
      kind: network
      prefixLength: 32
      networkInstanceRef: 
        namespace: default
        name: sample-vpc
      selector:
        matchLabels:
          nephio.org/region: us-central1
          nephio.org/site: edge1
          nephio.org/network-name: net1   # TODO: Does this come from ClusterContext?
    ---
    apiVersion: ipam.nephio.org/v1alpha1
    kind: VLANAllocation
    metadata:
      name: n6-vlan-defg
    spec:
      networkInstanceRef: 
        namespace: default
        name: sample-vpc
      selector:
        matchLabels:
          nephio.org/region: us-central1
          nephio.org/site: edge1
          nephio.org/network-name: net1


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
      - .....
    status:
        conditions:
            - type: req-nephio-org-v1alpha1-interface-n6
              status: true
            - type: ipam-nephio-org-v1alpha1-ipallocation-n6-ip-abcd
              status: false
            - type: ipam-nephio-org-v1alpha1-vlanallocation-n6-vlan-defg
              status: false
            - type: req-nephio-org-v1alpha1-interface-n6-nad-generated
              status: false


Implementation details
----------------------

The function ensures that the child resources of an `Interface` are always in sync with the `Spec` of the input resources: the `Interface` and the `ClusterContext`. The function always sets the `ownerReference` field of the child resources it creates to point to the originating `Interface` resource. That way it can easily keep track of previously created resources and update or delete them if the input resources are changed.

The process that is described below is executed for each `Interface` objects in the kpt package. In the followings I will use the `IPAllocation` as an example child resource, but the same process will work for the `VLANAllocation`, as well.


### Creating a child resource, if it doesn't exist

If there is no `IPAllocation` exists with an `ownerReference` to the `Interface`, then the following steps should be taken:
- check if `ClusterContext` exists in the kpt package, and exit if it isn't
- add a new `IPAllocation` to the package. 
    - the `name` of the object is generated from the name of the `Interface`. A short hash is appended to the name in order to avoid collisions during later updates/deletes.
    - the `ownerReference` field is set to point to the `Interface`
    - the `Spec` is filled based on the data found in the input resources (`Interface` and `ClusterContext`)

        TODO: explain in detail _how_

    - the `Status` is left empty
- add a `Condition` to the `Kptfile`'s status inidicating that the `IPAllocation` hasn't been fulfilled yet. The name of the `Condition` is generated by the following pattern: `ipam-nephio-org-v1alpha1-ipallocation-<name of the IPAllocation>`


### Deleting a child resource, if it is no longer needed

If there is an existing `IPAllocation` with an `ownerReference` to the `Interface`, but it is deemed to be superflous (e.g. `ClusterContext` was deleted), then the following steps should be taken:
- set the `deletionTimestamp` field of the `IpAllocation` to the current time
- add a `Condition` to the `Kptfile`'s status inidicating that the `IPAllocation` is not in sync (the same condition that was referred to above).

We rely on the `IPAllocation controller` to detect that the already allocated IP range should be deallocated and to actually delete the `IPAllocation` from the package after that happened succesfully.


### Garbage collection

In order to handle the deletion of `Interface` resources properly. The function should also look for `IPAllocation` resources that have an `ownerReference` to a non-existent `Interface`. The deletion process described above should be applied also to these orphaned `IPAllocation`s.


### Updating an existing child resource

If there is an existing `IPAllocation` with an `ownerReference` to the `Interface`, then the following steps should be taken:
- calculate how the `IPAllocation` resource should look like based on the rules described in the "Creating a child resource" paragraph. It will be referred to as the _desired_ _`IPAllocation`_.
- find the the `IPAllocation` resource that is actually in the package. It will be referred to as the _observed_ _`IPAllocation`_.
- compare the `Spec` part of the desired and observed `IPAllocations`
- if there is no difference: exit and do nothing (for this `Interface`)
- if there is a difference, then delete the old `IPAllocation` by the process described above and create the new one with an empty `Status` (also described above). This is needed for the proper reallocation of the IP addresses in the IPAM system by the `IPAllocation controller`.
- set the `ipam-nephio-org-v1alpha1-ipallocation-<name of the IPAllocation>` `Condition` to false for both the "to-be deleted" and the brand new `IPAllocation`s