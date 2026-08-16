# FOCOM Operator Deployment Guide

## Overview

The FOCOM operator provides a REST API for managing O-RAN infrastructure resources with GitOps-based storage. This guide covers deployment, configuration, and verification.

**Architecture:** REST API → Porch → Git Repository → ConfigSync → Kubernetes CRs

The ConfigSync RootSync for focom-resources is integrated into the deployment process and is automatically deployed with `make deploy` and removed with `make undeploy`.

## Prerequisites

### Required Components

1. **Kubernetes Cluster** (v1.24+)
2. **Nephio Porch** installed and running
3. **ConfigSync** installed (part of Anthos Config Management or Nephio)
4. **⚠️ Git Repository** for storing FOCOM resources (CRITICAL - must exist before deployment)
5. **⚠️ Porch Repository CR** configured and ready (CRITICAL - must be applied before deployment)
6. **Git Credentials** stored in `gitea-secret` in `default` namespace
7. **Docker Registry Access** (for private images from GitHub Container Registry)

### Verify Prerequisites

```bash
# Check Porch
kubectl get crd | grep porch
kubectl api-resources | grep porch
kubectl get pods -n porch-system

# Check ConfigSync
kubectl get crd | grep configsync
kubectl get pods -n config-management-system

# Check Git secret
kubectl get secret gitea-secret -n default

# Check Docker registry authentication (for private images)
kubectl get secret ghcr-secret -n focom-operator-system 2>/dev/null || echo "Docker auth not configured"
```

## Pre-Deployment Checklist

**⚠️ Complete these steps BEFORE running `make deploy`:**

- [ ] Git repository created (e.g., `focom-resources.git`)
- [ ] `gitea-secret` created in `default` namespace
- [ ] Porch Repository CR applied (`focom-porch-repository.yaml`)
- [ ] Repository shows `READY=True` status
- [ ] ConfigSync is installed and running

**Verify prerequisites:**
```bash
# Check Git secret exists
kubectl get secret gitea-secret -n default

# Check Porch Repository is ready
kubectl get repository focom-resources -n default
# Must show READY=True

# Check ConfigSync is running
kubectl get pods -n config-management-system
```

## Deployment Methods

The FOCOM operator supports multiple deployment methods:

1. **Standard Deployment** - Using `make deploy` (recommended for development)
2. **kpt-based Deployment** - Using kpt CLI for GitOps workflows
3. **Direct kubectl** - Using the generated bundle file

Choose the method that best fits your workflow.

---

## Method 1: Standard Deployment (Recommended)

### Quick Start

**After completing the pre-deployment checklist:**

**Option A: Using pre-built private registry image (recommended):**
```bash
# Set up Docker registry authentication
make docker-auth REGISTRY_USER=your-username REGISTRY_PASSWORD=your-token

# Deploy with pre-built image
make deploy IMG=ghcr.io/your-org/focom-operator:latest IMAGE_PULL_SECRET=ghcr-secret
```

**Option B: Build and push your own image:**
```bash
# Build and push image
make docker-build docker-push IMG=<registry>/<image>:<tag>

# Deploy everything
make deploy IMG=<registry>/<image>:<tag>
```

### Detailed Steps

#### `make deploy`

1. **Create focom-system namespace** (if it doesn't exist)
2. **Deploy FOCOM operator**
   - Controller manager deployment
   - RBAC (ClusterRole, ClusterRoleBinding, ServiceAccount)
   - CRDs (OCloud, TemplateInfo, FocomProvisioningRequest)
   - Services (metrics, webhook)
3. **Copy gitea-secret** from `default` to `config-management-system` namespace
4. **Deploy ConfigSync RootSync** using kustomize
5. **Verify deployment** (optional health checks)

#### `make undeploy`

1. **Remove ConfigSync RootSync** (this also deletes managed CRs)
2. **Delete gitea-secret** from `config-management-system` namespace
3. **Undeploy FOCOM operator** (controller manager, RBAC, CRDs, services)
4. **Clean up namespaces** (optional)

## Files Involved

### Kustomize Configuration
- `config/configsync/kustomization.yaml` - Kustomize configuration for ConfigSync resources
- `config/configsync/focom-resources-rootsync.yaml` - RootSync resource definition

### Makefile Targets
- `deploy` - Deploys operator and ConfigSync
- `undeploy` - Removes operator and ConfigSync

### Documentation
- `config/configsync/README.md` - User-facing documentation
- `config/configsync/DEPLOYMENT.md` - This file (deployment integration details)

## Prerequisites

### Required Secret

The `gitea-secret` must exist in the `default` namespace before running `make deploy`. This secret contains Git credentials for accessing the focom-resources repository.

**Create the secret using the helper script:**
```bash
cd focom-operator/config/samples
./create-gitea-secret.sh
```

**Or create manually:**
```bash
kubectl create secret generic gitea-secret \
  -n default \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=<git-username> \
  --from-literal=password=<git-password> \
  --from-literal=bearerToken=<git-access-token>
```

**Secret structure:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gitea-secret
  namespace: default
type: kubernetes.io/basic-auth
data:
  username: <base64-encoded-username>
  bearerToken: <base64-encoded-token>
  password: <base64-encoded-password>
```

**Template available at:** `config/samples/gitea-secret.yaml`

If the secret doesn't exist, the deployment will show a warning but continue. ConfigSync will not work without valid Git credentials.

## Verification

### After Deployment

Check that ConfigSync is working:

```bash
# Check RootSync status
kubectl get rootsync focom-resources -n config-management-system

# Expected output:
# NAME              RENDERINGCOMMIT   SOURCECOMMIT      SYNCCOMMIT        SYNCERRORCOUNT
# focom-resources   <commit-hash>     <commit-hash>     <commit-hash>     0

# Check for synced CRs
kubectl get oclouds,templateinfoes,focomprovisioningrequests -A \
  -l app.kubernetes.io/managed-by=configmanagement.gke.io

# Check ConfigSync logs
kubectl logs -n config-management-system -l app=reconciler-manager --tail=50
```

### Troubleshooting

**RootSync not syncing:**
```bash
# Check RootSync status for errors
kubectl describe rootsync focom-resources -n config-management-system

# Check if secret exists
kubectl get secret gitea-secret -n config-management-system

# Check ConfigSync logs
kubectl logs -n config-management-system -l app=reconciler-manager --tail=100
```

**Secret not found warning during deploy:**
```bash
# Verify secret exists in default namespace
kubectl get secret gitea-secret -n default

# If missing, create it or copy from another namespace
kubectl get secret gitea-secret -n <source-namespace> -o yaml | \
  sed 's/namespace: <source-namespace>/namespace: default/' | \
  kubectl apply -f -
```

## Manual Operations

### Deploy ConfigSync Only

If you need to deploy ConfigSync separately (without redeploying the operator):

```bash
# Copy secret
kubectl get secret gitea-secret -n default -o yaml | \
  sed 's/namespace: default/namespace: config-management-system/' | \
  kubectl apply -f -

# Deploy RootSync
kubectl apply -k config/configsync
```

### Remove ConfigSync Only

If you need to remove ConfigSync without undeploying the operator:

```bash
kubectl delete -k config/configsync
kubectl delete secret gitea-secret -n config-management-system
```

## Integration with CI/CD

### Automated Deployment

```bash
# Build and push image
make docker-build docker-push IMG=<registry>/<image>:<tag>

# Deploy to cluster
make deploy IMG=<registry>/<image>:<tag>
```

### Automated Undeployment

```bash
make undeploy
```

## Advantages of Integration

1. **Single Command Deployment** - One `make deploy` command deploys everything
2. **Consistent State** - Operator and ConfigSync are always deployed together
3. **Automatic Cleanup** - `make undeploy` removes all resources
4. **No Manual Steps** - Secret copying is automated
5. **Error Handling** - Graceful handling of missing secrets

## Configuration

### Environment Variables

**Required:**
```bash
NBI_STORAGE_BACKEND=porch          # Use Porch storage
NBI_STAGE=2                        # Stage 2 (Porch)
```

**Optional (with defaults):**
```bash
FOCOM_NAMESPACE=focom-system       # Default namespace for FOCOM resources (default: focom-system)
KUBERNETES_BASE_URL=https://kubernetes.default.svc  # K8s API URL (auto-detected)
TOKEN=/var/run/secrets/kubernetes.io/serviceaccount/token  # Auth token (auto-detected)
PORCH_NAMESPACE=default            # Namespace for PackageRevisions (default: default)
PORCH_REPOSITORY=focom-resources   # Porch repository name (default: focom-resources)
```

**Note:** The `FOCOM_NAMESPACE` environment variable sets the default namespace for all FOCOM resources (OCloud, TemplateInfo, FocomProvisioningRequest). This eliminates the need to specify namespace in API request bodies.

### Git Repository Setup

**⚠️ CRITICAL: This must be completed BEFORE deploying the FOCOM operator!**

#### Step 1: Create Git Repository

Create a Git repository in your Git server (Gitea, GitHub, GitLab):
- Repository name: `focom-resources`
- Initialize: Empty or with README
- Branch: `main`
- Example URL: `http://172.18.0.200:3000/nephio/focom-resources.git`

#### Step 2: Create Porch Repository CR

**Option A: Using the sample file (recommended):**
```bash
# Edit the sample to match your Git repository
vim config/samples/focom-porch-repository.yaml

# Apply to cluster
kubectl apply -f config/samples/focom-porch-repository.yaml
```

**Option B: Create directly:**
```bash
kubectl apply -f - <<EOF
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: focom-resources
  namespace: default
spec:
  type: git
  content: Package
  deployment: true
  git:
    repo: http://172.18.0.200:3000/nephio/focom-resources.git
    branch: main
    directory: /
    secretRef:
      name: gitea-secret
EOF
```

#### Step 3: Verify Repository is Ready

```bash
kubectl get repository focom-resources -n default
# Should show READY=True

# Check detailed status
kubectl describe repository focom-resources -n default
```

**⚠️ DO NOT proceed with operator deployment until the repository shows READY=True!**

## Docker Registry Authentication

### Overview

The FOCOM operator supports deployment from both public and private Docker registries. When using GitHub Container Registry (`ghcr.io`) for private images, you need to set up authentication.

### GitHub Container Registry Setup

#### Step 1: Create Personal Access Token (PAT)

1. Go to GitHub → Settings → Developer settings → Personal access tokens
2. Generate new token (classic) with the following scopes:
   - `read:packages` - Required for pulling images
   - `write:packages` - Required for pushing images (CI/CD only)
3. Copy the token (you won't see it again!)

#### Step 2: Create Kubernetes Authentication Secret

**Using the Makefile helper (recommended):**

```bash
# Create authentication secret for GitHub Container Registry
make docker-auth REGISTRY_USER=your-username REGISTRY_PASSWORD=your-token

# Optional: specify email (defaults to username@users.noreply.github.com)
make docker-auth REGISTRY_USER=johndoe REGISTRY_PASSWORD=ghp_xxxx
```

**What this does:**
- Creates `focom-operator-system` namespace if it doesn't exist
- Creates `ghcr-secret` docker-registry secret in `focom-operator-system` namespace
- Configures authentication for `ghcr.io` registry
- Provides helpful next steps and troubleshooting info

**Manual creation (if needed):**

```bash
# Create namespace
kubectl create namespace focom-operator-system --dry-run=client -o yaml | kubectl apply -f -

# Create docker registry secret
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=your-username \
  --docker-password=your-pat-token \
  --docker-email=your-email@example.com \
  --namespace=focom-operator-system
```

#### Step 3: Verify Authentication Secret

```bash
# Check if secret exists
kubectl get secret ghcr-secret -n focom-operator-system

# View secret details (base64 encoded)
kubectl describe secret ghcr-secret -n focom-operator-system

# Test image pull (optional)
kubectl run test-pull --image=ghcr.io/your-org/focom/focom-operator:latest --rm -it --restart=Never
```

### Registry Configuration

**Current Configuration:**
- **Registry**: `ghcr.io` (GitHub Container Registry)
- **Image Name**: `ghcr.io/{org}/focom/focom-operator`
- **Authentication**: Kubernetes `docker-registry` secret
- **Namespace**: `focom-operator-system`
- **Secret Name**: `ghcr-secret`

**Available Images:**
- `ghcr.io/{org}/focom/focom-operator:latest` - Latest main branch build
- `ghcr.io/{org}/focom/focom-operator:main` - Main branch builds
- `ghcr.io/{org}/focom/focom-operator:develop` - Develop branch builds
- `ghcr.io/{org}/focom/focom-operator:sha-abc123...` - Commit-specific builds

### Deployment with Private Registry

**Using pre-built images from GitHub Container Registry:**

```bash
# Set up authentication first
make docker-auth REGISTRY_USER=your-username REGISTRY_PASSWORD=your-token

# Deploy with private registry image
make deploy IMG=ghcr.io/your-org/focom-operator:latest IMAGE_PULL_SECRET=ghcr-secret
```

**The deploy target automatically:**
- Detects if you're using a `ghcr.io` image
- Checks if authentication secret exists
- Warns if authentication is missing
- Provides helpful guidance for setup

### Troubleshooting Docker Authentication

**Get help with authentication setup:**
```bash
make docker-auth-help
```

**Common issues:**

1. **ImagePullBackOff errors:**
   ```bash
   # Check if secret exists
   kubectl get secret ghcr-secret -n focom-operator-system
   
   # Check pod events
   kubectl describe pod -n focom-operator-system -l control-plane=controller-manager
   
   # Verify image name and tag
   kubectl get deployment focom-operator-controller-manager -n focom-operator-system -o yaml | grep image:
   ```

2. **Invalid credentials:**
   ```bash
   # Delete and recreate secret
   kubectl delete secret ghcr-secret -n focom-operator-system
   make docker-auth REGISTRY_USER=your-username REGISTRY_PASSWORD=new-token
   
   # Restart deployment to pick up new secret
   kubectl rollout restart deployment/focom-operator-controller-manager -n focom-operator-system
   ```

3. **Token permissions:**
   - Ensure PAT has `read:packages` scope
   - Verify token hasn't expired
   - Check if repository/organization has package access restrictions

**Remove authentication secret:**
```bash
kubectl delete secret ghcr-secret -n focom-operator-system
```

### Using Public Registry

If you prefer to use a public registry (Docker Hub, etc.), you can skip the authentication setup:

```bash
# Build and push to public registry
make docker-build docker-push IMG=your-dockerhub-user/focom-operator:tag

# Deploy without authentication
make deploy IMG=your-dockerhub-user/focom-operator:tag
```

**Note:** You may see a warning about missing `ghcr-secret` when using public images, but this is harmless and the deployment will work correctly. See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for details.

## Post-Deployment Verification

### Check Operator Status

```bash
# Check pods
kubectl get pods -n focom-operator-system

# Check logs
kubectl logs -n focom-operator-system -l control-plane=controller-manager --tail=50

# Test API
curl http://localhost:8080/health/live
curl http://localhost:8080/api/info
```

### Check ConfigSync Status

```bash
# Check RootSync
kubectl get rootsync focom-resources -n config-management-system

# Expected output:
# NAME              RENDERINGCOMMIT   SOURCECOMMIT      SYNCCOMMIT        SYNCERRORCOUNT
# focom-resources   <commit-hash>     <commit-hash>     <commit-hash>     0

# Check for errors
kubectl describe rootsync focom-resources -n config-management-system

# Check logs
kubectl logs -n config-management-system -l app=reconciler-manager --tail=50
```

### Test End-to-End Flow

```bash
# 1. Create draft
curl -X POST http://localhost:8080/api/v1/o-clouds/draft \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "focom-system",
    "name": "test-ocloud",
    "description": "Test deployment"
  }'

# 2. Validate draft
curl -X POST http://localhost:8080/api/v1/o-clouds/test-ocloud/draft/validate

# 3. Approve draft
curl -X POST http://localhost:8080/api/v1/o-clouds/test-ocloud/draft/approve

# 4. Wait for ConfigSync (up to 15 seconds)
sleep 15

# 5. Verify CR was created
kubectl get ocloud test-ocloud -n focom-system

# 6. Check ConfigSync annotations
kubectl get ocloud test-ocloud -n focom-system -o yaml | grep configmanagement
```

---

## Method 2: kpt-based Deployment

### Overview

The kpt-based deployment method uses the kpt CLI to manage the operator lifecycle. This approach provides:
- **State tracking** - kpt tracks what resources it deployed
- **Automatic pruning** - Removes resources deleted from the package
- **Diff preview** - See changes before applying
- **GitOps-friendly** - Integrates well with GitOps workflows

### Prerequisites

**Install kpt CLI:**
```bash
# Linux
curl -L https://github.com/kptdev/kpt/releases/download/v1.0.0-beta.49/kpt_linux_amd64 -o kpt
chmod +x kpt
sudo mv kpt /usr/local/bin/

# macOS
brew install kpt

# Verify installation
kpt version
```

### Step 1: Generate kpt Package

```bash
# Generate the kpt package bundle
make kpt-package IMG=<registry>/<image>:<tag>

# This creates: kpt-package/focom-operator-bundle.yaml
```

### Step 2: Initialize kpt Package

**⚠️ Important:** The namespace `focom-operator-system` will be created by the package resources, so it doesn't need to exist beforehand.

```bash
# Navigate to package directory
cd kpt-package/

# Initialize for kpt live apply
kpt live init --namespace focom-operator-system --inventory-id focom-operator

# This adds inventory tracking to the Kptfile
```

**What this does:**
- Adds `config.k8s.io/inventory` annotations to Kptfile
- Creates a ResourceGroup template for tracking deployed resources
- Does NOT create the namespace (the bundle includes namespace definition)

### Step 3: Preview Changes

```bash
# See what will be applied (dry-run)
kpt live apply --dry-run

# View diff against current cluster state
kpt live apply --dry-run --output=table
```

### Step 4: Apply Package

```bash
# Apply all resources
kpt live apply

# Expected output:
# namespace/focom-operator-system created
# customresourcedefinition.apiextensions.k8s.io/o-clouds.focom.nephio.org created
# customresourcedefinition.apiextensions.k8s.io/templateinfoes.provisioning.oran.org created
# ...
# 15 resource(s) applied. 15 created, 0 unchanged, 0 configured, 0 failed
```

### Step 5: Verify Operator Deployment

```bash
# Check status of deployed resources
kpt live status

# Check operator pods
kubectl get pods -n focom-operator-system

# Expected output:
# NAME                                                READY   STATUS    RESTARTS   AGE
# focom-operator-controller-manager-xxxxxxxxx-xxxxx   1/1     Running   0          30s

# Check operator logs
kubectl logs -n focom-operator-system -l control-plane=controller-manager --tail=50

# Wait for operator to be ready
kubectl wait --for=condition=available --timeout=60s \
  deployment/focom-operator-controller-manager -n focom-operator-system
```

### Step 6: Deploy ConfigSync

**⚠️ IMPORTANT:** The kpt package includes ONLY the operator. You must deploy ConfigSync separately for the GitOps workflow to function.

#### 6.1: Copy Git Secret

```bash
# Return to operator root directory
cd ..

# Copy gitea-secret from default to config-management-system namespace
kubectl get secret gitea-secret -n default -o yaml | \
  sed 's/namespace: default/namespace: config-management-system/' | \
  kubectl apply -f -

# Verify secret was copied
kubectl get secret gitea-secret -n config-management-system
```

**If the secret doesn't exist in default namespace:**
```bash
# Create it first (see config/samples/create-gitea-secret.sh)
cd config/samples
./create-gitea-secret.sh

# Then copy to config-management-system
kubectl get secret gitea-secret -n default -o yaml | \
  sed 's/namespace: default/namespace: config-management-system/' | \
  kubectl apply -f -
```

#### 6.2: Deploy ConfigSync RootSync

```bash
# Deploy the RootSync resource
kubectl apply -k config/configsync

# Expected output:
# rootsync.configsync.gke.io/focom-resources created
```

#### 6.3: Verify ConfigSync

```bash
# Check RootSync status
kubectl get rootsync focom-resources -n config-management-system

# Expected output (after a few seconds):
# NAME              RENDERINGCOMMIT   SOURCECOMMIT      SYNCCOMMIT        SYNCERRORCOUNT
# focom-resources   <commit-hash>     <commit-hash>     <commit-hash>     0

# Check for errors
kubectl describe rootsync focom-resources -n config-management-system

# Check ConfigSync logs
kubectl logs -n config-management-system -l app=reconciler-manager --tail=50
```

**⚠️ Troubleshooting:** If SYNCERRORCOUNT > 0, check:
- Git repository is accessible
- Git secret has correct credentials
- Porch Repository CR is READY=True
- Git repository URL is correct in RootSync

### Step 7: Test End-to-End Flow

```bash
# Test the complete workflow: API → Porch → Git → ConfigSync → CR

# 1. Create a draft OCloud
curl -X POST http://localhost:8080/api/v1/o-clouds/draft \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "focom-system",
    "name": "test-ocloud",
    "description": "Test kpt deployment"
  }'

# 2. Approve the draft
curl -X POST http://localhost:8080/api/v1/o-clouds/test-ocloud/draft/approve

# 3. Wait for ConfigSync to sync (up to 15 seconds)
sleep 15

# 4. Verify CR was created by ConfigSync
kubectl get ocloud test-ocloud -n focom-system

# 5. Check ConfigSync annotations
kubectl get ocloud test-ocloud -n focom-system -o yaml | grep configmanagement

# Expected annotation:
# configmanagement.gke.io/managed: enabled
```

### Updating with kpt

```bash
# Regenerate package with new image
make kpt-package IMG=<registry>/<image>:<new-tag>

# Preview changes
cd kpt-package/
kpt live apply --dry-run

# Apply updates
kpt live apply

# kpt will update changed resources and leave others unchanged
```

### Undeploying with kpt

Complete removal requires removing both the operator and ConfigSync in the correct order.

#### Step 1: Remove ConfigSync RootSync

**⚠️ IMPORTANT:** Remove ConfigSync FIRST to stop it from recreating CRs.

```bash
# Remove the RootSync
kubectl delete -k config/configsync

# Expected output:
# rootsync.configsync.gke.io "focom-resources" deleted

# Verify RootSync is gone
kubectl get rootsync -n config-management-system
# Should show: No resources found

# Remove the Git secret from config-management-system
kubectl delete secret gitea-secret -n config-management-system
```

**What this does:**
- Stops ConfigSync from syncing Git to cluster
- ConfigSync will delete all CRs it was managing (OClouds, TemplateInfos, FPRs)
- This is the clean way to remove resources managed by ConfigSync

#### Step 2: Remove Operator with kpt

```bash
# Remove all operator resources managed by kpt
cd kpt-package/
kpt live destroy

# Expected output:
# namespace/focom-operator-system deleted
# customresourcedefinition.apiextensions.k8s.io/o-clouds.focom.nephio.org deleted
# customresourcedefinition.apiextensions.k8s.io/templateinfoes.provisioning.oran.org deleted
# ...
# 15 resource(s) deleted, 0 skipped
```

**What this removes:**
- Operator deployment
- CRDs (OCloud, TemplateInfo, FocomProvisioningRequest)
- RBAC resources (ClusterRoles, ClusterRoleBindings, ServiceAccount)
- Services (metrics, NBI)
- Namespace (focom-operator-system)

**⚠️ Warning:** Deleting CRDs will cascade delete ANY remaining custom resources (OClouds, TemplateInfos, FPRs) that weren't removed by ConfigSync.

#### Step 3: Verify Complete Removal

```bash
# Check operator namespace is gone
kubectl get namespace focom-operator-system
# Should show: Error from server (NotFound)

# Check CRDs are gone
kubectl get crd | grep -E "focom|provisioning.oran"
# Should show no results

# Check ConfigSync is gone
kubectl get rootsync -n config-management-system
# Should show: No resources found

# Check no CRs remain
kubectl get oclouds,templateinfoes,focomprovisioningrequests -A
# Should show: error: the server doesn't have a resource type "oclouds"
```

#### Complete Undeployment Script

```bash
#!/bin/bash
# Complete undeployment of FOCOM operator and ConfigSync

echo "Step 1: Removing ConfigSync RootSync..."
kubectl delete -k config/configsync
kubectl delete secret gitea-secret -n config-management-system --ignore-not-found=true

echo "Waiting for ConfigSync to clean up CRs..."
sleep 10

echo "Step 2: Removing operator with kpt..."
cd kpt-package/
kpt live destroy

echo "Step 3: Verifying cleanup..."
kubectl get namespace focom-operator-system 2>/dev/null && echo "⚠️ Namespace still exists" || echo "✅ Namespace removed"
kubectl get rootsync -n config-management-system 2>/dev/null | grep focom-resources && echo "⚠️ RootSync still exists" || echo "✅ RootSync removed"
kubectl get crd | grep -E "focom|provisioning.oran" && echo "⚠️ CRDs still exist" || echo "✅ CRDs removed"

echo "Undeployment complete!"
```

#### Partial Removal Options

**Remove only ConfigSync (keep operator running):**
```bash
kubectl delete -k config/configsync
kubectl delete secret gitea-secret -n config-management-system
```

**Remove only operator (keep ConfigSync):**
```bash
cd kpt-package/
kpt live destroy
# Note: ConfigSync will continue running but won't have an operator to create drafts
```

### kpt Best Practices

**1. Always preview before applying:**
```bash
kpt live apply --dry-run
```

**2. Check status after deployment:**
```bash
kpt live status
```

**3. Use inventory for tracking:**
```bash
# View what kpt is managing
kpt live status --output=table
```

**4. Keep package in Git:**
```bash
# Commit the initialized package
git add kpt-package/
git commit -m "Initialize kpt package for operator deployment"
```

### Troubleshooting kpt Deployment

**Package not initialized:**
```bash
# Error: package uninitialized
# Solution: Run kpt live init
cd kpt-package/
kpt live init --namespace focom-operator-system --inventory-id focom-operator
```

**Inventory conflict:**
```bash
# Error: inventory object already exists
# Solution: Use existing inventory or force reinit
kpt live init --force --namespace focom-operator-system --inventory-id focom-operator
```

**Resources already exist:**
```bash
# kpt will show conflicts if resources exist
# Solution: Either remove existing resources or use kubectl apply instead
kubectl delete -f kpt-package/focom-operator-bundle.yaml
kpt live apply
```

---

## Method 3: Direct kubectl Deployment

### Quick Deployment

If you don't need kpt's state tracking, deploy directly with kubectl:

```bash
# Generate the bundle
make kpt-package IMG=<registry>/<image>:<tag>

# Apply directly
kubectl apply -f kpt-package/focom-operator-bundle.yaml

# Deploy ConfigSync
kubectl get secret gitea-secret -n default -o yaml | \
  sed 's/namespace: default/namespace: config-management-system/' | \
  kubectl apply -f -
kubectl apply -k config/configsync
```

### Removing

```bash
# Remove operator
kubectl delete -f kpt-package/focom-operator-bundle.yaml

# Remove ConfigSync
kubectl delete -k config/configsync
kubectl delete secret gitea-secret -n config-management-system
```

---

## Comparison of Deployment Methods

| Feature | make deploy | kpt live apply | kubectl apply |
|---------|-------------|----------------|---------------|
| **State Tracking** | No | Yes | No |
| **Auto Pruning** | No | Yes | No |
| **Diff Preview** | No | Yes | No |
| **ConfigSync Integration** | Automatic | Manual | Manual |
| **Complexity** | Low | Medium | Low |
| **Best For** | Development | GitOps/Production | Quick testing |
| **Rollback** | Manual | `kpt live destroy` | Manual |

**Recommendations:**
- **Development:** Use `make deploy` for quick iterations
- **Production/GitOps:** Use `kpt live apply` for better tracking
- **CI/CD:** Use `kubectl apply` for simplicity
- **Testing:** Use `kubectl apply` for quick validation

---

## Upgrading

### Upgrade Operator

```bash
# Build new image
make docker-build docker-push IMG=<registry>/<image>:<new-tag>

# Update deployment
kubectl set image deployment/focom-operator-controller-manager \
  manager=<registry>/<image>:<new-tag> \
  -n focom-operator-system

# Or redeploy
make deploy IMG=<registry>/<image>:<new-tag>
```

### Upgrade ConfigSync Configuration

```bash
# Update RootSync
kubectl apply -f config/configsync/focom-resources-rootsync.yaml

# ConfigSync will automatically resync
```

## Backup and Recovery

### Backup

```bash
# Backup Git repository
cd /path/to/focom-resources
git clone --mirror http://172.18.0.200:3000/nephio/focom-resources.git backup.git

# Backup Kubernetes resources
kubectl get oclouds,templateinfoes,focomprovisioningrequests -A -o yaml > focom-crs-backup.yaml
```

### Recovery

```bash
# Restore from Git
# ConfigSync will automatically recreate CRs from Git

# Or manually apply backup
kubectl apply -f focom-crs-backup.yaml
```

## Uninstallation

### Complete Removal

```bash
# Undeploy operator and ConfigSync
make undeploy

# Remove CRDs (optional, will delete all CRs)
kubectl delete crd oclouds.focom.nephio.org
kubectl delete crd templateinfoes.provisioning.oran.org
kubectl delete crd focomprovisioningrequests.focom.nephio.org

# Remove namespaces (optional)
kubectl delete namespace focom-operator-system
kubectl delete namespace focom-system
```

### Keep Data (Remove Operator Only)

```bash
# Remove operator but keep CRDs and CRs
kubectl delete deployment focom-operator-controller-manager -n focom-operator-system
kubectl delete service focom-operator-controller-manager-metrics-service -n focom-operator-system

# Keep ConfigSync running to maintain CRs
```

## Production Considerations

### High Availability

```bash
# Increase replicas
kubectl scale deployment focom-operator-controller-manager --replicas=3 -n focom-operator-system

# Add pod anti-affinity
# (edit deployment to spread across nodes)
```

### Security

```bash
# Enable TLS verification
# Set in PorchStorageConfig: HTTPSVerify: true

# Use network policies
kubectl apply -f config/network-policies/

# Rotate service account tokens regularly
```

### Monitoring

```bash
# Enable metrics
kubectl port-forward -n focom-operator-system svc/focom-operator-controller-manager-metrics-service 8443:8443

# Add Prometheus scraping
# (add annotations to service)

# Set up alerts for:
# - RootSync SYNCERRORCOUNT > 0
# - Operator pod restarts
# - API error rate
```

### Performance Tuning

```bash
# Reduce ConfigSync poll interval (not recommended < 5s)
kubectl patch rootsync focom-resources -n config-management-system --type=merge -p '{"spec":{"git":{"period":"10s"}}}'

# Increase operator resources
kubectl set resources deployment focom-operator-controller-manager \
  --limits=cpu=1000m,memory=512Mi \
  --requests=cpu=500m,memory=256Mi \
  -n focom-operator-system
```

## Troubleshooting

See [TROUBLESHOOTING.md](../../docs/TROUBLESHOOTING.md) for detailed troubleshooting guide.

**Quick checks:**
```bash
# Check operator logs
kubectl logs -n focom-operator-system -l control-plane=controller-manager --tail=100

# Check ConfigSync status
kubectl get rootsync focom-resources -n config-management-system
kubectl describe rootsync focom-resources -n config-management-system

# Check Porch connectivity
kubectl get repositories -n default
kubectl get packagerevisions -n default
```

## Additional Resources

- **Architecture:** [ARCHITECTURE.md](../../docs/ARCHITECTURE.md)
- **Troubleshooting:** [TROUBLESHOOTING.md](../../docs/TROUBLESHOOTING.md)
- **Porch Setup:** [porch-setup.md](../../docs/porch-setup.md)
- **ConfigSync README:** [README.md](README.md)
- **API Documentation:** [OpenAPI Spec](../../api/openapi/focom-nbi-api.yaml)
