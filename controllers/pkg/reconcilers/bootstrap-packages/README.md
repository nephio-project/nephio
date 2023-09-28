# bootstrap package controller

The bootstrap controller is a k8s controller which goal is to bootstrap packages on a newly installed cluster. This ensure e.g. that a gitops tool like `config-sync` is installed on a cluster and subsequent configurations tasks get handled through the gitops toolchain.

## implementation

The controller acts on package revision resources. It first figures out if the resources of a package revision are to be installed on the remote cluster, by checking if:
- repository has the  `nephio.org/staging` key set

If the controller knows the package is to be installed on the remote cluster it finds the cluster name by checking the `nephio.org/cluster-name` annotation of the first resource in the package. (we assume the `nephio.org/cluster-name` annotation is set on all resources). Once the controller knows the cluster name it finds the credentials of the remote cluster and the type of cluster based on the signatures of the secret (right now only cluster api is implemented, but the code is able to handle other implementations).
Once the remote credentials are found and the cluster is deemed ready, the package get installed on the remote cluster.

If any of the validation fail the controller will retry installing the package. Right now the watch on package revisions is a timed based loop.

Multiple packages can be installed by the bootstrap package controller as long as they are made available in a repo with the annotation key `nephio.org/staging` and a corresponding annotation `nephio.org/cluster-name` is set on the resources of the package.