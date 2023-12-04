# Bootstrap Secret Controller

The Bootstrap Secret Controller is a Kubernetes (k8s) controller designed to facilitate the seamless bootstrapping of secrets on a freshly installed cluster. The primary objective is to ensure the installation of essential tools such as config-sync, argo, flux, or others on a remote cluster, enabling subsequent configuration tasks to be efficiently managed through the GitOps toolchain.

## Implementation

This controller operates based on a set of predefined rules and annotations within a secret. The following criteria determine whether the secret should be installed on the remote cluster:

The annotation key `nephio.org/app` must be set to `tobeinstalledonremotecluster`.
The annotation key `nephio.org/cluster-name` should not be an empty string or equal to `mgmt`. 
The cluster name can contain multiple clusters in a comma-separated list without spaces (e.g., nephio.org/cluster-name = cluster01,cluster02).

For each cluster specified in the cluster-name, the controller follows the subsequent process. Both steps must succeed; otherwise, a reconciliation is triggered.

Per-cluster logic:

- Determine Installation Status:
If the controller identifies that the secret is meant to be installed on the remote cluster, it locates the credentials and determines the type of cluster based on the secret's signatures (currently implemented for Cluster API, but extensible for other implementations).
Once the remote credentials are obtained, and the cluster is considered ready, the secret is installed on the remote cluster. The controller validates the existence of the corresponding namespace before installation.

- Namespace:
The corresponding namespace for installation can be different from the original secret's namespace. An additional annotation, nephio.org/remote-namespace, can be used to set a custom namespace.
If any of the validation steps fail during the installation process, the controller will automatically retry, ensuring the robust deployment of secrets on the remote cluster.

## example 

This secret will be picked up by the bootstrap secret controller and will be installed on
cluster: `edge01` in namespace: `config-management-system`


```yaml
apiVersion: v1
kind: Secret
metadata:
  name: example-site-name-access-token-configsync
  namespace: default
  annotations:
    nephio.org/gitops: configsync
    nephio.org/app: tobeinstalledonremotecluster
    nephio.org/remote-namespace: config-management-system
    nephio.org/cluster-name: edge01
...
```