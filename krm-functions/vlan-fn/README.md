# vlan-fn

The ipam-fn is a KRM function leveraging the `cond sdk` using the vlan.alloc.nephio.org/v1alpha1.VLANAllocation as a `for` KRM resource.
The function allocates VLANs from a VLAN backend based on the content of the VLANAllocation. A function is implemented to align with the fn sdk, but more importantly it allows us to use the fn in a `kpt` pipeline w/o relying on porch. When used in the `kpt` pipeline it uses a stub backend for testing purposes

## usage

```
kpt fn source <krm resource package> | go run main.go 
```

```
kpt fn eval --type mutator <krm resource package>  -i <ipam-container-image> 
```

## build

make docker-build; make docker-push