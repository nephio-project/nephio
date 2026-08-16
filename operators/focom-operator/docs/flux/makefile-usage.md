# Makefile Targets for Flux Deployment

## Overview

The Makefile now includes targets for deploying the FOCOM operator with Flux instead of ConfigSync.

## Available Targets

### Deployment Targets

#### `make deploy` (Default - Uses ConfigSync)
Deploys the operator with ConfigSync (default behavior).

```bash
make deploy IMG=<registry>/<image>:<tag>
```

**What it does:**
1. Creates `focom-system` namespace
2. Deploys FOCOM operator
3. Copies `gitea-secret` from `default` to `config-management-system` namespace
4. Deploys ConfigSync RootSync

**Use when:**
- You want the standard ConfigSync-based deployment
- Following Nephio reference architecture
- Default choice for most users

---

#### `make deploy-with-flux` (Alternative - Uses Flux)
Deploys the operator with Flux instead of ConfigSync.

```bash
make deploy-with-flux IMG=<registry>/<image>:<tag>
```

**What it does:**
1. Creates `focom-system` namespace
2. Deploys FOCOM operator
3. Copies `gitea-secret` from `default` to `flux-system` namespace
4. Deploys Flux GitRepository and Kustomization resources

**Use when:**
- You want to use Flux instead of ConfigSync
- You need webhook support for instant sync
- You prefer CNCF graduated projects
- You want better observability

**Prerequisites:**
- Flux must be installed on the cluster
- `gitea-secret` must exist in `default` namespace
- Porch Repository CR must be ready

---

### Undeployment Targets

#### `make undeploy` (Removes ConfigSync deployment)
Removes the operator and ConfigSync resources.

```bash
make undeploy
```

**What it does:**
1. Removes ConfigSync RootSync
2. Deletes `gitea-secret` from `config-management-system` namespace
3. Removes FOCOM operator

**Use with:** `make deploy`

---

#### `make deploy-with-kpt` (Alternative - Uses kpt live apply)
Deploys the operator using kpt package management, with Flux for focom-resources sync.

```bash
make deploy-with-kpt

# With private registry
make deploy-with-kpt IMG=<your-registry>/focom-operator:<tag> IMAGE_PULL_SECRET=<secret-name>
```

**What it does:**
1. Generates kpt package bundle from kustomize
2. Deploys FOCOM operator via `kpt live apply` (with inventory tracking)
3. Copies `gitea-secret` from `default` to `flux-system` namespace
4. Deploys Flux GitRepository and Kustomization resources

**Use when:**
- You want kpt inventory tracking for clean upgrades/rollbacks
- You want a deployment method closer to how the E2E tests work
- You want `kpt live destroy` for clean teardown

**Undeploy with:** `make undeploy-with-kpt`

---

#### `make undeploy-flux` (Removes Flux deployment)
Removes the operator and FOCOM-specific Flux resources.

```bash
make undeploy-flux
```

**What it does:**
1. Removes FOCOM Flux GitRepository and Kustomization resources
2. Deletes `gitea-secret` from `flux-system` namespace
3. Waits 10 seconds for Flux to clean up CRs
4. Removes FOCOM operator

**What it does NOT do:**
- ❌ Does NOT remove Flux controllers (they keep running)
- ❌ Does NOT remove `flux-system` namespace
- ❌ Does NOT affect other Flux resources in your cluster

**Use with:** `make deploy-with-flux`

**Important:** 
- Always use `undeploy-flux` if you deployed with `deploy-with-flux`. Don't mix deployment and undeployment methods.
- Flux remains installed and can be used for other purposes after running this command.

---

## Complete Workflows

### Workflow 1: Deploy with ConfigSync (Default)

```bash
# Build and push image
make docker-build docker-push IMG=myregistry/focom-operator:latest

# Deploy with ConfigSync
make deploy IMG=myregistry/focom-operator:latest

# Verify
kubectl get rootsync focom-resources -n config-management-system
kubectl get pods -n focom-operator-system

# Later, undeploy
make undeploy
```

---

### Workflow 2: Deploy with Flux

```bash
# Build and push image
make docker-build docker-push IMG=myregistry/focom-operator:latest

# Deploy with Flux
make deploy-with-flux IMG=myregistry/focom-operator:latest

# Verify
kubectl get gitrepository focom-resources -n flux-system
kubectl get kustomization focom-resources -n flux-system
kubectl get pods -n focom-operator-system

# Later, undeploy
make undeploy-flux
```

---

### Workflow 3: Switch from ConfigSync to Flux

```bash
# Remove ConfigSync deployment
make undeploy

# Deploy with Flux
make deploy-with-flux IMG=myregistry/focom-operator:latest
```

---

### Workflow 4: Switch from Flux to ConfigSync

```bash
# Remove Flux deployment
make undeploy-flux

# Deploy with ConfigSync
make deploy IMG=myregistry/focom-operator:latest
```

---

## Comparison

| Aspect | `make deploy` | `make deploy-with-flux` | `make deploy-with-kpt` |
|--------|---------------|-------------------------|------------------------|
| **GitOps Tool** | ConfigSync | Flux | Flux |
| **Operator Deploy** | kustomize + kubectl | kustomize + kubectl | kpt live apply |
| **Namespace** | config-management-system | flux-system | flux-system |
| **Resources Created** | RootSync | GitRepository + Kustomization | GitRepository + Kustomization |
| **Secret Location** | config-management-system | flux-system | flux-system |
| **Undeploy Command** | `make undeploy` | `make undeploy-flux` | `make undeploy-with-kpt` |
| **Prerequisites** | ConfigSync installed | Flux installed | Flux + kpt installed |
| **Inventory Tracking** | No | No | Yes (kpt ResourceGroup) |
| **Webhook Support** | No | Yes (optional) | Yes (optional) |

---

## Secret Management

Both deployment methods automatically copy the `gitea-secret` from the `default` namespace to the appropriate namespace:

**ConfigSync:**
```bash
# Secret copied to: config-management-system
kubectl get secret gitea-secret -n config-management-system
```

**Flux:**
```bash
# Secret copied to: flux-system
kubectl get secret gitea-secret -n flux-system
```

**If secret doesn't exist:**
Both commands will show a warning but continue. You'll need to create the secret manually:

```bash
kubectl create secret generic gitea-secret \
  -n default \
  --from-literal=username=<git-username> \
  --from-literal=password=<git-password>
```

Then re-run the deployment command.

---

## Verification Commands

### After `make deploy` (ConfigSync)

```bash
# Check ConfigSync status
kubectl get rootsync focom-resources -n config-management-system

# Check operator
kubectl get pods -n focom-operator-system

# Check synced CRs
kubectl get oclouds,templateinfoes,focomprovisioningrequests -A

# Check logs
kubectl logs -n config-management-system -l app=reconciler-manager --tail=50
```

### After `make deploy-with-flux` (Flux)

```bash
# Check Flux resources
kubectl get gitrepository focom-resources -n flux-system
kubectl get kustomization focom-resources -n flux-system

# Check operator
kubectl get pods -n focom-operator-system

# Check synced CRs
kubectl get oclouds,templateinfoes,focomprovisioningrequests -A

# Check logs
kubectl logs -n flux-system -l app=source-controller --tail=50
kubectl logs -n flux-system -l app=kustomize-controller --tail=50
```

---

## Troubleshooting

### Secret Not Found Warning

**Problem:**
```
Warning: gitea-secret not found in default namespace
```

**Solution:**
```bash
# Create the secret
kubectl create secret generic gitea-secret \
  -n default \
  --from-literal=username=<git-username> \
  --from-literal=password=<git-password>

# Re-run deployment
make deploy-with-flux IMG=<registry>/<image>:<tag>
```

---

### Flux Not Installed

**Problem:**
```
Error: flux-system namespace not found
```

**Solution:**
```bash
# Install Flux
flux install

# Or check if it's in a different namespace
kubectl get pods -A | grep flux
```

---

### Wrong Undeploy Command

**Problem:**
Used `make undeploy` after `make deploy-with-flux` (or vice versa)

**Solution:**
```bash
# If deployed with Flux, use:
make undeploy-flux

# If deployed with ConfigSync, use:
make undeploy
```

---

### Resources Not Syncing

**ConfigSync:**
```bash
# Check RootSync status
kubectl describe rootsync focom-resources -n config-management-system

# Check logs
kubectl logs -n config-management-system -l app=reconciler-manager --tail=100
```

**Flux:**
```bash
# Check GitRepository status
kubectl describe gitrepository focom-resources -n flux-system

# Check Kustomization status
kubectl describe kustomization focom-resources -n flux-system

# Check logs
kubectl logs -n flux-system -l app=source-controller --tail=100
kubectl logs -n flux-system -l app=kustomize-controller --tail=100
```

---

## Advanced Usage

### Deploy with Custom Image

```bash
# ConfigSync
make deploy IMG=myregistry/focom-operator:v1.2.3

# Flux
make deploy-with-flux IMG=myregistry/focom-operator:v1.2.3
```

### Undeploy with Ignore Not Found

```bash
# ConfigSync
make undeploy ignore-not-found=true

# Flux
make undeploy-flux ignore-not-found=true
```

### Build, Push, and Deploy in One Go

```bash
# ConfigSync
make docker-build docker-push deploy IMG=myregistry/focom-operator:latest

# Flux
make docker-build docker-push deploy-with-flux IMG=myregistry/focom-operator:latest
```

---

## See Also

- [Flux Configuration README](README.md) - Detailed Flux configuration
- [Flux Testing Guide](TESTING_GUIDE.md) - Step-by-step testing
- [Main Deployment Guide](../../docs/DEPLOYMENT.md) - Complete deployment documentation
- [ConfigSync Configuration](../configsync/README.md) - ConfigSync details
