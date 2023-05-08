# ipam-fn

The interface-fn is a KRM function leveraging the `cond sdk` using the req.nephio.org/v1alpha1.Interface as a `for` KRM resource.
It uses the clusterContext as a `watch` to determine its ready state. If no clusterContext is present in the package or if mandatory information is missing it determines its state as not ready. The `cond sdk` will delete any child resource the interface-fn owned if the state is determined as `not ready`. On top the cluster context `watch` is used to determine information such as CNI Type, masterInterface, site code which is used when creating its child resources.

The interface function has 3 `own` resources:
- ipam.alloc.nephio.org/v1alpha1.IPALlocation
- vlan.alloc.nephio.org/v1alpha1.VLANALlocation
- k8s.cni.cncf.io/v1.NetworkAttachmentDefinition

The interface fn supports various scenario's:
- default network:
    - No child resources are created since the interface is attached to the default network of the k8s cluster and all requirements will be satisfied within the k8s cluster. Its status will be determined as True, since there are no dependencies.
- No CNI type present:
    - When no CNI type is present this is seen as a loopback interface request for which only an IPAllocation (kind loopback) child resource will be requested
- CNI Type present:
    - When a CNI type is present the CNI Type of the interface request is validated against the cluster. If no match is found an error is returned
    - If the CNI type matches the cluster context a NAD, IPAllocation (kind network) and potentially a VLANAllocation is requested based on the content of the attachementType in.

Only when all child/`own` resources are satisfied the status is determined as True. The interface-fn will update the status in its Status field of the Interface KRM resource.

## usage

```
kpt fn source <krm resource package> | go run main.go 
```

```
kpt fn eval --type mutator <krm resource package>  -i <interface-container-image> 