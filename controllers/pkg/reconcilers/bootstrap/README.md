# bootstrap controller

The bootstrap controller is a k8s controller which goal is to bootstrap packages on a newly installed cluster. This ensure e.g. that a gitops tool like `config-sync` is installed on a cluster and subsequent configurations tasks get handled through the gitops toolchain.

## implementation

The controller acts on a secret. This ensures that the bootstrap controller is agnostic to the cluster implementation. It figures out the type of cluster based on the signatures of the secret. (right now only cluster api is implemented, but the code is able to handle other implementations)

The are 2 scenario's wrt secret:

- A secret representing the kubeconfig of a cluster. This secret (representing the kubeconfig of the remote cluster) is used to access the cluster. The bootstrap controller install the packages in the staging repo belonging to the cluster when the cluster is ready.
    - clustername is derive from the annotation: `nephio.org/site`
- A secret belonging to the namespace `config-management-system` will be installed on the corresponding cluster. This ensures the token to access the repo is installed on the cluster.
    - clustername is derive from the annotation: `nephio.org/site`
    - when the `nephio.org/site` is equal the `mgmt` nothing is done.