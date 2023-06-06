# Cross Package Injector

The cross package injector is a specializer that finds resources in other
package revisions, and copies them into the subject package revision. For
example, in the case of free5gc, the SMF needs to know and be configured with
information from every deployed UPF. This specializer handles that case.

A package wishing to recieve configuration inputs from other packages creates a
sentinel resource with apiVersion `config.nephio.org/v1alpha1` and Kind
`CrossPackageRequest`.

```go
type CrossPackageRequestSpec struct {
        PackageSelector PackageSelector `json:"packageSelector"`
        ResourceSelector ResourceSelector `json:"resourceSelector"`
}

type PackageSelector struct {
        metav1.LabelSelector

        Name *string `json:"name,omitempty"`
}

type ResourceSelector struct {
        metav1.LabelSelector

        Name *string `json:"name,omitempty"`
        Namespace *string `json:"namespace,omitempty"`
}
```

First, the controller will look for any PackageRevisions matching the
PackageSelector. Note that the only the namespace matching that of the subject
PackageRevision will be searched.

Next, within each selected package, the PackageRevisionResources will be
searched for any resources matching the ResourceSelector. These resources will
all be injected into the subject package, and annotated to indicate that they
were injected by this specializer. Each time this specializer reconciles, it
will ensure that injected resources are added, removed, or updated according to
changes in the selector matches.

A package condition will be created for each injected resource. The package
condition will not be set to True until the source package passes all of its
readiness gates.

Questions:
 - Does the readiness check make sense? Other criteria?
 - Should we match Draft revisions on selection, or only Published revisions? I
   am thinking Draft, and if a new Draft pops up, we update our Draft
   accordingly.
 - Do we need to rename the injected resources? I am guessing we do. Should we
   use a calculated, fixed name based on the CrossPackageRequest name, or should
   we allow a CEL expression for the name?

## PackageVariant Support
If the subject package revision was created by a PackageVariant, this
specializer will wait until the PackageVariant is Ready before performing any
mutations.
