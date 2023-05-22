# token controller

The token controller is a k8s controller acting on token.infra.nephio.org and handles the lifecycle of the token in gitea. It also adds a corresponding secret in k8s within the namespace the token was applied.

The token is immutable, so if you want to change the token it has to be deleted/created

## implementation

Based on the environment variables we help the controller to connect to the gitea server.

A secret is required to connect to the git server with username and password. The default name and namespace are resp. `git-user-secret ` and POD_NAMESPACE where the token controller runs.
With the following environment variable the defaults can be changed:
- GIT_SECRET_NAME = sets the name of the secret to connect to the git server
- GIT_NAMESPACE: sets the namespace where to find the secret to connect to the git server

The URL to connect to the git server is provided through an environment variable. This is a mandatory environment variable

- GIT_URL = https://172.18.0.200:3000

example environment variables

```
- name: "GIT_URL"
  value: "https://172.18.0.200:3000"
```

## example CRD

```yaml
cat <<EOF | kubectl apply -f - 
    apiVersion: infra.nephio.org/v1alpha1
    kind: Token
    metadata:
      name: mgmt-access-token-porch
    spec:
EOF
```

```yaml
cat <<EOF | kubectl apply -f - 
    apiVersion: infra.nephio.org/v1alpha1
    kind: Token
    metadata:
      name: mgmt-access-token-configsync
      annotations:
        nephio.org/app: configsync
    spec:
EOF
```