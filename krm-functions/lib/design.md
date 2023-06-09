# krm functions library

## why

In `Nephio` we leverage packages of `KRM resources` that represent a deployment. An example of such deployment is a UPF, SMF or AMF network function, but other use cases can be envisioned. Before such package get deployed a specialization is required to obtain a set of specific parameters for this deployment instance. E.g. the specific IPs, VLANs for the deployment need to be claimed and the specific KRM resources attributes need to be actuated. In Nephio this is performed through a `condition dance/choreography` of `krm functions and/or controllers` that together reach a certain outcome. In order to reach such outcome a set of CRUD operations need to be performed on the KRM of the packages. In order to avoid reinventing the wheel a set of libraries are required that can be reused by the functions and controllers that implement the CRUD operations on the package.

## requirements

- comments MUST be retained when updating the content of a KRM resource
- MUST be able to generate new KRM resources in the package
- MUST be able to actuate a KRM resource in the package with newly obtained parameters
- MUST be able to read any attribute of a KRM resource in the package
- MUST be able to be type safe
- MUST validate new input before applying to the attributes to the new KRM object
- FUTURE: code generate the libraries

## design choices

We leverage (go)[https://go.dev] as the programming language for these libraries

We provide on interface to the function/controller that represents this api/operations on the KRM resource. A set of generic and specific methods are exposed through the go interface.

- Generic methods allow kpt fn sdk operations as well as go struct operations.
- Specific methods expose getter and setter operation on the spec/status field of the KRM resource

### generation of new KRM resources

We use a go struct to generate a new KRM resource since this allows type safety

### reading an existing resource from a package

we use the [kpt fn sdk](https://github.com/GoogleContainerTools/kpt-functions-sdk) for this

### validation of attributes within spec and status

Validation code is best located in the api spec associated with the <krm-resource>_type.go in the <krm-resource>_interface.go. As such any changes in the api would add the specific validation rules in the api spec close to where the types are retained.

## kptfile

Any operation on the kptfile is performed through a library. This library provides a set of operation including updating/deleting/adding conditions. Once kpt team endorses this approach or any alternative, this library becomes obsolete

## resource list

The resourcelist of the kpt package is consumed through a library. As such adding. deleting and updating resource to the package MUST be performed through this library. Once kpt endorses this approach or any alternative, this library becomes obsolete

## see also

- [https://go.dev](https://go.dev)
- [configuration as data](https://github.com/GoogleContainerTools/kpt/blob/main/docs/design-docs/06-config-as-data.md)
- [cad concepts](https://kpt.dev/book/02-concepts/)
- [kpt](https://kpt.dev/book/02-concepts/03-functions)
- [porch](https://kpt.dev/guides/porch-user-guide)
- [kpt fn sdk](https://github.com/GoogleContainerTools/kpt-functions-sdk)
