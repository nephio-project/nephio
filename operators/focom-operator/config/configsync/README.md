# ConfigSync Setup for FOCOM Resources

This directory contains ConfigSync configuration to enable GitOps for FOCOM resources.

## Overview

Instead of using a custom sync controller, we leverage ConfigSync to automatically sync FOCOM resources from the Git repository to Kubernetes CRs.

**Flow:**
```
REST API → Porch → Git Repository (focom-resources)
                        ↓
                   ConfigSync (RootSync)
                        ↓
                   Kubernetes CRs (OCloud, TemplateInfo, FPR)
```

## Setup Instructions

### Automatic Deployment (Recommended)

The ConfigSync RootSync is automatically deployed as part of the FOCOM operator deployment:

```bash
# From the focom-operator directory
make deploy IMG=<your-image>
```

This will:
1. Deploy the FOCOM operator
2. Copy the gitea-secret to config-management-system namespace
3. Deploy the ConfigSync RootSync
4. Start syncing FOCOM resources from Git

### Manual Deployment (Alternative)

If you need to deploy ConfigSync separately:

#### 1. Copy Git Credentials

```bash
./copy-gitea-secret.sh
```

This copies the `gitea-secret` from the `default` namespace to `config-management-system`.

#### 2. Apply the RootSync

```bash
kubectl apply -f focom-resources-rootsync.yaml
```

### Verify ConfigSync is Working

Check the RootSync status:

```bash
kubectl get rootsync focom-resources -n config-management-system
```

Expected output:
```
NAME              RENDERINGCOMMIT   SOURCECOMMIT      SYNCCOMMIT        SYNCERRORCOUNT
focom-resources   <commit-hash>     <commit-hash>     <commit-hash>     0
```

Check for errors:

```bash
kubectl describe rootsync focom-resources -n config-management-system
```

### 4. Test the Flow

1. **Create a draft via REST API:**
   ```bash
   curl -X POST http://localhost:8080/o-clouds/my-test-cloud/draft \
     -H "Content-Type: application/json" \
     -d '{
       "name": "Test Cloud",
       "description": "Testing ConfigSync",
       "namespace": "focom-system"
     }'
   ```

2. **Approve the draft:**
   ```bash
   curl -X POST http://localhost:8080/o-clouds/my-test-cloud/draft/approve
   ```

3. **Wait for ConfigSync (15 seconds max):**
   ConfigSync polls every 15 seconds

4. **Verify CR was created:**
   ```bash
   kubectl get ocloud my-test-cloud -n focom-system
   ```

## How It Works

### Git Repository Structure

Porch creates packages in the Git repository like:
```
focom-resources/
├── ocloud-123-v1/
│   ├── Kptfile
│   └── ocloud.yaml          ← ConfigSync syncs this
├── templateinfo-456-v1/
│   ├── Kptfile
│   └── templateinfo.yaml    ← ConfigSync syncs this
```

### ConfigSync Behavior

- **Watches:** `http://172.18.0.200:3000/nephio/focom-resources.git`
- **Syncs:** Every 15 seconds
- **Creates:** CRs based on YAML files in the repo
- **Respects:** Namespace specified in each YAML's `metadata.namespace`
- **Deletes:** CRs when files are removed from Git
- **Ignores:** Non-Kubernetes YAML files (like Kptfile)

### Namespace Handling

The RootSync is configured with `namespaceStrategy: implicit`, which means:
- ConfigSync will automatically create namespaces if they don't exist
- Each CR's `metadata.namespace` field determines where it's created
- No need to pre-create the `focom-system` namespace

## Troubleshooting

### Check RootSync Status

```bash
kubectl get rootsync focom-resources -n config-management-system -o yaml
```

Look at the `status` section for errors.

### Check ConfigSync Logs

```bash
# Reconciler logs
kubectl logs -n config-management-system -l app=reconciler-manager --tail=100

# Git-sync logs
kubectl logs -n config-management-system -l app=git-sync --tail=100
```

### Common Issues

**1. Authentication Failure**
- Verify secret exists: `kubectl get secret gitea-secret -n config-management-system`
- Check secret has correct keys: `username`, `token`, `password`

**2. Kptfile Errors**
- ConfigSync might try to apply Kptfiles
- Solution: Add resource filtering in the RootSync spec (see commented section)

**3. Namespace Not Found**
- Ensure `namespaceStrategy: implicit` is set
- Or pre-create the namespace: `kubectl create namespace focom-system`

**4. CRs Not Appearing**
- Check if Git repo has the files: Browse to http://172.18.0.200:3000/nephio/focom-resources
- Verify RootSync is syncing: `kubectl get rootsync focom-resources -n config-management-system`
- Check for sync errors in status

## Advantages Over Custom Controller

✅ **No custom code** - ConfigSync is production-tested  
✅ **True GitOps** - Git is the source of truth  
✅ **Drift correction** - ConfigSync fixes manual changes  
✅ **Multi-cluster ready** - Can sync to multiple clusters  
✅ **Battle-tested** - Used in production by Google and others  

## Undeployment

### Automatic Undeployment (Recommended)

The ConfigSync RootSync is automatically removed when undeploying the FOCOM operator:

```bash
# From the focom-operator directory
make undeploy
```

This will:
1. Remove the ConfigSync RootSync
2. Delete the gitea-secret from config-management-system
3. Undeploy the FOCOM operator

**Note:** CRs managed by ConfigSync will be automatically deleted when the RootSync is removed.

### Manual Undeployment (Alternative)

If you need to remove ConfigSync separately:

```bash
kubectl delete rootsync focom-resources -n config-management-system
kubectl delete secret gitea-secret -n config-management-system
```

## Disabling ConfigSync

If you want to temporarily disable ConfigSync without removing it:

```bash
# Scale down the ConfigSync reconciler
kubectl scale deployment reconciler-manager -n config-management-system --replicas=0

# To re-enable
kubectl scale deployment reconciler-manager -n config-management-system --replicas=1
```

To permanently remove ConfigSync:

```bash
kubectl delete rootsync focom-resources -n config-management-system
```

## References

- [ConfigSync Documentation](https://cloud.google.com/anthos-config-management/docs/config-sync-overview)
- [RootSync API Reference](https://cloud.google.com/anthos-config-management/docs/reference/rootsync-reposync-fields)
