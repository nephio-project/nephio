# token controller

The token controller is a k8s controller acting on token.infra.nephio.org and handles the lifecycle of the token in gitea. It also adds a corresponding secret in k8s within the namespace the token was applied.

The token is immutable, so if you want to change the token it has to be deleted/created

## implementation

The implementation assumes the token-controller runs in the same cluster as the gitea server. Based on the environment variables we help the controller to connect to the gitea server.

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

## example CRD

```yaml
cat <<EOF | kubectl apply -f - 
    apiVersion: infra.nephio.org/v1alpha1
    kind: Token
    metadata:
      name: mgmt
    spec:
EOF
```

```yaml
cat <<EOF | kubectl apply -f - 
    apiVersion: infra.nephio.org/v1alpha1
    kind: Token
    metadata:
      name: mgmt
      namespace: config-management-system
    spec:
EOF
```