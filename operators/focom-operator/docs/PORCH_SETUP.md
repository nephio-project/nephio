# Porch Storage Setup Guide

This guide provides step-by-step instructions for setting up Nephio Porch storage for the FOCOM NBI (North Bound Interface) system.

## Overview

The FOCOM operator supports two storage backends:
- **Stage 1 (Memory):** In-memory storage for development and testing
- **Stage 2 (Porch):** Persistent Git-backed storage using Nephio Porch

This guide covers the setup required for Stage 2 (Porch storage).

## Prerequisites

Before setting up Porch storage, ensure you have:

1. **Kubernetes Cluster:** A running Kubernetes cluster with kubectl access
2. **Nephio Porch:** Porch controller installed and running in the cluster
3. **Git Server:** A Git server (e.g., Gitea, GitHub, GitLab) accessible from the cluster
4. **kubectl:** Command-line tool configured to access your cluster

### Verify Porch Installation

Check that Porch is installed and running:

```bash
# Check Porch CRDs are installed
kubectl get crd | grep porch

# Check Porch API resources are available
kubectl api-resources | grep porch

# Check Porch controller is running
kubectl get pods -n porch-system
```

Expected output for CRDs:
```
packagerevs.config.porch.kpt.dev                             2025-09-11T10:19:43Z
packagevariants.config.porch.kpt.dev                         2025-09-11T10:19:43Z
packagevariantsets.config.porch.kpt.dev                      2025-09-11T10:19:43Z
repositories.config.porch.kpt.dev                            2025-09-11T10:19:43Z
```

Expected output for API resources (these are what we actually use):
```
NAME                       SHORTNAMES   APIVERSION                  NAMESPACED   KIND
repositories                            config.porch.kpt.dev/v1alpha1   true     Repository
packagerevisions                        porch.kpt.dev/v1alpha1          true     PackageRevision
packagerevisionresources                porch.kpt.dev/v1alpha1          true     PackageRevisionResources
```

Expected output for pods:
```
NAME                                READY   STATUS    RESTARTS   AGE
porch-controllers-<hash>            1/1     Running   0          5d
porch-server-<hash>                 1/1     Running   0          5d
```

**Note:** The Porch API server extends Kubernetes to provide `packagerevisions` and `packagerevisionresources` as API resources (in the `porch.kpt.dev` API group), even though the underlying CRDs have different names. Our code uses the API resources, not the CRDs directly.

## Setup Steps

### Step 1: Create Git Repository

Create a Git repository to store FOCOM resources. This repository will be managed by Porch and will contain all FOCOM resources (OClouds, TemplateInfo, FocomProvisioningRequests) as Kpt packages.

#### Option A: Using Gitea (Recommended for Development)

```bash
# Access Gitea UI (adjust URL for your environment)
# Example: http://172.18.0.200:3000

# Create a new repository:
# - Repository Name: focom-resources
# - Visibility: Public or Private (with credentials)
# - Initialize: Yes (with README or empty)
# - Branch: main

# Note the repository URL:
# Example: http://172.18.0.200:3000/nephio/focom-resources.git
```

#### Option B: Using GitHub

```bash
# Create repository via GitHub UI or CLI
gh repo create nephio/focom-resources --public

# Note the repository URL:
# Example: https://github.com/nephio/focom-resources.git
```

#### Option C: Using GitLab

```bash
# Create repository via GitLab UI or CLI
# Note the repository URL:
# Example: https://gitlab.com/nephio/focom-resources.git
```

### Step 2: Create Git Credentials Secret (Required for Porch Write Access)

Porch requires authentication credentials to write to the Git repository. Even if your repository is public for reading, Porch needs write access to commit PackageRevisions.

#### Important: Gitea Access Token Required

**Porch requires a Gitea personal access token** (not just username/password) for write operations.

**Step 2.1: Generate Gitea Access Token**

1. Log in to Gitea UI (e.g., http://172.18.0.200:3000)
2. Go to **Settings** → **Applications** → **Manage Access Tokens**
3. Click **Generate New Token**
4. Give it a name: `porch-focom-resources`
5. Select scopes: **repo** (read and write access to repositories)
6. Click **Generate Token**
7. **Copy the token immediately** (you won't be able to see it again)

**Step 2.2: Create Kubernetes Secret**

The secret must include three fields: `username`, `password`, and `token`.

**Option A: Using the Helper Script (Recommended)**

```bash
cd focom-operator/config/samples
./create-gitea-secret.sh
```

The script will prompt for:
- Username (default: nephio)
- Password
- Access token (from Step 2.1)

**Option B: Using kubectl Command**

```bash
kubectl create secret generic gitea-secret \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=<git-username> \
  --from-literal=password=<git-password> \
  --from-literal=bearerToken=<gitea-access-token> \
  --namespace=default
```

**Example:**
```bash
kubectl create secret generic gitea-secret \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=nephio \
  --from-literal=password=secret \
  --from-literal=bearerToken=a86c69a2a58eb45892d20a443c1e294f4e7c1e7a \
  --namespace=default
```

**Option C: Using YAML File**

1. Copy the template:
```bash
cp focom-operator/config/samples/gitea-secret.yaml /tmp/gitea-secret.yaml
```

2. Edit the file and replace placeholders:
```bash
# Encode your values
echo -n "nephio" | base64          # Username
echo -n "your-password" | base64   # Password
echo -n "your-token" | base64      # Access token (bearerToken)

# Edit the file with encoded values
vim /tmp/gitea-secret.yaml
```

3. Apply the secret:
```bash
kubectl apply -f /tmp/gitea-secret.yaml
```

**Step 2.3: Verify Secret**

```bash
# Check secret exists
kubectl get secret gitea-secret -n default

# Verify secret type and fields
kubectl get secret gitea-secret -n default -o yaml
```

Expected output:
```yaml
apiVersion: v1
kind: Secret
type: kubernetes.io/basic-auth
data:
  username: bmVwaGlv          # base64 encoded
  password: c2VjcmV0          # base64 encoded
  bearerToken: YTg2YzY5YTJhNThl... # base64 encoded
```

#### For SSH Authentication (Alternative)

If you prefer SSH authentication:

```bash
kubectl create secret generic git-ssh-secret \
  --from-file=ssh-privatekey=<path-to-private-key> \
  --namespace=default
```

**Note:** For public repositories, you still need credentials for Porch to write (commit) changes.

**Step 2.4: Restart Porch Components (Important!)**

After creating or updating the secret, you must restart Porch components to pick up the new credentials:

```bash
# Restart Porch controllers
kubectl rollout restart deployment/porch-controllers -n porch-system

# Restart Porch server
kubectl rollout restart deployment/porch-server -n porch-system

# Wait for rollout to complete
kubectl rollout status deployment/porch-controllers -n porch-system --timeout=60s
kubectl rollout status deployment/porch-server -n porch-system --timeout=60s
```

**Why this is necessary:** Porch loads secrets at startup. If you create or update the secret after Porch is running, you must restart Porch to pick up the changes.

### Step 3: Create Porch Repository CR

**⚠️ CRITICAL:** This must be completed BEFORE deploying the FOCOM operator!

Create a Porch Repository Custom Resource that connects Porch to your Git repository. This is the main Git repository where all FOCOM resources will be stored.

#### Option A: Use the Sample File (Recommended)

A sample file is provided at `config/samples/focom-porch-repository.yaml`:

```bash
# Edit the sample to match your Git repository
vim config/samples/focom-porch-repository.yaml

# Apply to cluster
kubectl apply -f config/samples/focom-porch-repository.yaml
```

#### Option B: Create Repository YAML Manually

Create a file named `focom-porch-repository.yaml`:

```yaml
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: focom-resources
  namespace: default
spec:
  # Repository type (always 'git' for Git-backed storage)
  type: git
  
  # Content type (always 'Package' for Kpt packages)
  content: Package
  
  # Deployment flag (set to true to enable package deployment)
  deployment: true
  
  # Git repository configuration
  git:
    # Git repository URL
    repo: http://172.18.0.200:3000/nephio/focom-resources.git
    
    # Branch to use (typically 'main' or 'master')
    branch: main
    
    # Directory within the repository (use '/' for root)
    directory: /
    
    # Secret reference (optional, only if authentication is required)
    secretRef:
      name: gitea-secret
```

#### Configuration Options

**For Public Repository (No Authentication):**
```yaml
spec:
  type: git
  content: Package
  deployment: true
  git:
    repo: http://172.18.0.200:3000/nephio/focom-resources.git
    branch: main
    directory: /
    # No secretRef needed
```

**For Private Repository with HTTP(S) Authentication:**
```yaml
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
```

**For Private Repository with SSH Authentication:**
```yaml
spec:
  type: git
  content: Package
  deployment: true
  git:
    repo: git@github.com:nephio/focom-resources.git
    branch: main
    directory: /
    secretRef:
      name: git-ssh-secret
```

#### Apply Repository CR

```bash
# If using the sample file
kubectl apply -f config/samples/focom-porch-repository.yaml

# Or if you created your own file
kubectl apply -f focom-porch-repository.yaml
```

### Step 4: Verify Repository is Ready

**⚠️ DO NOT proceed with FOCOM operator deployment until this shows READY=True!**

Wait for the Repository to become ready. This may take a few seconds as Porch clones the Git repository.

```bash
# Check Repository status
kubectl get repositories focom-resources -n default

# Expected output:
# NAME              TYPE   CONTENT   DEPLOYMENT   READY   ADDRESS
# focom-resources   git    Package   true         True    http://172.18.0.200:3000/nephio/focom-resources.git
```

**Important:** The `READY` column must show `True` before proceeding.

#### Detailed Status Check

```bash
# Get detailed status information
kubectl describe repository focom-resources -n default

# Look for:
# Status:
#   Conditions:
#     Type: Ready
#     Status: True
#     Reason: Ready
```

### Step 5: Configure FOCOM Operator

Configure the FOCOM operator to use Porch storage by setting environment variables.

#### For In-Cluster Deployment

Edit the operator deployment to add environment variables:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: focom-operator-controller-manager
  namespace: focom-operator-system
spec:
  template:
    spec:
      containers:
      - name: manager
        env:
        # Enable Porch storage backend
        - name: NBI_STORAGE_BACKEND
          value: "porch"
        
        # Set implementation stage to 2
        - name: NBI_STAGE
          value: "2"
        
        # Default namespace for FOCOM resources
        - name: FOCOM_NAMESPACE
          value: "focom-system"
        
        # Optional: Override default Kubernetes API URL
        # - name: KUBERNETES_BASE_URL
        #   value: "https://kubernetes.default.svc"
        
        # Optional: Override default namespace for PackageRevisions
        # - name: PORCH_NAMESPACE
        #   value: "default"
        
        # Optional: Override default repository name
        # - name: PORCH_REPOSITORY
        #   value: "focom-resources"
```

Apply the changes:

```bash
kubectl apply -f <deployment-file>.yaml

# Restart the operator to pick up new configuration
kubectl rollout restart deployment focom-operator-controller-manager -n focom-operator-system
```

**Note:** The service account token is automatically mounted by Kubernetes at `/var/run/secrets/kubernetes.io/serviceaccount/token` and will be used for authentication.

#### For Local Development

Set environment variables before running the operator:

```bash
# Required: Enable Porch storage
export NBI_STORAGE_BACKEND=porch
export NBI_STAGE=2

# Optional: Set default namespace for FOCOM resources
export FOCOM_NAMESPACE=focom-system

# Required: Point to your kubeconfig
export KUBECONFIG=/home/user/.kube/config

# Option 1: Let the operator auto-extract token from kubeconfig (Recommended)
# No additional configuration needed - token will be extracted automatically

# Option 2: Manually extract and set token from kubeconfig
export TOKEN=$(kubectl config view --raw -o jsonpath='{.users[0].user.token}')

# Option 3: Create a service account token
export TOKEN=$(kubectl create token focom-operator-dev --duration=24h)

# Option 4: Point to a token file
echo $TOKEN > /tmp/kube-token
export TOKEN=/tmp/kube-token

# Optional: Override Kubernetes API URL (if not using default from kubeconfig)
# export KUBERNETES_BASE_URL="https://api.your-cluster.com:6443"

# Optional: Override namespace (default: "default")
# export PORCH_NAMESPACE=default

# Optional: Override repository name (default: "focom-resources")
# export PORCH_REPOSITORY=focom-resources

# Run the operator
go run ./cmd/main.go
```

### Step 6: Verify Operator Startup

Check that the operator starts successfully with Porch storage:

```bash
# Check operator logs
kubectl logs -n focom-operator-system deployment/focom-operator-controller-manager -f

# Look for log messages:
# "Initializing NBI system" stage=2 storageBackend="porch"
# "Initialized Porch storage" repository="focom-resources" namespace="default"
# "NBI server started" port=8080
```

### Step 7: Test Porch Storage

Test that the operator can communicate with Porch:

```bash
# Create a test OCloud via REST API
curl -X POST http://localhost:8080/api/v1/o-clouds \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-ocloud",
    "description": "Test OCloud for Porch verification",
    "namespace": "default",
    "o2imsSecret": {
      "secretRef": {
        "name": "test-secret",
        "namespace": "default"
      }
    }
  }'

# Verify PackageRevision was created in Porch
kubectl get packagerevisions -n default

# Expected output should show a Published PackageRevision for the OCloud

# Verify the package was committed to Git
# Check your Git repository - you should see a new directory with:
# - Kptfile
# - ocloud.yaml
```

## Token Resolution

The FOCOM operator automatically resolves authentication tokens in the following priority order:

1. **Explicit Configuration:** Token provided in `PorchStorageConfig.Token`
2. **TOKEN Environment Variable:** Token string or file path in `TOKEN` env var
3. **In-Cluster Token:** `/var/run/secrets/kubernetes.io/serviceaccount/token` (automatic in-cluster)
4. **Kubeconfig Token:** Extracted from `KUBECONFIG` file (automatic for local development)

### Token Resolution Examples

#### In-Cluster (Automatic)
```bash
# No configuration needed - token automatically mounted by Kubernetes
# at /var/run/secrets/kubernetes.io/serviceaccount/token
```

#### Local Development (Automatic)
```bash
# Set KUBECONFIG - token will be extracted automatically
export KUBECONFIG=/home/user/.kube/config
# No TOKEN variable needed!
```

#### Local Development (Manual Token)
```bash
# Extract token from kubeconfig
export TOKEN=$(kubectl config view --raw -o jsonpath='{.users[0].user.token}')
```

#### Local Development (Service Account Token)
```bash
# Create a temporary service account token
export TOKEN=$(kubectl create token focom-operator-dev --duration=24h)
```

## RBAC Configuration

Ensure the FOCOM operator service account has the necessary permissions to access Porch resources.

### Required ClusterRole

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: focom-operator-porch-access
rules:
# Porch PackageRevision access
- apiGroups: ["porch.kpt.dev"]
  resources: ["packagerevisions", "packagerevisionresources"]
  verbs: ["get", "list", "create", "update", "delete", "patch", "watch"]

# Porch Repository access (read-only)
- apiGroups: ["config.porch.kpt.dev"]
  resources: ["repositories"]
  verbs: ["get", "list"]
```

### ClusterRoleBinding

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: focom-operator-porch-access
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: focom-operator-porch-access
subjects:
- kind: ServiceAccount
  name: focom-operator-controller-manager
  namespace: focom-operator-system
```

Apply the RBAC configuration:

```bash
kubectl apply -f focom-porch-rbac.yaml
```

## Troubleshooting

### Issue: Repository Not Ready

**Symptom:** `kubectl get repositories` shows `READY=False`

**Possible Causes:**
1. Git repository URL is incorrect or inaccessible
2. Authentication credentials are missing or incorrect
3. Network connectivity issues between cluster and Git server
4. Porch controller is not running

**Solutions:**

```bash
# Check Repository status details
kubectl describe repository focom-resources -n default

# Look for error messages in Status.Conditions

# Check Porch controller logs
kubectl logs -n porch-system deployment/porch-controllers

# Verify Git repository is accessible
curl -I http://172.18.0.200:3000/nephio/focom-resources.git

# Verify credentials secret exists (if using authentication)
kubectl get secret gitea-secret -n default

# Test Git access manually
git clone http://172.18.0.200:3000/nephio/focom-resources.git /tmp/test-clone
```

### Issue: Operator Fails to Initialize Porch Storage

**Symptom:** Operator logs show "failed to initialize Porch storage"

**Possible Causes:**
1. Token resolution failed
2. Kubernetes API URL is incorrect
3. Namespace or repository name is incorrect
4. RBAC permissions are missing

**Solutions:**

```bash
# Check operator logs for detailed error
kubectl logs -n focom-operator-system deployment/focom-operator-controller-manager

# Verify token is available (in-cluster)
kubectl exec -n focom-operator-system deployment/focom-operator-controller-manager -- \
  cat /var/run/secrets/kubernetes.io/serviceaccount/token

# Verify RBAC permissions
kubectl auth can-i get packagerevisions --as=system:serviceaccount:focom-operator-system:focom-operator-controller-manager

# Verify Repository exists
kubectl get repository focom-resources -n default

# Test Porch API access manually
TOKEN=$(kubectl create token focom-operator-controller-manager -n focom-operator-system)
curl -H "Authorization: Bearer $TOKEN" \
  https://kubernetes.default.svc/apis/porch.kpt.dev/v1alpha1/namespaces/default/packagerevisions
```

### Issue: PackageRevisions Not Created

**Symptom:** REST API calls succeed but no PackageRevisions appear in Porch

**Possible Causes:**
1. Wrong namespace configured
2. Wrong repository name configured
3. Porch controller not processing PackageRevisions
4. Git repository not writable

**Solutions:**

```bash
# Check operator configuration
kubectl get deployment focom-operator-controller-manager -n focom-operator-system -o yaml | grep -A 5 "env:"

# List all PackageRevisions in all namespaces
kubectl get packagerevisions --all-namespaces

# Check Porch controller logs
kubectl logs -n porch-system deployment/porch-controllers -f

# Verify Git repository has write access
# Check Git repository for new commits
```

### Issue: Token Resolution Fails (Local Development)

**Symptom:** "failed to resolve authentication token" error

**Possible Causes:**
1. KUBECONFIG not set or invalid
2. Kubeconfig doesn't contain token
3. TOKEN environment variable not set

**Solutions:**

```bash
# Verify KUBECONFIG is set and valid
echo $KUBECONFIG
kubectl config view

# Check if kubeconfig contains token
kubectl config view --raw -o jsonpath='{.users[0].user.token}'

# If no token in kubeconfig, create one manually
export TOKEN=$(kubectl create token default --duration=24h)

# Or point to service account token file
export TOKEN=/var/run/secrets/kubernetes.io/serviceaccount/token
```

### Issue: Git Authentication Errors

**Symptom:** "authentication required: Unauthorized" or "authentication failed" in Porch logs

**Possible Causes:**
1. Gitea secret missing or incorrect
2. Secret doesn't include access token
3. Secret type is wrong (must be `kubernetes.io/basic-auth`)
4. Porch hasn't picked up the new secret

**Solutions:**

```bash
# Check if secret exists
kubectl get secret gitea-secret -n default

# Verify secret has all required fields (username, password, bearerToken)
kubectl get secret gitea-secret -n default -o jsonpath='{.data}' | jq 'keys'
# Should show: ["bearerToken", "password", "username"]

# Verify secret type
kubectl get secret gitea-secret -n default -o jsonpath='{.type}'
# Should show: kubernetes.io/basic-auth

# Check Porch server logs for authentication errors
kubectl logs -n porch-system -l app=porch-server --tail=50 | grep -i auth

# If secret is correct but still failing, restart Porch
kubectl rollout restart deployment/porch-controllers -n porch-system
kubectl rollout restart deployment/porch-server -n porch-system

# Wait for restart
kubectl rollout status deployment/porch-server -n porch-system --timeout=60s
```

**Common Mistakes:**
- Using `type: Opaque` instead of `type: kubernetes.io/basic-auth`
- Missing the `bearerToken` field (only username/password not enough)
- Using password instead of Gitea access token in the `bearerToken` field
- Forgetting to restart Porch after creating/updating secret

### Issue: Unauthorized or Forbidden Errors

**Symptom:** "unauthorized" or "forbidden" errors in operator logs

**Possible Causes:**
1. Service account token is invalid or expired
2. RBAC permissions are missing or incorrect
3. Token doesn't have access to Porch APIs

**Solutions:**

```bash
# Verify RBAC permissions
kubectl auth can-i get packagerevisions \
  --as=system:serviceaccount:focom-operator-system:focom-operator-controller-manager

kubectl auth can-i create packagerevisions \
  --as=system:serviceaccount:focom-operator-system:focom-operator-controller-manager

# Check ClusterRole and ClusterRoleBinding
kubectl get clusterrole focom-operator-porch-access
kubectl get clusterrolebinding focom-operator-porch-access

# Recreate service account token (local development)
export TOKEN=$(kubectl create token focom-operator-controller-manager -n focom-operator-system --duration=24h)
```

### Issue: Git Commits Not Appearing

**Symptom:** PackageRevisions created but Git repository has no commits

**Possible Causes:**
1. Porch controller not syncing to Git
2. Git credentials are read-only
3. Git repository configuration is incorrect

**Solutions:**

```bash
# Check Porch controller logs for Git sync errors
kubectl logs -n porch-system deployment/porch-controllers | grep -i git

# Verify Repository configuration
kubectl get repository focom-resources -n default -o yaml

# Check Git repository directly
git clone http://172.18.0.200:3000/nephio/focom-resources.git /tmp/focom-check
cd /tmp/focom-check
git log --oneline

# Verify Git credentials have write access
# Try pushing a test commit manually using the same credentials
```

## Configuration Reference

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NBI_STORAGE_BACKEND` | Yes | `memory` | Storage backend type (`memory` or `porch`) |
| `NBI_STAGE` | Yes | `1` | Implementation stage (`1`, `2`, or `3`) |
| `FOCOM_NAMESPACE` | No | `focom-system` | Default namespace for FOCOM resources (OCloud, TemplateInfo, FocomProvisioningRequest) |
| `KUBERNETES_BASE_URL` | No | `https://kubernetes.default.svc` | Kubernetes API server URL |
| `TOKEN` | No | Auto-detected | Authentication token (string or file path) |
| `KUBECONFIG` | No | `~/.kube/config` | Path to kubeconfig file (local development) |
| `PORCH_NAMESPACE` | No | `default` | Namespace for PackageRevisions |
| `PORCH_REPOSITORY` | No | `focom-resources` | Porch repository name |

### PorchStorageConfig Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `KubernetesURL` | string | No | Auto-detected | Kubernetes API server URL |
| `Token` | string | No | Auto-detected | Authentication token |
| `Namespace` | string | Yes | - | Namespace for PackageRevisions |
| `Repository` | string | Yes | - | Porch repository name |
| `HTTPSVerify` | bool | No | `false` | Verify HTTPS certificates |

## Best Practices

### Production Deployments

1. **Use HTTPS with Certificate Verification:**
   ```go
   HTTPSVerify: true
   ```

2. **Use Private Git Repositories:**
   - Store sensitive configuration in private repositories
   - Use SSH keys or token-based authentication
   - Rotate credentials regularly

3. **Configure Resource Limits:**
   - Set appropriate CPU and memory limits for the operator
   - Monitor resource usage

4. **Enable Audit Logging:**
   - Enable Kubernetes audit logging for PackageRevision operations
   - Monitor Git commit history

5. **Backup Git Repository:**
   - Regularly backup the Git repository
   - Test restore procedures

### Development Environments

1. **Use Public Repositories:**
   - Simplifies setup (no credentials needed)
   - Faster iteration

2. **Use Local Git Server:**
   - Gitea or GitLab in Docker
   - Faster access, no external dependencies

3. **Disable HTTPS Verification:**
   ```go
   HTTPSVerify: false
   ```

4. **Use Automatic Token Resolution:**
   - Let the operator extract token from kubeconfig
   - No manual token management

## Next Steps

After completing the Porch setup:

1. **Test the REST API:** Create, read, update, and delete resources via the NBI REST API
2. **Verify Git Commits:** Check that resources are committed to the Git repository
3. **Test Draft Workflow:** Create drafts, validate, and approve them
4. **Test Revision History:** Create multiple revisions and verify history
5. **Monitor Performance:** Check operation latency and resource usage

## Quick Reference: Gitea Secret Setup

**TL;DR - Complete Secret Setup:**

```bash
# 1. Generate Gitea access token
# Go to Gitea UI → Settings → Applications → Generate New Token
# Copy the token

# 2. Create secret (replace values)
kubectl create secret generic gitea-secret \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=nephio \
  --from-literal=password=your-password \
  --from-literal=bearerToken=your-gitea-access-token \
  --namespace=default

# 3. Restart Porch (IMPORTANT!)
kubectl rollout restart deployment/porch-controllers -n porch-system
kubectl rollout restart deployment/porch-server -n porch-system

# 4. Verify
kubectl get secret gitea-secret -n default
kubectl rollout status deployment/porch-server -n porch-system
```

**Or use the helper script:**
```bash
cd focom-operator/config/samples
./create-gitea-secret.sh
```

**Secret Requirements:**
- ✅ Type: `kubernetes.io/basic-auth`
- ✅ Fields: `username`, `password`, `bearerToken`
- ✅ Token: Gitea personal access token (not just password)
- ✅ Restart: Must restart Porch after creating/updating

## Additional Resources

- [Nephio Porch Documentation](https://github.com/nephio-project/porch)
- [Kpt Package Documentation](https://kpt.dev/)
- [FOCOM Operator README](../README.md)
- [NBI Testing Guide](../README-NBI-Testing.md)
- [Gitea Secret Template](../config/samples/gitea-secret.yaml)
- [Secret Creation Script](../config/samples/create-gitea-secret.sh)

## Support

For issues or questions:
- Check the troubleshooting section above
- Review operator logs: `kubectl logs -n focom-operator-system deployment/focom-operator-controller-manager`
- Review Porch logs: `kubectl logs -n porch-system deployment/porch-controllers`
- Review Porch server logs: `kubectl logs -n porch-system deployment/porch-server`
- Open an issue in the project repository
