kubectl patch deployment argocd-repo-server -n argocd --type json -p='[
   {
    "op": "add",
    "path": "/spec/template/spec/containers/-",
    "value": {
      "name": "kpt-repo-argo-cmp",
      "image": "docker.io/nephio/kpt-repo-argo-cmp:latest",
      "command": ["/var/run/argocd/argocd-cmp-server"],
      "securityContext": {
          "runAsNonRoot": true,
          "runAsUser": 999
      },
      "volumeMounts": [
        {
           "name": "var-files",
           "mountPath": "/var/run/argocd"
        },
        {
           "name": "cmp-tmp",
           "mountPath": "/tmp"
        },
        {
           "name": "plugins",
           "mountPath": "/home/argocd/cmp-server/plugins"
        }
      ]
    }
  },
  {
    "op": "add",
    "path": "/spec/template/spec/containers/-",
    "value": {
      "name": "kpt-render-argo-cmp",
      "image": "docker.io/nephio/kpt-render-argo-cmp:latest",
      "command": ["/var/run/argocd/argocd-cmp-server"],
      "securityContext": {
          "runAsNonRoot": true,
          "runAsUser": 999
      },
      "volumeMounts": [
        {
           "name": "var-files",
           "mountPath": "/var/run/argocd"
        },
        {
           "name": "cmp-tmp",
           "mountPath": "/tmp"
        },
        {
           "name": "plugins",
           "mountPath": "/home/argocd/cmp-server/plugins"
        }
      ]
    }
  },
  {
    "op": "add",
    "path": "/spec/template/spec/volumes/-",
    "value": {
       "name": "cmp-tmp",
       "emptyDir": {}
    }
  },
  {
    "op": "add",
    "path": "/spec/template/spec/volumes/-",
    "value": {
       "name": "var-run-argocd",
       "emptyDir": {}
    }
  }
]'

exit_code=$?

if [ $exit_code -eq 0 ]; then
    echo "patched"
else
    echo "failed"
fi

