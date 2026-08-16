# kpt Undeployment Guide

## ⚠️ Critical: Correct Removal Order

**Always remove ConfigSync BEFORE the operator** to ensure clean deletion of custom resources.

## Why Order Matters

### ❌ Wrong Order (Operator First)
```bash
# DON'T DO THIS
kpt live destroy  # Removes operator and CRDs
kubectl delete -k config/configsync  # ConfigSync can't clean up CRs (CRDs are gone!)
```

**Problems:**
- CRDs deleted → CRs become orphaned
- ConfigSync can't clean up (no CRD to work with)
- CRs may get stuck in Terminating state
- Manual cleanup required

### ✅ Correct Order (ConfigSync First)
```bash
# DO THIS
kubectl delete -k config/configsync  # ConfigSync removes CRs it manages
sleep 10  # Wait for cleanup
kpt live destroy  # Then remove operator and CRDs
```

**Benefits:**
- ConfigSync cleanly removes all CRs
- CRDs deleted after CRs are gone
- No orphaned resources
- No stuck finalizers

## Complete Undeployment Procedure

### Step-by-Step

```bash
# 1. Remove ConfigSync RootSync
kubectl delete -k config/configsync

# Expected output:
# rootsync.configsync.gke.io "focom-resources" deleted

# 2. Remove Git secret
kubectl delete secret gitea-secret -n config-management-system

# Expected output:
# secret "gitea-secret" deleted

# 3. Wait for ConfigSync to clean up CRs
echo "Waiting for ConfigSync to remove CRs..."
sleep 10

# 4. Verify CRs are being removed
kubectl get oclouds,templateinfoes,focomprovisioningrequests -A

# 5. Remove operator with kpt
cd kpt-package/
kpt live destroy

# Expected output:
# namespace/focom-operator-system deleted
# customresourcedefinition.apiextensions.k8s.io/o-clouds.focom.nephio.org deleted
# ...
# 15 resource(s) deleted, 0 skipped

# 6. Verify complete removal
kubectl get namespace focom-operator-system
# Expected: Error from server (NotFound): namespaces "focom-operator-system" not found

kubectl get rootsync -n config-management-system
# Expected: No resources found

kubectl get crd | grep -E "focom|provisioning.oran"
# Expected: (no output)
```

## Automated Undeployment Script

Save this as `undeploy-focom-kpt.sh`:

```bash
#!/bin/bash
set -e

echo "=========================================="
echo "FOCOM Operator Undeployment (kpt method)"
echo "=========================================="
echo ""

# Step 1: Remove ConfigSync
echo "Step 1/5: Removing ConfigSync RootSync..."
kubectl delete -k config/configsync || echo "RootSync already removed or not found"

# Step 2: Remove Git secret
echo "Step 2/5: Removing Git secret..."
kubectl delete secret gitea-secret -n config-management-system --ignore-not-found=true

# Step 3: Wait for ConfigSync cleanup
echo "Step 3/5: Waiting for ConfigSync to clean up CRs (10 seconds)..."
sleep 10

# Step 4: Check remaining CRs
echo "Step 4/5: Checking for remaining CRs..."
REMAINING_CRS=$(kubectl get oclouds,templateinfoes,focomprovisioningrequests -A 2>/dev/null | wc -l)
if [ "$REMAINING_CRS" -gt 1 ]; then
    echo "⚠️  Warning: $((REMAINING_CRS-1)) CRs still exist. They will be deleted with CRDs."
    kubectl get oclouds,templateinfoes,focomprovisioningrequests -A
else
    echo "✅ All CRs removed by ConfigSync"
fi

# Step 5: Remove operator
echo "Step 5/5: Removing operator with kpt..."
cd kpt-package/
kpt live destroy

# Verification
echo ""
echo "=========================================="
echo "Verification"
echo "=========================================="

# Check namespace
if kubectl get namespace focom-operator-system 2>/dev/null; then
    echo "⚠️  Namespace focom-operator-system still exists"
else
    echo "✅ Namespace focom-operator-system removed"
fi

# Check RootSync
if kubectl get rootsync focom-resources -n config-management-system 2>/dev/null; then
    echo "⚠️  RootSync focom-resources still exists"
else
    echo "✅ RootSync focom-resources removed"
fi

# Check CRDs
if kubectl get crd 2>/dev/null | grep -E "focom|provisioning.oran" > /dev/null; then
    echo "⚠️  FOCOM CRDs still exist"
    kubectl get crd | grep -E "focom|provisioning.oran"
else
    echo "✅ All FOCOM CRDs removed"
fi

echo ""
echo "=========================================="
echo "Undeployment Complete!"
echo "=========================================="
```

Make it executable:
```bash
chmod +x undeploy-focom-kpt.sh
```

Run it:
```bash
./undeploy-focom-kpt.sh
```

## Verification Checklist

After undeployment, verify:

- [ ] Operator namespace deleted: `kubectl get namespace focom-operator-system` → NotFound
- [ ] RootSync deleted: `kubectl get rootsync -n config-management-system` → No resources found
- [ ] CRDs deleted: `kubectl get crd | grep -E "focom|provisioning.oran"` → No results
- [ ] No CRs remain: `kubectl get oclouds,templateinfoes,focomprovisioningrequests -A` → Error (no CRD)
- [ ] Git secret removed: `kubectl get secret gitea-secret -n config-management-system` → NotFound

## Troubleshooting

### CRs Stuck in Terminating State

**Cause:** Operator was removed before ConfigSync, leaving CRs with finalizers.

**Solution:**
```bash
# List stuck CRs
kubectl get oclouds,templateinfoes,focomprovisioningrequests -A | grep Terminating

# Remove finalizers manually
kubectl patch ocloud <name> -n <namespace> -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl patch templateinfo <name> -n <namespace> -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl patch focomprovisioningrequest <name> -n <namespace> -p '{"metadata":{"finalizers":[]}}' --type=merge
```

### RootSync Won't Delete

**Cause:** ConfigSync controller may be stuck or not running.

**Solution:**
```bash
# Force delete
kubectl delete rootsync focom-resources -n config-management-system --force --grace-period=0

# If still stuck, remove finalizers
kubectl patch rootsync focom-resources -n config-management-system -p '{"metadata":{"finalizers":[]}}' --type=merge
```

### Namespace Stuck in Terminating

**Cause:** Resources in the namespace have finalizers or are still being deleted.

**Solution:**
```bash
# Check what's stuck
kubectl get all -n focom-operator-system

# Force delete stuck resources
kubectl delete all --all -n focom-operator-system --force --grace-period=0

# If namespace still stuck, remove finalizers
kubectl get namespace focom-operator-system -o json | \
  jq '.spec.finalizers = []' | \
  kubectl replace --raw "/api/v1/namespaces/focom-operator-system/finalize" -f -
```

### kpt live destroy Fails

**Cause:** Inventory object not found or corrupted.

**Solution:**
```bash
# Check inventory
kpt live status

# If inventory is broken, use kubectl directly
kubectl delete -f kpt-package/focom-operator-bundle.yaml

# Or delete resources individually
kubectl delete namespace focom-operator-system
kubectl delete crd oclouds.focom.nephio.org
kubectl delete crd templateinfoes.provisioning.oran.org
kubectl delete crd focomprovisioningrequests.focom.nephio.org
```

## Partial Removal Scenarios

### Remove Only ConfigSync (Keep Operator)

**Use case:** Testing without GitOps, or switching to different sync mechanism.

```bash
kubectl delete -k config/configsync
kubectl delete secret gitea-secret -n config-management-system
```

**Result:**
- Operator continues running
- Can still create drafts via API
- No automatic CR creation (no ConfigSync)
- CRs must be created manually

### Remove Only Operator (Keep ConfigSync)

**Use case:** Upgrading operator, or testing ConfigSync independently.

```bash
cd kpt-package/
kpt live destroy
```

**Result:**
- ConfigSync continues syncing
- No API to create new drafts
- Existing CRs in Git will still be synced
- Can manually add YAML to Git

## Recovery After Failed Undeployment

If undeployment fails and cluster is in a bad state:

```bash
# Nuclear option: Force remove everything
kubectl delete namespace focom-operator-system --force --grace-period=0
kubectl delete namespace config-management-system --force --grace-period=0
kubectl delete crd oclouds.focom.nephio.org --force --grace-period=0
kubectl delete crd templateinfoes.provisioning.oran.org --force --grace-period=0
kubectl delete crd focomprovisioningrequests.focom.nephio.org --force --grace-period=0

# Clean up kpt inventory
cd kpt-package/
rm -rf .kpt-pipeline/

# Reinitialize if needed
kpt live init --namespace focom-operator-system --inventory-id focom-operator
```

## See Also

- [Deployment Guide](../DEPLOYMENT.md#method-2-kpt-based-deployment) - Full deployment documentation
- [Quick Reference](quick-reference.md) - Quick commands
- [Checklist](checklist.md) - Step-by-step checklist
- [Troubleshooting](../TROUBLESHOOTING.md) - General troubleshooting
