# FOCOM Operator Troubleshooting Guide

This guide helps diagnose and resolve common issues with the FOCOM operator, Porch storage, and ConfigSync integration.

## Quick Diagnostics

### Check Overall System Health

```bash
# 1. Check FOCOM operator
kubectl get pods -n focom-operator-system
kubectl logs -n focom-operator-system -l control-plane=controller-manager --tail=50

# 2. Check Porch
kubectl get repositories -n default
kubectl get packagerevisions -n default

# 3. Check ConfigSync
kubectl get rootsync focom-resources -n config-management-system
kubectl describe rootsync focom-resources -n config-management-system

# 4. Check FOCOM CRs
kubectl get oclouds,templateinfoes,focomprovisioningrequests -A
```

### API Health Checks

```bash
# Check if API is responding
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready

# Check API info
curl http://localhost:8080/api/info
```

## Common Issues

### 1. Docker Image Pull Issues

**Symptoms:**
- Operator pod stuck in `ImagePullBackOff` or `ErrImagePull` state
- Pod events show authentication errors
- Cannot pull private images from GitHub Container Registry

**Check the Issue:**
```bash
# Check pod status
kubectl get pods -n focom-operator-system

# Check pod events for image pull errors
kubectl describe pod -n focom-operator-system -l control-plane=controller-manager

# Look for events like:
# Failed to pull image "ghcr.io/org/focom/focom-operator:latest": 
# Error response from daemon: pull access denied
```

**Possible Causes & Solutions:**

#### Missing Authentication Secret
```bash
# Check if authentication secret exists
kubectl get secret ghcr-secret -n focom-operator-system

# If missing, create it
make docker-auth REGISTRY_USER=your-username REGISTRY_PASSWORD=your-token
```

#### Invalid GitHub Personal Access Token
```bash
# Delete and recreate secret with new token
kubectl delete secret ghcr-secret -n focom-operator-system
make docker-auth REGISTRY_USER=your-username REGISTRY_PASSWORD=new-valid-token

# Restart deployment to pick up new secret
kubectl rollout restart deployment/focom-operator-controller-manager -n focom-operator-system
```

#### Token Permissions
- Ensure PAT has `read:packages` scope
- Verify token hasn't expired
- Check if repository/organization has package access restrictions

#### Wrong Image Name or Tag
```bash
# Check what image the deployment is trying to pull
kubectl get deployment focom-operator-controller-manager -n focom-operator-system -o yaml | grep image:

# Verify the image exists in the registry
# For public images: docker pull ghcr.io/org/focom/focom-operator:tag
```

#### Test Image Pull Manually
```bash
# Test if authentication works
kubectl run test-pull --image=ghcr.io/your-org/focom/focom-operator:latest --rm -it --restart=Never

# If this fails with authentication error, the secret is not working
# If this succeeds, the issue is with the deployment configuration
```

**Get Help:**
```bash
# Get detailed authentication setup help
make docker-auth-help
```

#### Warning About Missing ghcr-secret with Public Images

**Symptoms:**
- Warning: "Unable to retrieve some image pull secrets (ghcr-secret); attempting to pull the image may not succeed"
- Pod starts successfully despite the warning

**Explanation:**
This warning is **harmless** and can be safely ignored when using public images. The deployment configuration includes `imagePullSecrets: ghcr-secret` to support private GitHub Container Registry images, but Kubernetes shows this warning when the secret doesn't exist.

**What happens:**
- Kubernetes tries to use the `ghcr-secret` for authentication
- The secret doesn't exist (because it's not needed for public images)
- Kubernetes shows a warning but continues with the image pull
- The public image pulls successfully without authentication
- The pod starts normally

**Action required:** None. The warning is cosmetic and doesn't affect functionality.

### 2. API Returns "Porch is not accessible"

**Symptoms:**
- API returns 500 error with message "Porch is not accessible"
- Health check fails

**Possible Causes:**
- Porch is not installed
- Kubernetes API server is unreachable
- Authentication token is invalid
- RBAC permissions are missing

**Diagnosis:**
```bash
# Check if Porch CRDs exist
kubectl get crd | grep porch

# Check if Porch API is available
kubectl api-resources | grep porch

# Check Porch pods
kubectl get pods -n porch-system

# Test API access manually
kubectl get packagerevisions -n default
```

**Solutions:**

**If Porch is not installed:**
```bash
# Install Porch (follow Nephio documentation)
# https://github.com/nephio-project/porch
```

**If authentication fails:**
```bash
# For in-cluster deployment, check service account
kubectl get serviceaccount focom-operator-controller-manager -n focom-operator-system

# For local development, check token
echo $TOKEN
# or
kubectl config view --raw -o jsonpath='{.users[0].user.token}'
```

**If RBAC is missing:**
```bash
# Check ClusterRole
kubectl get clusterrole focom-operator-manager-role -o yaml

# Check ClusterRoleBinding
kubectl get clusterrolebinding focom-operator-manager-rolebinding -o yaml

# Reapply RBAC
make deploy IMG=<your-image>
```

### 2. ConfigSync Not Syncing Resources

**Symptoms:**
- Resources approved via API but CRs not created
- RootSync shows SYNCERRORCOUNT > 0
- CRs missing ConfigSync annotations

**Diagnosis:**
```bash
# Check RootSync status
kubectl get rootsync focom-resources -n config-management-system

# Check for errors
kubectl describe rootsync focom-resources -n config-management-system

# Check ConfigSync logs
kubectl logs -n config-management-system -l app=reconciler-manager --tail=100

# Check if secret exists
kubectl get secret gitea-secret -n config-management-system
```

**Common Error Messages:**

**"authentication failed":**
```bash
# Check secret contents
kubectl get secret gitea-secret -n config-management-system -o yaml

# Verify secret has correct keys
# Required: username, token (or password)

# Recreate secret
kubectl delete secret gitea-secret -n config-management-system
kubectl get secret gitea-secret -n default -o yaml | \
  sed 's/namespace: default/namespace: config-management-system/' | \
  kubectl apply -f -
```

**"repository not found":**
```bash
# Check Git repository URL in RootSync
kubectl get rootsync focom-resources -n config-management-system -o yaml | grep repo

# Verify repository is accessible
curl http://172.18.0.200:3000/nephio/focom-resources.git

# Check Porch Repository CR
kubectl get repository focom-resources -n default -o yaml
```

**"invalid YAML":**
```bash
# Check Git repository for malformed YAML
# ConfigSync expects valid Kubernetes YAML

# View recent commits
cd /path/to/focom-resources
git log --oneline -10

# Check specific file
git show HEAD:demo-ocloud-01-v1/ocloud.yaml
```

### 3. Draft Operations Fail

**Symptoms:**
- CreateDraft returns 500 error
- UpdateDraft returns "not found"
- ValidateDraft returns "invalid state"

**Diagnosis:**
```bash
# List all PackageRevisions
kubectl get packagerevisions -n default

# Check specific PackageRevision
kubectl get packagerevision <name> -n default -o yaml

# Check PackageRevision lifecycle
kubectl get packagerevisions -n default -o custom-columns=NAME:.metadata.name,LIFECYCLE:.spec.lifecycle,PACKAGE:.spec.packageName
```

**Solutions:**

**"draft already exists":**
```bash
# Delete existing draft
curl -X DELETE http://localhost:8080/o-clouds/<id>/draft

# Or delete PackageRevision directly
kubectl delete packagerevision <name> -n default
```

**"not found":**
```bash
# Verify resource ID is correct
curl http://localhost:8080/o-clouds

# Check if PackageRevision exists
kubectl get packagerevisions -n default | grep <resource-id>
```

**"invalid state" (trying to validate already proposed):**
```bash
# Check current lifecycle state
kubectl get packagerevision <name> -n default -o jsonpath='{.spec.lifecycle}'

# If Proposed, either approve or reject
curl -X POST http://localhost:8080/o-clouds/<id>/draft/approve
# or
curl -X POST http://localhost:8080/o-clouds/<id>/draft/reject
```

### 4. Published PackageRevision Cannot Be Deleted

**Symptoms:**
- Delete operation returns error about "DeletionProposed"
- PackageRevision stuck in Published state

**Diagnosis:**
```bash
# Check PackageRevision lifecycle
kubectl get packagerevision <name> -n default -o jsonpath='{.spec.lifecycle}'
```

**Solution:**

This is handled automatically by the PorchStorage implementation. If you see this error, it means the code is not proposing deletion before deleting.

**Manual workaround:**
```bash
# Propose deletion first
kubectl patch packagerevision <name> -n default --type=merge -p '{"spec":{"lifecycle":"DeletionProposed"}}'

# Wait a moment
sleep 1

# Then delete
kubectl delete packagerevision <name> -n default
```

### 5. CRs Not Self-Healing

**Symptoms:**
- Manually deleted CR is not recreated
- Manually modified CR is not reverted

**Diagnosis:**
```bash
# Check RootSync status
kubectl get rootsync focom-resources -n config-management-system

# Check if CR has ConfigSync annotations
kubectl get ocloud <name> -n <namespace> -o yaml | grep configmanagement

# Check ConfigSync logs
kubectl logs -n config-management-system -l app=reconciler-manager --tail=50
```

**Solutions:**

**If RootSync is not syncing:**
```bash
# Check RootSync status
kubectl describe rootsync focom-resources -n config-management-system

# Look for errors in status.conditions
```

**If CR doesn't have ConfigSync annotations:**
```bash
# CR was not created by ConfigSync
# Delete it and let ConfigSync recreate it
kubectl delete ocloud <name> -n <namespace>

# Wait for ConfigSync to recreate (up to 15 seconds)
```

**If Git doesn't have the resource:**
```bash
# Check Git repository
cd /path/to/focom-resources
git pull
ls -la

# If missing, the resource was deleted from Git
# Recreate via API
curl -X POST http://localhost:8080/o-clouds/<id>/draft ...
```

### 6. Namespace Not Auto-Created

**Symptoms:**
- CR not created because namespace doesn't exist
- ConfigSync shows namespace error

**Diagnosis:**
```bash
# Check RootSync namespaceStrategy
kubectl get rootsync focom-resources -n config-management-system -o jsonpath='{.spec.override.namespaceStrategy}'

# Should return: implicit
```

**Solution:**
```bash
# If not set to implicit, update RootSync
kubectl patch rootsync focom-resources -n config-management-system --type=merge -p '{"spec":{"override":{"namespaceStrategy":"implicit"}}}'

# Or manually create namespace
kubectl create namespace <namespace-name>
```

### 7. Slow CR Creation (More Than 15 Seconds)

**Symptoms:**
- CR takes longer than 15 seconds to appear after approval

**Diagnosis:**
```bash
# Check ConfigSync poll period
kubectl get rootsync focom-resources -n config-management-system -o jsonpath='{.spec.git.period}'

# Should return: 15s

# Check last sync time
kubectl get rootsync focom-resources -n config-management-system -o jsonpath='{.status.sync.lastUpdate}'
```

**Solutions:**

**Reduce poll period (not recommended for production):**
```bash
kubectl patch rootsync focom-resources -n config-management-system --type=merge -p '{"spec":{"git":{"period":"5s"}}}'
```

**Check if Git commit actually happened:**
```bash
cd /path/to/focom-resources
git log --oneline -5

# Verify latest commit contains your resource
git show HEAD
```

### 8. Dependency Validation Fails

**Symptoms:**
- Cannot delete OCloud because FPR references it
- Cannot create FPR because OCloud doesn't exist

**Diagnosis:**
```bash
# Check what FPRs reference the OCloud
kubectl get focomprovisioningrequests -A -o yaml | grep -A 5 oCloudId

# Check if referenced OCloud exists
kubectl get ocloud <ocloud-id> -n <namespace>
```

**Solutions:**

**To delete OCloud with dependent FPRs:**
```bash
# First delete all dependent FPRs
curl -X DELETE http://localhost:8080/focom-provisioning-requests/<fpr-id>

# Then delete OCloud
curl -X DELETE http://localhost:8080/o-clouds/<ocloud-id>
```

**To create FPR with missing OCloud:**
```bash
# First create the OCloud
curl -X POST http://localhost:8080/o-clouds/draft ...

# Then create the FPR
curl -X POST http://localhost:8080/focom-provisioning-requests/draft ...
```

### 9. Git Repository Issues

**Symptoms:**
- Porch Repository shows READY=False
- ConfigSync cannot access Git

**Diagnosis:**
```bash
# Check Porch Repository status
kubectl get repository focom-resources -n default -o yaml

# Check Git server accessibility
curl http://172.18.0.200:3000/nephio/focom-resources.git

# Check Git credentials
kubectl get secret gitea-secret -n default -o yaml
```

**Solutions:**

**If Git server is unreachable:**
```bash
# Check network connectivity
ping 172.18.0.200

# Check if Gitea is running
kubectl get pods -n gitea  # if running in cluster
```

**If credentials are wrong:**
```bash
# Update secret
kubectl delete secret gitea-secret -n default
kubectl create secret generic gitea-secret \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=<username> \
  --from-literal=password=<password> \
  --from-literal=bearerToken=<gitea-access-token> \
  -n default

# Copy to config-management-system
kubectl get secret gitea-secret -n default -o yaml | \
  sed 's/namespace: default/namespace: config-management-system/' | \
  kubectl apply -f -
```

**If repository doesn't exist:**
```bash
# Create repository in Gitea UI or via API
# Then update Porch Repository CR
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

### 10. RBAC Permission Denied

**Symptoms:**
- API returns 403 Forbidden
- Logs show "forbidden: User cannot ..."

**Diagnosis:**
```bash
# Check service account
kubectl get serviceaccount focom-operator-controller-manager -n focom-operator-system

# Check ClusterRole
kubectl get clusterrole focom-operator-manager-role -o yaml

# Check ClusterRoleBinding
kubectl get clusterrolebinding focom-operator-manager-rolebinding -o yaml

# Test permissions
kubectl auth can-i get packagerevisions --as=system:serviceaccount:focom-operator-system:focom-operator-controller-manager
```

**Solution:**
```bash
# Reapply RBAC
kubectl apply -f config/rbac/

# Or redeploy operator
make deploy IMG=<your-image>
```

## Debugging Tips

### Enable Verbose Logging

```bash
# Set log level in operator deployment
kubectl set env deployment/focom-operator-controller-manager -n focom-operator-system LOG_LEVEL=debug

# View logs
kubectl logs -n focom-operator-system -l control-plane=controller-manager -f
```

### Inspect PackageRevision Contents

```bash
# Get PackageRevision name
kubectl get packagerevisions -n default

# Get PackageRevisionResources
kubectl get packagerevisionresources <name> -n default -o yaml

# View actual YAML content
kubectl get packagerevisionresources <name> -n default -o jsonpath='{.spec.resources}' | jq -r '.[] | select(.file=="ocloud.yaml") | .data' | base64 -d
```

### Test API Manually

```bash
# Create draft
curl -v -X POST http://localhost:8080/api/v1/o-clouds/draft \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "focom-system",
    "name": "test-ocloud",
    "description": "Test"
  }'

# Get draft
curl -v http://localhost:8080/api/v1/o-clouds/test-ocloud/draft

# Validate
curl -v -X POST http://localhost:8080/api/v1/o-clouds/test-ocloud/draft/validate

# Approve
curl -v -X POST http://localhost:8080/api/v1/o-clouds/test-ocloud/draft/approve

# Check CR (wait up to 15 seconds)
kubectl get ocloud test-ocloud -n focom-system
```

### Check Git Commit History

```bash
cd /path/to/focom-resources
git log --oneline --graph --all
git show <commit-hash>
```

### Force ConfigSync Resync

```bash
# Delete and recreate RootSync
kubectl delete rootsync focom-resources -n config-management-system
kubectl apply -f config/configsync/focom-resources-rootsync.yaml

# ConfigSync will resync all resources from Git
```

## Getting Help

### Collect Diagnostic Information

```bash
# Create diagnostic bundle
mkdir focom-diagnostics
cd focom-diagnostics

# Operator logs
kubectl logs -n focom-operator-system -l control-plane=controller-manager --tail=500 > operator-logs.txt

# ConfigSync logs
kubectl logs -n config-management-system -l app=reconciler-manager --tail=500 > configsync-logs.txt

# Porch logs
kubectl logs -n porch-system -l app=porch-server --tail=500 > porch-logs.txt

# Resource status
kubectl get packagerevisions -n default -o yaml > packagerevisions.yaml
kubectl get rootsync focom-resources -n config-management-system -o yaml > rootsync.yaml
kubectl get repository focom-resources -n default -o yaml > repository.yaml
kubectl get oclouds,templateinfoes,focomprovisioningrequests -A -o yaml > focom-crs.yaml

# RBAC
kubectl get clusterrole focom-operator-manager-role -o yaml > clusterrole.yaml
kubectl get clusterrolebinding focom-operator-manager-rolebinding -o yaml > clusterrolebinding.yaml

# Create tarball
cd ..
tar -czf focom-diagnostics.tar.gz focom-diagnostics/
```

### Useful Resources

- **Porch Documentation:** https://github.com/nephio-project/porch
- **ConfigSync Documentation:** https://cloud.google.com/anthos-config-management/docs/config-sync-overview
- **FOCOM Operator Issues:** https://github.com/your-org/focom-operator/issues
- **Nephio Community:** https://nephio.org/community

## Known Limitations

1. **CR Creation Delay:** CRs are created within 15 seconds (ConfigSync poll interval), not immediately
2. **Async Errors:** Sync errors not visible in API response, must check RootSync status
3. **No Webhook Notifications:** Cannot notify when ConfigSync creates CRs
4. **Single Repository:** Currently supports one Git repository per operator instance
5. **No Caching:** All operations hit Kubernetes API (no local cache)

## Best Practices

1. **Always check RootSync status** after approving resources
2. **Use health checks** before performing operations
3. **Monitor ConfigSync logs** for sync issues
4. **Keep Git repository clean** (don't manually edit)
5. **Use proper RBAC** (don't use cluster-admin)
6. **Test in development** before deploying to production
7. **Backup Git repository** regularly
8. **Document custom configurations** for your environment
