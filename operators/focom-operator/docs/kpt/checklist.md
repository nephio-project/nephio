# kpt Deployment with ConfigSync - Checklist

## ⚠️ Important Note

**The kpt package includes ONLY the FOCOM operator. ConfigSync must be deployed separately.**

## Pre-Deployment Checklist

- [ ] Git repository created (`focom-resources`)
- [ ] `gitea-secret` exists in `default` namespace
- [ ] Porch Repository CR applied and shows `READY=True`
- [ ] ConfigSync is installed in the cluster
- [ ] kubectl is configured and pointing to correct cluster
- [ ] kpt CLI is installed

## Deployment Steps

### 1. Generate kpt Package
```bash
cd focom-operator/
make kpt-package IMG=<your-registry>/<image>:<tag>
```

**Verify:**
```bash
grep "image:" kpt-package/focom-operator-bundle.yaml
# Should show your image
```

### 2. Initialize kpt Package (One-Time Only)
```bash
cd kpt-package/
kpt live init --namespace focom-operator-system --inventory-id focom-operator
```

**Verify:**
```bash
grep "inventory" Kptfile
# Should show inventory annotations
```

### 3. Deploy Operator
```bash
kpt live apply
```

**Verify:**
```bash
kpt live status
kubectl get pods -n focom-operator-system
# Should show Running pod
```

### 4. Wait for Operator Ready
```bash
kubectl wait --for=condition=available --timeout=60s \
  deployment/focom-operator-controller-manager -n focom-operator-system
```

### 5. Copy Git Secret
```bash
cd ..
kubectl get secret gitea-secret -n default -o yaml | \
  sed 's/namespace: default/namespace: config-management-system/' | \
  kubectl apply -f -
```

**Verify:**
```bash
kubectl get secret gitea-secret -n config-management-system
# Should exist
```

### 6. Deploy ConfigSync
```bash
kubectl apply -k config/configsync
```

**Verify:**
```bash
kubectl get rootsync focom-resources -n config-management-system
# Should show SYNCERRORCOUNT=0
```

### 7. Test End-to-End
```bash
# Create and approve a draft
curl -X POST http://localhost:8080/api/v1/o-clouds/draft \
  -H "Content-Type: application/json" \
  -d '{"namespace": "focom-system", "name": "test", "description": "Test"}'

curl -X POST http://localhost:8080/api/v1/o-clouds/test/draft/approve

# Wait for ConfigSync
sleep 15

# Verify CR created
kubectl get ocloud test -n focom-system
```

## Post-Deployment Checklist

- [ ] Operator pod is Running
- [ ] `kpt live status` shows all resources healthy
- [ ] ConfigSync RootSync shows SYNCERRORCOUNT=0
- [ ] Test API responds: `curl http://localhost:8080/health/live`
- [ ] End-to-end test passes (draft → approve → CR created)

## Uninstall Steps

**⚠️ CRITICAL:** Remove ConfigSync FIRST, then operator. This ensures clean CR deletion.

### 1. Remove ConfigSync RootSync
```bash
# Remove the RootSync
kubectl delete -k config/configsync

# Verify it's gone
kubectl get rootsync focom-resources -n config-management-system
# Should show: No resources found
```

**Verify:**
- [ ] RootSync deleted
- [ ] ConfigSync stops syncing

### 2. Remove Git Secret
```bash
kubectl delete secret gitea-secret -n config-management-system
```

**Verify:**
- [ ] Secret deleted from config-management-system namespace

### 3. Wait for ConfigSync Cleanup
```bash
# Wait for ConfigSync to remove CRs it was managing
sleep 10

# Check CRs are being removed
kubectl get oclouds,templateinfoes,focomprovisioningrequests -A
```

**Verify:**
- [ ] CRs are being deleted or already gone

### 4. Remove Operator with kpt
```bash
cd kpt-package/
kpt live destroy
```

**Verify:**
- [ ] Operator deployment deleted
- [ ] CRDs deleted
- [ ] Namespace deleted

### 5. Verify Complete Cleanup
```bash
# Check operator namespace is gone
kubectl get namespace focom-operator-system
# Should show: Error from server (NotFound)

# Check RootSync is gone
kubectl get rootsync -n config-management-system
# Should show: No resources found

# Check CRDs are gone
kubectl get crd | grep -E "focom|provisioning.oran"
# Should show no results
```

## Troubleshooting

### Operator Pod Not Starting
```bash
# Check events
kubectl describe pod -n focom-operator-system -l control-plane=controller-manager

# Common issues:
# - Wrong image (ImagePullBackOff)
# - Insufficient resources
# - RBAC issues
```

### ConfigSync Not Syncing
```bash
# Check RootSync status
kubectl describe rootsync focom-resources -n config-management-system

# Common issues:
# - Git secret missing or wrong credentials
# - Git repository URL incorrect
# - Porch Repository not READY
# - Network connectivity to Git server
```

### CRs Not Created After Approval
```bash
# Check ConfigSync logs
kubectl logs -n config-management-system -l app=reconciler-manager --tail=100

# Check Git repository
# Verify files were written by Porch

# Check RootSync sync status
kubectl get rootsync focom-resources -n config-management-system -o yaml
```

## Quick Reference

| What | Command |
|------|---------|
| Deploy operator | `kpt live apply` |
| Check status | `kpt live status` |
| Deploy ConfigSync | `kubectl apply -k config/configsync` |
| Check ConfigSync | `kubectl get rootsync -n config-management-system` |
| Remove operator | `kpt live destroy` |
| Remove ConfigSync | `kubectl delete -k config/configsync` |
| Update operator | Regenerate package, then `kpt live apply` |

## See Also

- [Full Deployment Guide](../DEPLOYMENT.md#method-2-kpt-based-deployment)
- [Quick Reference](quick-reference.md)
- [Undeployment Guide](undeployment.md)
- [Troubleshooting Guide](../TROUBLESHOOTING.md)
