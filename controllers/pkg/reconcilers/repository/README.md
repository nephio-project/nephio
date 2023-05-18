# repository controller

The repo controller is a k8s controller acting on repository.infra.nephio.org and handles the lifecycle of the repository in gitea.

For each repo CR the repo-controller handles the lifecycle of the repository in gitea. Updates are limited based on gitea's capabilities, so it is better to delete and recreate the repo or handle updates directly in gitea.

## implementation

The implementation assumes the repo-controller runs in the same cluster as the gitea server. Based on the environment variables we help the controller to connect to the gitea server.

The following environment variables are defined

- GIT_NAMESPACE: sets the namespace where the gitea server runs 
- GIT_SECRET_NAME = the secret to connect to gitea 
- GIT_SERVICE_NAME = the service to connect to gitea

example environment variables

```
- name: "GIT_NAMESPACE"
  value: "gitea"
- name: "GIT_SECRET_NAME"
  value: "git-user-secret"
- name: "GIT_SERVICE_NAME"
  value: "gitea-http"
```


## example repo CRD

```yaml
cat <<EOF | kubectl apply -f - 
    apiVersion: infra.nephio.org/v1alpha1
    kind: Repository
    metadata:
      name: mgmt
    spec:
EOF
```