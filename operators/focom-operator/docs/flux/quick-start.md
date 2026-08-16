# Flux Quick Start

## TL;DR

Deploy FOCOM operator with Flux in one command:

```bash
# Option A: Use official image
make deploy-with-flux

# Option B: Use your own registry image (for local development)
make deploy-with-flux IMG=<your-registry>/focom-operator:<tag> IMAGE_PULL_SECRET=<secret-name>
```

Undeploy:

```bash
make undeploy-flux
```

## Prerequisites

- [ ] Flux installed on cluster
- [ ] `gitea-secret` exists in `default` namespace
- [ ] Porch Repository CR is ready

## Quick Test

```bash
# 1. Build and push to your registry (for local development)
make docker-build docker-push IMG=<your-registry>/focom-operator:dev

# 2. Deploy
make deploy-with-flux IMG=<your-registry>/focom-operator:dev IMAGE_PULL_SECRET=<secret-name>

# 3. Verify
kubectl get gitrepository,kustomization -n flux-system

# 3. Test end-to-end
curl -X POST http://localhost:8080/api/v1/o-clouds/draft \
  -H "Content-Type: application/json" \
  -d '{"namespace": "focom-system", "name": "test", "description": "Test"}'

curl -X POST http://localhost:8080/api/v1/o-clouds/test/draft/approve

sleep 15

kubectl get ocloud test -n focom-system

# 4. Cleanup
make undeploy-flux
```

## What Gets Deployed

1. **FOCOM Operator** → `focom-operator-system` namespace
2. **Flux GitRepository** → `flux-system` namespace (polls Git every 15s)
3. **Flux Kustomization** → `flux-system` namespace
4. **Git Secret** → Copied to `flux-system` namespace

**Note:** Flux polls Git every 15 seconds by default. For instant sync, add webhooks (see [webhook-setup.md](webhook-setup.md)). Webhooks and polling work together - webhooks provide instant sync while polling provides a reliable fallback.

## Verification

```bash
# Check Flux resources
kubectl get gitrepository focom-resources -n flux-system
kubectl get kustomization focom-resources -n flux-system

# Both should show READY=True

# Check operator
kubectl get pods -n focom-operator-system

# Should show Running
```

## Troubleshooting

**Flux not installed:**
```bash
flux install
```

**Secret not found:**
```bash
kubectl create secret generic gitea-secret \
  -n default \
  --from-literal=username=<git-username> \
  --from-literal=password=<git-password>
```

**Not syncing:**
```bash
kubectl describe gitrepository focom-resources -n flux-system
kubectl describe kustomization focom-resources -n flux-system
```

## See Also

- [MAKEFILE_USAGE.md](MAKEFILE_USAGE.md) - Complete Makefile documentation
- [TESTING_GUIDE.md](TESTING_GUIDE.md) - Detailed testing procedures
- [README.md](README.md) - Full Flux configuration documentation
