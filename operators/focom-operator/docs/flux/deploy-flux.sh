#!/bin/bash
set -e

echo "=========================================="
echo "FOCOM Flux Deployment Script"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Step 1: Verify Flux is installed
echo "Step 1/5: Verifying Flux installation..."
if kubectl get pods -n flux-system &>/dev/null; then
    FLUX_PODS=$(kubectl get pods -n flux-system --no-headers 2>/dev/null | wc -l)
    if [ "$FLUX_PODS" -ge 4 ]; then
        print_status "Flux is installed ($FLUX_PODS pods running)"
    else
        print_error "Flux is installed but not all pods are running"
        kubectl get pods -n flux-system
        exit 1
    fi
else
    print_error "Flux is not installed. Please install Flux first."
    echo "Install with: flux install"
    exit 1
fi

# Step 2: Check if gitea-secret exists in default namespace
echo ""
echo "Step 2/5: Checking Git secret..."
if kubectl get secret gitea-secret -n default &>/dev/null; then
    print_status "Git secret exists in default namespace"
else
    print_error "Git secret 'gitea-secret' not found in default namespace"
    echo "Create it with:"
    echo "  kubectl create secret generic gitea-secret -n default \\"
    echo "    --from-literal=username=<git-username> \\"
    echo "    --from-literal=password=<git-password>"
    exit 1
fi

# Step 3: Copy secret to flux-system namespace
echo ""
echo "Step 3/5: Copying Git secret to flux-system namespace..."
if kubectl get secret gitea-secret -n flux-system &>/dev/null; then
    print_warning "Secret already exists in flux-system namespace (skipping)"
else
    kubectl get secret gitea-secret -n default -o yaml | \
        sed 's/namespace: default/namespace: flux-system/' | \
        kubectl apply -f - &>/dev/null
    print_status "Git secret copied to flux-system namespace"
fi

# Step 4: Deploy Flux resources
echo ""
echo "Step 4/5: Deploying Flux resources..."
kubectl apply -k config/flux

# Wait a moment for resources to be created
sleep 2

# Step 5: Verify deployment
echo ""
echo "Step 5/5: Verifying deployment..."

# Check GitRepository
echo -n "Checking GitRepository... "
if kubectl get gitrepository focom-resources -n flux-system &>/dev/null; then
    print_status "Created"
else
    print_error "Not found"
    exit 1
fi

# Check Kustomization
echo -n "Checking Kustomization... "
if kubectl get kustomization focom-resources -n flux-system &>/dev/null; then
    print_status "Created"
else
    print_error "Not found"
    exit 1
fi

# Wait for GitRepository to be ready
echo ""
echo "Waiting for GitRepository to be ready (max 30s)..."
for i in {1..30}; do
    READY=$(kubectl get gitrepository focom-resources -n flux-system -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "Unknown")
    if [ "$READY" == "True" ]; then
        print_status "GitRepository is ready"
        break
    fi
    echo -n "."
    sleep 1
done

if [ "$READY" != "True" ]; then
    print_warning "GitRepository not ready yet. Check status with:"
    echo "  kubectl describe gitrepository focom-resources -n flux-system"
fi

# Wait for Kustomization to be ready
echo ""
echo "Waiting for Kustomization to be ready (max 30s)..."
for i in {1..30}; do
    READY=$(kubectl get kustomization focom-resources -n flux-system -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "Unknown")
    if [ "$READY" == "True" ]; then
        print_status "Kustomization is ready"
        break
    fi
    echo -n "."
    sleep 1
done

if [ "$READY" != "True" ]; then
    print_warning "Kustomization not ready yet. Check status with:"
    echo "  kubectl describe kustomization focom-resources -n flux-system"
fi

# Final status
echo ""
echo "=========================================="
echo "Deployment Summary"
echo "=========================================="

# GitRepository status
echo ""
echo "GitRepository Status:"
kubectl get gitrepository focom-resources -n flux-system

# Kustomization status
echo ""
echo "Kustomization Status:"
kubectl get kustomization focom-resources -n flux-system

# Check for synced CRs
echo ""
echo "Synced Custom Resources:"
CR_COUNT=$(kubectl get oclouds,templateinfoes,focomprovisioningrequests -A 2>/dev/null | grep -v NAMESPACE | wc -l || echo "0")
if [ "$CR_COUNT" -gt 0 ]; then
    print_status "$CR_COUNT custom resources synced"
    kubectl get oclouds,templateinfoes,focomprovisioningrequests -A
else
    print_warning "No custom resources synced yet (this is normal if Git repo is empty)"
fi

echo ""
echo "=========================================="
echo "Next Steps"
echo "=========================================="
echo ""
echo "1. Check detailed status:"
echo "   kubectl describe gitrepository focom-resources -n flux-system"
echo "   kubectl describe kustomization focom-resources -n flux-system"
echo ""
echo "2. Monitor Flux logs:"
echo "   kubectl logs -n flux-system -l app=source-controller --tail=50 -f"
echo "   kubectl logs -n flux-system -l app=kustomize-controller --tail=50 -f"
echo ""
echo "3. Test end-to-end flow:"
echo "   See config/flux/TESTING_GUIDE.md"
echo ""
echo "4. Force reconciliation:"
echo "   kubectl annotate gitrepository focom-resources -n flux-system reconcile.fluxcd.io/requestedAt=\"\$(date +%s)\""
echo ""

print_status "Flux deployment complete!"
