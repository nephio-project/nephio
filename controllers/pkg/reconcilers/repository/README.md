# repository controller

The repo controller is a k8s controller acting on repository.infra.nephio.org and handles the lifecycle of the repository in gitea.

For each repo CR the repo-controller handles the lifecycle of the repository in gitea. Updates are limited based on gitea's capabilities, so it is better to delete and recreate the repo or handle updates directly in gitea.

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