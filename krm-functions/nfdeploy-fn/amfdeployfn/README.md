# amf-deploy-fn

## Overview

The amf-deploy-fn is a KRM function leveraging the `cond sdk` using the `workload.nephio.org/v1alpha1.AMFDeployment` as `for` KRM resource.

The amf deployment function has 3 `watch` resources:
- `req.nephio.org/v1alpha1.Interface` 
   - For every interface resource status, it will use `status.ipAllocationStatus` and `status.vlanAllocationStatus` fields to populate AMFDeployment `spec.interfaces` values.
- `req.nephio.org/v1alpha1.DataNetwork`
    - For every data network resource status, it will use `status.pools.ipAllocation` fields to populate AMFDeployment `spec.networkInstances` values.
- `req.nephio.org/v1alpha1.Capacity`
    - This resource is the source of truth for the capacity details in AMFDeployment spec.

The function generates a final resource with the same name as the `Kptfile`. It does not panic or error out if status fields are missing for any resource. It generates the `AMFDeployment` using the available data. 

## usage

```
kpt fn source <krm resource package> | go run main.go 
```

```
kpt fn eval --type mutator <krm resource package>  -i <amf-deploy-fn-container-image> 
```


## build

make amf-docker-build; make amf-docker-push