# dnn-fn

## Overview

<!--mdtogo:Short-->

`dnn-fn` is a KRM function with two main purposes:
- for each IP `pool` specified in each `DataNetwork` resource in the kpt package, `dnn-fn` will create a corresponding `IPClaim`. 
- for each `IPClaim` that was created by a previous call of `dnn-fn` it will copy the result of the IP claim to the `status` of the original `DataNetwork` resource.

<!--mdtogo-->


<!--mdtogo:Long-->

## More details

`dnn-fn` is primarily meant to be used declaratively, as part of the pipeline of a kpt package. It reads all of its inputs from the resources present in the package, and writes all of its outputs back into the package by creating/updating resources. It doesn't have any configuration parameters.

`dnn-fn` iterates through all resources of type `DataNetwork.req.nephio.org/v1alpha1`, and creates an IPClaim resource for each `pool` listed in the `spec` of the `DataNetwork`. It also uses information from the singleton `WorkloadCluster` type resource that the kpt package is expected to contain.

`dnn-fn` keeps track of the resources it created by setting their `specializer.nephio.org/owner` annotation to point to the `DataNetwork` resource that it was created for. 

Based on these owner annotations `dnn-fn` automatically deletes (actually marks for deletion) all of the resources that it created and whose owner doesn't exist anymore. This can happen by deleting the owner `DataNetwork` resource from the package, or by deleting the corresponding `pool` form the `spec` of the owner `DataNetwork`. All in all, the role of `specializer.nephio.org/owner` annotation for Nephio KRM functions is very similarly to the role of the `ownerReference` field in the Kubernetes API server.

`dnn-fn` never deletes `IPClaim` resources directly, but it marks them for deletion if needed, by setting the `specializer.nephio.org/delete` annotation to `"true"`. It expects the IPClaim specializer to actually delete the marked resources after properly releasing the IP ranges.

<!--mdtogo-->

## Usage

```
kpt fn source <krm resource package> | go run main.go 
```

```
kpt fn eval --type mutator <krm resource package>  -i <dnn-fn-container-image> 
```
