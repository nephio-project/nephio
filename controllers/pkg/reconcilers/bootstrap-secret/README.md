# bootstrap secret controller

The bootstrap controller is a k8s controller which goal is to bootstrap secrets on a newly installed cluster. This ensure e.g. that a gitops tool like `config-sync`, `argo`, `flux` or others are installed on a remote cluster and subsequent configurations tasks get handled through the gitops toolchain.

## implementation

The controller acts on a secret. It first figures out if the secret is to be installed on the remote cluster, by checking if:
- annotation key `nephio.org/app` is equal to `tobeinstalledonremotecluster`
- annotation key `nephio.org/cluster-name` is not an empty string or `mgmt`

If the controller knows the secret is to be installed on the remote cluster, it finds the credentials of the remote cluster and the type of cluster based on the signatures of the secret (right now only cluster api is implemented, but the code is able to handle other implementations).
Once the remote credentials are found and the cluster is deemed ready, the secret gets installed on the remote cluster after validating if the corresponding namespace exists. The secret get installed on the remote cluster if the corresponding namespace exists

If any of the validation fail the controller will retry installing the secret.

