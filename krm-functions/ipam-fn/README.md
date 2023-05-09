# ipam-fn

The `ipam-fn` is a KRM function leveraging the `cond fn sdk`. It uses the `ipam.alloc.nephio.org/v1alpha1.IPAllocation` as a `for` KRM resource.

## details

The function allocates IPs from a IP backend based on the content of the IPAllocation. The function is implemented to align with the `cond fn sdk` but, more importantly, the function can be used in a `kpt` pipeline without relying on porch. When used in a kpt pipeline, a stub backend can be deployed for testing purposes.

## usage

```
kpt fn source <krm resource package> | go run main.go 
```

```
kpt fn eval --type mutator <krm resource package>  -i <ipam-container-image> 
```

## build

make docker-build; make docker-push