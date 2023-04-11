NF Deploy KRM function design
========================================

Purpose of the function
-----------------------

The `NF Deploy` KRM function is designed to be a part of the pipeline of a Nephio NF kpt package.

Its primary purpose is to scan the entire package for `IPAllocation`, `VLANAllocation` and `NetworkAttachmentDefinition` upstream custom resources in the package and then generate it to a set of more specific (child) KRM resource, i.e. `UPFDeployment` resources. 

The function is also designed to be part of the "`Condition`" choreography" that is meant to synchronize the effects of multiple KRM functions acting on the same kpt package. It updates the `Status` of the `UPFDeployment` in `Kptfile` indicating if the upstream requests are fulfilled and child resource is successfully generated yet.


Upstream resources for an `UPFDeployment` resource
-------------------------------------------------------
If all the `IPAllocation`, `VLANAllocation` and `NetworkAttachmentDefinition` requests are fulfilled by upstream functions and controllers, then this function generates `UPFDeployment` resource.

Let's see an example of a UPF package. Assuming that a kpt package contains the following resources for interface `n3`, `n4` and `n6`:

    apiVersion: ipam.nephio.org/v1alpha1
    kind: IPAllocation
    metadata:
      name: n3-ip-abcd
      annotations:
        krm-fn.nephio.org/owner: req.nephio.org/v1alpha1/interface/n3
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
          nephio.org/network-name: net1   
    ---
    apiVersion: ipam.nephio.org/v1alpha1
    kind: VLANAllocation
    metadata:
      name: n3-vlan-defg
      annotations:
        krm-fn.nephio.org/owner: req.nephio.org/v1alpha1/interface/n3
    spec:
      networkInstanceRef: 
        namespace: default
        name: sample-vpc
      selector:
        matchLabels:
          nephio.org/region: us-central1
          nephio.org/site: edge1
          nephio.org/network-name: net1
    ---

    apiVersion: ipam.nephio.org/v1alpha1
    kind: IPAllocation
    metadata:
      name: n4-ip-abcd
      annotations:
        krm-fn.nephio.org/owner: req.nephio.org/v1alpha1/interface/n4
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
          nephio.org/network-name: net1   
    ---
    apiVersion: ipam.nephio.org/v1alpha1
    kind: VLANAllocation
    metadata:
      name: n4-vlan-defg
      annotations:
        krm-fn.nephio.org/owner: req.nephio.org/v1alpha1/interface/n4
    spec:
      networkInstanceRef: 
        namespace: default
        name: sample-vpc
      selector:
        matchLabels:
          nephio.org/region: us-central1
          nephio.org/site: edge1
          nephio.org/network-name: net1
    ---
    apiVersion: ipam.nephio.org/v1alpha1
    kind: IPAllocation
    metadata:
      name: n6-ip-abcd
      annotations:
        krm-fn.nephio.org/owner: req.nephio.org/v1alpha1/interface/n6
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
          nephio.org/network-name: net1   
    ---
    apiVersion: ipam.nephio.org/v1alpha1
    kind: VLANAllocation
    metadata:
      name: n6-vlan-defg
      annotations:
        krm-fn.nephio.org/owner: req.nephio.org/v1alpha1/interface/n6
    spec:
      networkInstanceRef: 
        namespace: default
        name: sample-vpc
      selector:
        matchLabels:
          nephio.org/region: us-central1
          nephio.org/site: edge1
          nephio.org/network-name: net1
    ---
    apiVersion: kpt.dev/v1
    kind: Kptfile
    metadata:
        name: pkg-upf
        annotations:
            config.kubernetes.io/local-config: "true"
    pipeline:
      mutators:
      - image: nephio.org/interface-manager:v1
      - .....
      - image: nephio.org/nf-deployment:v1
    status:
        conditions:
            - type: req-nephio-org-v1alpha-upfdeployment
              status: false
            - type: req-nephio-org-v1alpha1-interface-n6
              status: true 
            - type: ipam-nephio-org-v1alpha1-ipallocation-n6-ip-abcd
              status: true 
            - type: ipam-nephio-org-v1alpha1-vlanallocation-n6-vlan-defg
              status: true 
            - type: req-nephio-org-v1alpha1-interface-n6-nad-generated
              status: true 
            - type: req-nephio-org-v1alpha1-interface-n3
              status: true 
            - type: ipam-nephio-org-v1alpha1-ipallocation-n3-ip-abcd
              status: true 
            - type: ipam-nephio-org-v1alpha1-vlanallocation-n3-vlan-defg
            - status: true  
            - type: req-nephio-org-v1alpha1-interface-n4
              status: true  
            - type: ipam-nephio-org-v1alpha1-ipallocation-n4-ip-abcd
              status: true 
            - type: ipam-nephio-org-v1alpha1-vlanallocation-n4-vlan-defg
              status: true 

Child resources after generation and Kptfile
-------------------------------------------------------

If all the upstream resource conditions are satisfied without error then function generates `UPFDeployment` function and marks the condition to `true`. 

For the UPF example, 

    apiVersion: kpt.dev/v1
    kind: Kptfile
    metadata:
        name: pkg-upf
        annotations:
            config.kubernetes.io/local-config: "true"
    pipeline:
      mutators:
      - image: nephio.org/interface-manager:v1
      - .....
      - image: nephio.org/nf-deployment:v1
    status:
        conditions:
            - type: req-nephio-org-v1alpha-nfdeployment
              status: true
            - type: req-nephio-org-v1alpha1-interface-n6
              status: true 
            - type: ipam-nephio-org-v1alpha1-ipallocation-n6-ip-abcd
              status: true
            - type: ipam-nephio-org-v1alpha1-vlanallocation-n6-vlan-defg
              status: true
            - type: req-nephio-org-v1alpha1-interface-n6-nad-generated
              status: true
            - type: req-nephio-org-v1alpha1-interface-n3
              status: true
            - type: ipam-nephio-org-v1alpha1-ipallocation-n3-ip-abcd
              status: true
            - type: ipam-nephio-org-v1alpha1-vlanallocation-n3-vlan-defg
            - type: req-nephio-org-v1alpha1-interface-n4
              status: true
            - type: ipam-nephio-org-v1alpha1-ipallocation-n4-ip-abcd
              status: true
            - type: ipam-nephio-org-v1alpha1-vlanallocation-n4-vlan-defg
              status: true

    apiVersion: nf.nephio.org/v1alpha1
    kind: UPFDeployment
    metadata: # kpt-merge: /upf-deployment
        name: upf-deployment
        annotations:
            automation.nephio.org/config-injection: "True"
            internal.kpt.dev/upstream-identifier: 'nf.nephio.org|UPFDeployment|default|upf-deployment'
        namespace: upf
    spec:
        # TODO: Add all the expected out for above sample inputs
    
Implementation details
----------------------

1. Function decides to create,update or delete the child resource based on the conditions of the upstream resources.
2. Construction of child resource. This function derives the values from the upstream resource and generates the `UPFDeployment` resource.

If there is no `UPFDeployment` exists, then the following steps should be taken:
- check if conditions for every resource's `IPAllocation`, `VLANAllocation` & `NetworkAttachment` conditions.
- add a new `UPFDeployment` to the package.
    - the `Name` TODO: How to derive the name of the resource?
    - the `Spec` is filled based on the data found in the input resources (`IPAllocation`, `VLANAllocation`, `Capacity`)

      TODO: explain in detail _how_
    - the `Status` is left empty
    - add a `Condition` to the `Kptfile`'s status indicating that the `UPFDeployment` hasn't been fulfilled yet. The name of the `Condition` is generated by the following pattern: `ipam-nephio-org-v1alpha1-upfdeployment-<name of the deployment>`
If `UPFDeployment` already exists, functions would just update the resource than creating a new one.
  
If any of the condition not satisfied, `UPFDeployment` will be deleted. If Delete annotation `fnruntime.nephio.org/delete` is present, function will delete the condition as well.