# repo-controller

The repo controller is a k8s controller acting on `repository.infra.nephio.org` and handles the lifecycle of the repository in gitea.

For each repo cr the `repo-controller` creates/deletes the following resources:
- repository in gitea
- access token to the repository in gitea
- a secret in k8s representing the access token to access the gitea repo. The secret is created in the same namespace as the cr. When working with `config-sync` this must be `config-management-system` since the secret must be created in namespace: `config-management-system`

There is no update, so if a change is required it should be handled directly in gitea or delete/create the repository.

## implementation

The implementation assumes the repo-controller runs in the same cluster as the gitea server. Based on the environment variables we help the controller to connect to the gitea server.

The following environment variables are defined

GIT_NAMESPACE: sets the namespace where the gitea server runs
GIT_SECRET_NAME = the secret to connect to gitea
GIT_SERVICE_NAME = the service to connect to gitea


## example environment variables

```
- name: "GIT_NAMESPACE"
  value: "gitea"
- name: "GIT_SECRET_NAME"
  value: "git-user-secret"
- name: "GIT_SERVICE_NAME"
  value: "gitea-http"
```

## example repo CRD

```
cat <<EOF | kubectl apply -f - 
    apiVersion: infra.nephio.org/v1alpha1
    kind: Repository
    metadata:
      name: mgmt
    spec:
EOF
```