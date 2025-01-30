upsert-kustomize-res function
============================

## Overview

Insert a Kustomize KRM, or if the resource already exists, update it's `resources` list to include the paths of all non `local-config` KRM resources.
The function provides the ability to generate a kustomization.yaml, containing the KRM to be applied to a cluster by a gitops tool. 

## Usage

The KRM function is used declaratively, particularly in the mutation pipeline of KPT packages.

The internal logic first checks for the existence of a [Kustomize](https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/) KRM in the pkg. 

If found, it will:
- parse the existing `resources` list
- generate a list of the paths of all non local-config KRMs
- merge the lists, removing any possible duplciates
- add the new list to the Kustomize resources field
- upsert the Kustomize resource with the new data

Otherwise, if none exists, it will:
- create a new Kustomize KRM
- generate a list of the paths of all non local-config KRMs
- add the new list to the Kustomize resources field
- insert the new Kustomize resource to the fn resources

## Example

The function can be used inline, e.g.:

Get a sample pkg:
```bash
kpt pkg get https://github.com/nephio-project/catalog/workloads/free5gc/free5gc-cp@main
```

Invoke the function to upsert the kustomization.yaml:
```bash
kpt fn eval free5gc-cp/ --image docker.io/nephio/gen-kustomize-res:v1
```

<br>

The function can also be used in the context of a `KptFile` resource, e.g.:
```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: free5gc-cp
  annotations:
    config.kubernetes.io/local-config: "true"
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-namespace:v0.4.1
      configPath: package-context.yaml
    - image: docker.io/nephio/gen-kustomize-res:v1

```
The above example will insert or update the `kustomization.yaml` KRM in the KPT package to include the `resources` list required by kustomize based tools.


<br>

The function can also be used in the context of a `PackageVariant` resource, e.g.:
```yaml
apiVersion: config.porch.kpt.dev/v1alpha1
kind: PackageVariant
metadata:
  name: example
spec:
  upstream:
    repo: catalog
    package: blueprint
    revision: v1
  downstream:
    repo: deployments
  pipeline:
    mutators:
    - image: docker.io/nephio/gen-kustomize-res:v1
```
The above example will insert or update the `kustomization.yaml` KRM in the downstream package to include the `resources` list required by kustomize based tools.

**_NOTE:_**  
The PackageVariant approach will `prepend` the mutator function to the list any existing kpt functions defined in the KptFile pipeline of the target pkg. Ideally, this function should be appended to the pipeline, as it requires the final list of KRMs to be processed and included in the kustomization resources list.