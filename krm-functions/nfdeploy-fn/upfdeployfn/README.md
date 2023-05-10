# nf-deploy-fn

## Overview

 <!--mdtogo:Short-->

`nf-deploy-fn` is a KRM function with two main purposes:
- for each Interface `pool` specified in each `DataNetwork` resource in the kpt package, `nf-deploy-fn` will create a corresponding `IPAllocation`.
- for each `IPAllocation` that was created by a previous call of `nf-deploy-fn` it will copy the result of the IP allocation to the `status` of the original `DataNetwork` resource.

 <!--mdtogo-->


 <!--mdtogo:Long-->

## Usage

`nf-deploy-fn` is primarily meant to be used declaratively, as part of the pipeline of a kpt package. It reads all of its inputs from the resources present in the package, and writes all of its outputs back into the package by creating/updating resources. It doesn't have any configuration parameters.

`nf-deploy-fn` iterates through all resources of type `DataNetwork.req.nephio.org/v1alpha1`, and creates an IPAllocation resource for each `pool` listed in the `spec` of the `DataNetwork`. It also uses information from the singleton `ClusterContext` type resource that the kpt package is expected to contain.

`nf-deploy-fn` keeps track of the resources it created by setting their `specializer.nephio.org/owner` annotation to point to the `DataNetwork` resource that it was created for.

Based on these owner annotations `nf-deploy-fn` automatically deletes (actually marks for deletion) all of the resources that it created and whose owner doesn't exist anymore. This can happen by deleting the owner `DataNetwork` resource from the package, or by deleting the corresponding `pool` form the `spec` of the owner `DataNetwork`. All in all, the role of `specializer.nephio.org/owner` annotation for Nephio KRM functions is very similarly to the role of the `ownerReference` field in the Kubernetes API server.

`nf-deploy-fn` never deletes `IPAllocation` resources directly, but it marks them for deletion if needed, by setting the `specializer.nephio.org/delete` annotation to `"true"`. It expects the IPAllocation specializer to actually delete the marked resources after properly deallocating the IP ranges.

 <!--mdtogo-->