# Testing Flux with FOCOM Operator

This guide walks you through testing Flux as an alternative to ConfigSync with your existing FOCOM operator deployment.

## Prerequisites Checklist

Before starting, verify:

- [ ] Flux is installed and running
- [ ] FOCOM operator is deployed
- [ ] Porch Repository CR is ready
- [ ] Git repository exists and is accessible
- [ ] `gitea-secret` exists in `default` namespace

## Step-by-Step Testing

### Step 1: Verify Prerequisites

```bash
# 1. Check Flux is running
kubectl get pods -n flux-system

# Expected: All pods Running
# source-controller, kustomize-controller, helm-controller, notification-controller

# 2. Check FOCOM operator
kubectl get pods -n focom-operator-system

# Expected: focom-operator-controller-manager Running

# 3. Check Porch Repository
kubectl get repository focom-resources -n default

# Expected: READY=True

# 4. Check Git secret
kubectl get secret gitea-secret -n default

# Expected: Secret exists
```

### Step 2: Copy Git Secret to flux-system Namespace

```bash
# Copy the secret
kubectl get secret gitea-secret -n default -o yaml | \
  sed 's/namespace: default/namespace: flux-system/' | \
  kubectl apply -f -

# Verify
kubectl get secret gitea-secret -n flux-system
```

### Step 3: Deploy Flux Resources

```bash
# Apply Flux configuration
kubectl apply -k config/flux

# Expected output:
# gitrepository.source.toolkit.fluxcd.io/focom-resources created
# kustomization.kustomize.toolkit.fluxcd.io/focom-resources created
```

### Step 4: Verify Flux Resources

```bash
# Check GitRepository
kubectl get gitrepository focom-resources -n flux-system

# Expected: READY=True, STATUS shows "stored artifact"

# Check Kustomization
kubectl get kustomization focom-resources -n flux-system

# Expected: READY=True, STATUS shows "Applied revision"

# Get detailed status
kubectl describe gitrepository focom-resources -n flux-system
kubectl describe kustomization focom-resources -n flux-system
```

### Step 5: Test End-to-End Flow

Now test the complete workflow: API → Porch → Git → Flux → CR

```bash
# 1. Create a draft OCloud via API
curl -X POST http://localhost:8080/api/v1/o-clouds/draft \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "focom-system",
    "name": "flux-test-ocloud",
    "description": "Testing Flux sync",
    "spec": {
      "o2imsSecret": {
        "secretRef": {
          "name": "test-secret",
          "namespace": "focom-system"
        }
      }
    }
  }'

# Expected: Draft created successfully

# 2. Approve the draft
curl -X POST http://localhost:8080/api/v1/o-clouds/flux-test-ocloud/draft/approve

# Expected: Draft approved, PackageRevision published

# 3. Wait for Flux to sync (up to 15 seconds)
echo "Waiting for Flux to sync..."
sleep 15

# 4. Check if CR was created
kubectl get ocloud flux-test-ocloud -n focom-system

# Expected: OCloud resource exists

# 5. Verify it was created by Flux
kubectl get ocloud flux-test-ocloud -n focom-system -o yaml | grep -A 5 "kustomize.toolkit.fluxcd.io"

# Expected: Flux annotations present
```

### Step 6: Test Update Flow

```bash
# 1. Update the OCloud via API
curl -X PATCH http://localhost:8080/api/v1/o-clouds/flux-test-ocloud/draft \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated via Flux"
  }'

# 2. Approve the update
curl -X POST http://localhost:8080/api/v1/o-clouds/flux-test-ocloud/draft/approve

# 3. Wait for Flux
sleep 15

# 4. Verify update
kubectl get ocloud flux-test-ocloud -n focom-system -o yaml | grep "Updated via Flux"
```

### Step 7: Test Delete Flow

```bash
# 1. Delete via API
curl -X DELETE http://localhost:8080/api/v1/o-clouds/flux-test-ocloud

# 2. Wait for Flux to prune
sleep 15

# 3. Verify deletion
kubectl get ocloud flux-test-ocloud -n focom-system

# Expected: Error from server (NotFound)
```

### Step 8: Monitor Flux

```bash
# Watch Flux events
kubectl get events -n flux-system --sort-by='.lastTimestamp' --watch

# Watch GitRepository
kubectl get gitrepository focom-resources -n flux-system --watch

# Watch Kustomization
kubectl get kustomization focom-resources -n flux-system --watch

# Check logs
kubectl logs -n flux-system -l app=source-controller --tail=50 -f
kubectl logs -n flux-system -l app=kustomize-controller --tail=50 -f
```

## Verification Checklist

After testing, verify:

- [ ] GitRepository shows READY=True
- [ ] Kustomization shows READY=True
- [ ] CRs are created after approval
- [ ] CRs are updated when drafts are re-approved
- [ ] CRs are deleted when removed from Git
- [ ] Flux logs show successful reconciliation
- [ ] No errors in Flux events

## Comparison Test: ConfigSync vs Flux

If you want to compare both:

### Test 1: Sync Latency

**ConfigSync:**
```bash
# Approve a draft and time how long until CR appears
time (curl -X POST http://localhost:8080/api/v1/o-clouds/test1/draft/approve && \
      while ! kubectl get ocloud test1 -n focom-system 2>/dev/null; do sleep 1; done)
```

**Flux:**
```bash
# Same test with Flux
time (curl -X POST http://localhost:8080/api/v1/o-clouds/test2/draft/approve && \
      while ! kubectl get ocloud test2 -n focom-system 2>/dev/null; do sleep 1; done)
```

**Expected:** Both should be 0-15 seconds (polling interval)

### Test 2: Self-Healing

**Test:**
```bash
# Create a CR via Flux
curl -X POST http://localhost:8080/api/v1/o-clouds/healing-test/draft/approve
sleep 15

# Manually modify the CR
kubectl patch ocloud healing-test -n focom-system -p '{"spec":{"description":"Manual change"}}'

# Wait for Flux to revert (up to 15s)
sleep 15

# Check if reverted
kubectl get ocloud healing-test -n focom-system -o yaml | grep description
```

**Expected:** Flux should revert the manual change

### Test 3: Observability

**Flux:**
```bash
# Rich status information
kubectl describe kustomization focom-resources -n flux-system

# Shows:
# - Last reconciliation time
# - Applied revision (Git commit)
# - Health status of each resource
# - Detailed conditions
# - Events
```

**ConfigSync:**
```bash
# Basic status
kubectl describe rootsync focom-resources -n config-management-system

# Shows:
# - Sync status
# - Error count
# - Commit hashes
```

## Troubleshooting

### GitRepository Not Ready

```bash
# Check status
kubectl describe gitrepository focom-resources -n flux-system

# Common issues:
# 1. Git URL incorrect
kubectl get gitrepository focom-resources -n flux-system -o yaml | grep url

# 2. Secret missing or wrong
kubectl get secret gitea-secret -n flux-system
kubectl get secret gitea-secret -n flux-system -o yaml

# 3. Network connectivity
kubectl run -it --rm debug --image=alpine --restart=Never -- sh
apk add git curl
curl http://172.18.0.200:3000/nephio/focom-resources.git
git clone http://172.18.0.200:3000/nephio/focom-resources.git

# 4. Check source-controller logs
kubectl logs -n flux-system -l app=source-controller --tail=100
```

### Kustomization Not Ready

```bash
# Check status
kubectl describe kustomization focom-resources -n flux-system

# Common issues:
# 1. Invalid YAML in Git
git clone http://172.18.0.200:3000/nephio/focom-resources.git
kubectl apply --dry-run=client -f focom-resources/

# 2. CRDs not installed
kubectl get crd | grep -E "focom|provisioning.oran"

# 3. Check kustomize-controller logs
kubectl logs -n flux-system -l app=kustomize-controller --tail=100
```

### CRs Not Created

```bash
# 1. Check Git repository has files
git clone http://172.18.0.200:3000/nephio/focom-resources.git
ls -la focom-resources/

# 2. Check Flux is syncing
kubectl get gitrepository,kustomization -n flux-system

# 3. Check for errors
kubectl get events -n flux-system --field-selector involvedObject.name=focom-resources

# 4. Manually test applying
kubectl apply -f focom-resources/ocloud-*/
```

### Kptfile Warnings

```bash
# Check if Kptfiles are being ignored
kubectl get gitrepository focom-resources -n flux-system -o yaml | grep -A 10 ignore

# Should show:
# ignore: |
#   **/Kptfile

# If not, update the GitRepository
kubectl edit gitrepository focom-resources -n flux-system
```

## Performance Testing

### Test Sync Performance

```bash
# Create 10 OClouds rapidly
for i in {1..10}; do
  curl -X POST http://localhost:8080/api/v1/o-clouds/draft \
    -H "Content-Type: application/json" \
    -d "{\"namespace\": \"focom-system\", \"name\": \"perf-test-$i\", \"description\": \"Performance test $i\"}"
  
  curl -X POST http://localhost:8080/api/v1/o-clouds/perf-test-$i/draft/approve
done

# Wait for Flux
sleep 20

# Check how many were created
kubectl get oclouds -n focom-system | grep perf-test | wc -l

# Expected: 10
```

### Monitor Resource Usage

```bash
# Check Flux controller resource usage
kubectl top pods -n flux-system

# Check operator resource usage
kubectl top pods -n focom-operator-system
```

## Cleanup

### Remove Test Resources

```bash
# Delete test OClouds
kubectl delete oclouds -n focom-system -l test=flux

# Or delete all
kubectl delete oclouds -n focom-system --all
```

### Remove Flux Configuration (Optional)

```bash
# Remove Flux resources
kubectl delete -k config/flux

# Remove secret
kubectl delete secret gitea-secret -n flux-system

# Verify cleanup
kubectl get gitrepository,kustomization -n flux-system | grep focom-resources
```

## Next Steps

After successful testing:

1. **Document findings** - Note any differences from ConfigSync
2. **Update deployment docs** - Add Flux as an option
3. **Create migration guide** - If switching from ConfigSync
4. **Set up monitoring** - Add Prometheus metrics, alerts
5. **Configure webhooks** - For instant sync (optional)
6. **Add notifications** - Slack/Teams alerts (optional)

## See Also

- [Flux Configuration README](README.md)
- [Flux vs ConfigSync Investigation](../../.kiro/specs/focom-operator/porch-storage-implementation/research/flux-vs-configsync-investigation.md)
- [Main Deployment Guide](../../docs/DEPLOYMENT.md)
