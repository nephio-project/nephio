# What Gets Removed by `make undeploy-flux`

## TL;DR

**`make undeploy-flux` removes ONLY the FOCOM-specific resources. Flux itself remains installed and running.**

## What Gets Removed ❌

When you run `make undeploy-flux`, these resources are deleted:

### 1. FOCOM Flux Resources
```bash
# These are removed:
kubectl delete gitrepository focom-resources -n flux-system
kubectl delete kustomization focom-resources -n flux-system
```

**What they are:**
- `GitRepository` - Points to your focom-resources Git repo
- `Kustomization` - Tells Flux what to apply from that repo

### 2. FOCOM Git Secret
```bash
# This is removed:
kubectl delete secret gitea-secret -n flux-system
```

**What it is:**
- Git credentials for accessing the focom-resources repository

### 3. FOCOM Operator
```bash
# These are removed:
kubectl delete -k config/default
```

**What it includes:**
- Operator deployment
- CRDs (OCloud, TemplateInfo, FocomProvisioningRequest)
- RBAC resources
- Services
- Namespace: focom-operator-system

### 4. Custom Resources (Indirectly)
Flux automatically removes CRs it was managing when the Kustomization is deleted.

---

## What Does NOT Get Removed ✅

### 1. Flux Controllers
```bash
# These keep running:
kubectl get pods -n flux-system

# Output (still there):
# source-controller-xxx        1/1     Running
# kustomize-controller-xxx     1/1     Running
# helm-controller-xxx          1/1     Running
# notification-controller-xxx  1/1     Running
```

**Why:** Flux is a cluster-wide installation that may be used by other applications.

### 2. flux-system Namespace
```bash
# This remains:
kubectl get namespace flux-system
```

**Why:** Other Flux resources may exist in this namespace.

### 3. Other Flux Resources
```bash
# Other GitRepositories and Kustomizations remain:
kubectl get gitrepository,kustomization -n flux-system

# Only focom-resources is removed, others stay
```

**Why:** Your cluster may have other applications using Flux.

### 4. Flux CRDs
```bash
# These remain:
kubectl get crd | grep fluxcd

# Output (still there):
# gitrepositories.source.toolkit.fluxcd.io
# kustomizations.kustomize.toolkit.fluxcd.io
# helmreleases.helm.toolkit.fluxcd.io
# etc.
```

**Why:** Flux CRDs are cluster-wide and may be used by other resources.

---

## Verification After Undeploy

### Check What Was Removed

```bash
# FOCOM Flux resources (should be gone)
kubectl get gitrepository focom-resources -n flux-system
# Expected: Error from server (NotFound)

kubectl get kustomization focom-resources -n flux-system
# Expected: Error from server (NotFound)

# FOCOM operator (should be gone)
kubectl get pods -n focom-operator-system
# Expected: No resources found

# FOCOM CRDs (should be gone)
kubectl get crd | grep -E "focom|provisioning.oran"
# Expected: (no output)
```

### Check What Remains

```bash
# Flux controllers (should still be running)
kubectl get pods -n flux-system
# Expected: All Flux pods Running

# flux-system namespace (should still exist)
kubectl get namespace flux-system
# Expected: Active

# Other Flux resources (should still exist)
kubectl get gitrepository,kustomization -n flux-system
# Expected: Other resources (if any) still present
```

---

## Complete Removal Scenarios

### Scenario 1: Remove FOCOM but Keep Flux (Default)

```bash
# Remove FOCOM resources only
make undeploy-flux

# Result:
# ✅ FOCOM operator removed
# ✅ FOCOM Flux resources removed
# ✅ Flux controllers still running
# ✅ Can deploy other apps with Flux
```

### Scenario 2: Remove FOCOM and Flux

If you want to remove Flux entirely from your cluster:

```bash
# 1. Remove FOCOM resources
make undeploy-flux

# 2. Remove Flux itself (if you're sure!)
flux uninstall

# Or manually:
kubectl delete namespace flux-system
kubectl delete crd -l app.kubernetes.io/part-of=flux

# Result:
# ✅ FOCOM operator removed
# ✅ FOCOM Flux resources removed
# ✅ Flux controllers removed
# ✅ flux-system namespace removed
# ⚠️ All Flux resources in cluster removed
```

**⚠️ Warning:** Only remove Flux if you're certain no other applications are using it!

---

## Comparison with ConfigSync

| Aspect | `make undeploy` (ConfigSync) | `make undeploy-flux` (Flux) |
|--------|------------------------------|------------------------------|
| **Removes FOCOM operator** | ✅ Yes | ✅ Yes |
| **Removes sync resources** | ✅ RootSync | ✅ GitRepository + Kustomization |
| **Removes sync tool** | ❌ No (ConfigSync stays) | ❌ No (Flux stays) |
| **Removes CRs** | ✅ Yes | ✅ Yes |
| **Affects other apps** | ❌ No | ❌ No |

Both commands remove only FOCOM-specific resources, leaving the GitOps tool (ConfigSync or Flux) installed for other uses.

---

## FAQ

### Q: Will this break other applications using Flux?
**A:** No. Only FOCOM-specific Flux resources are removed. Other GitRepositories and Kustomizations remain untouched.

### Q: Can I redeploy FOCOM with Flux after undeploy?
**A:** Yes. Just run `make deploy-with-flux` again.

### Q: Do I need to reinstall Flux after undeploy?
**A:** No. Flux remains installed and ready to use.

### Q: What if I want to completely remove Flux?
**A:** Run `flux uninstall` after `make undeploy-flux`, but only if no other apps are using Flux.

### Q: Will this remove the flux-system namespace?
**A:** No. The namespace remains because Flux controllers run in it.

### Q: Can I switch back to ConfigSync after using Flux?
**A:** Yes. Run `make undeploy-flux`, then `make deploy` to switch to ConfigSync.

---

## See Also

- [MAKEFILE_USAGE.md](MAKEFILE_USAGE.md) - Complete Makefile documentation
- [README.md](README.md) - Flux configuration overview
- [Flux Uninstall Documentation](https://fluxcd.io/flux/installation/uninstall/) - How to remove Flux itself
