# smf-deploy-fn

## Overview

The smf-deploy-fn is a KRM function leveraging the `cond sdk` using the `workload.nephio.org/v1alpha1.SMFDeployment` as `for` KRM resource.

The smf deployment function has 3 `watch` resources:
- `req.nephio.org/v1alpha1.Interface` 
   - For every interface resource status, it will use `status.ipAllocationStatus` and `status.vlanAllocationStatus` fields to populate SMFDeployment `spec.interfaces` values.
- `req.nephio.org/v1alpha1.DataNetwork`
    - For every data network resource status, it will use `status.pools.ipAllocation` fields to populate SMFDeployment `spec.networkInstances` values.
- `req.nephio.org/v1alpha1.Capacity`
    - This resource is the source of truth for the capacity details in SMFDeployment spec.

The function generates a final resource with the same name as the `Kptfile`. It does not panic or error out if status fields are missing for any resource. It generates the `SMFDeployment` using the available data. 

## usage

```
kpt fn source <krm resource package> | go run main.go 
```

```
kpt fn eval --type mutator <krm resource package>  -i <smf-deploy-fn-container-image> 
```


## build

make smf-docker-build; make smf-docker-push