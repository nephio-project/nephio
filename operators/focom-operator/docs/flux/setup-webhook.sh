#!/bin/bash
set -e

echo "=========================================="
echo "Flux Webhook Setup for FOCOM Resources"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Step 1: Check if Receiver already exists
echo "Step 1/5: Checking for existing Receiver..."
if kubectl get receiver focom-resources -n flux-system &>/dev/null; then
    print_warning "Receiver already exists. Deleting and recreating..."
    kubectl delete receiver focom-resources -n flux-system
fi

# Step 2: Generate webhook token
echo ""
echo "Step 2/5: Generating webhook token..."
WEBHOOK_TOKEN=$(head -c 12 /dev/urandom | shasum | cut -d ' ' -f1)

# Step 3: Create webhook token secret
echo ""
echo "Step 3/5: Creating webhook token secret..."
kubectl create secret generic webhook-token \
  -n flux-system \
  --from-literal=token=$WEBHOOK_TOKEN \
  --dry-run=client -o yaml | kubectl apply -f -

print_status "Webhook token secret created"

# Step 4: Create Receiver
echo ""
echo "Step 4/5: Creating Receiver..."
kubectl apply -f config/flux/focom-resources-receiver.yaml

# Wait for Receiver to be ready
echo "Waiting for Receiver to be ready..."
for i in {1..30}; do
    READY=$(kubectl get receiver focom-resources -n flux-system -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "Unknown")
    if [ "$READY" == "True" ]; then
        print_status "Receiver is ready"
        break
    fi
    echo -n "."
    sleep 1
done

if [ "$READY" != "True" ]; then
    print_warning "Receiver not ready yet. Check status with: kubectl describe receiver focom-resources -n flux-system"
fi

# Step 5: Get webhook URL
echo ""
echo "Step 5/5: Getting webhook URL..."

# Get the webhook URL from the Receiver status
WEBHOOK_PATH=$(kubectl get receiver focom-resources -n flux-system -o jsonpath='{.status.webhookPath}')

if [ -z "$WEBHOOK_PATH" ]; then
    print_error "Could not get webhook path. Check Receiver status."
    exit 1
fi

# Determine the webhook URL based on your setup
# This assumes notification-controller is exposed via a service
NOTIFICATION_SERVICE=$(kubectl get svc -n flux-system -l app=notification-controller -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "notification-controller")

echo ""
echo "=========================================="
echo "Webhook Configuration"
echo "=========================================="
echo ""
print_status "Webhook token: $WEBHOOK_TOKEN"
echo ""
echo "Webhook URL (cluster-internal):"
echo "  http://${NOTIFICATION_SERVICE}.flux-system.svc.cluster.local${WEBHOOK_PATH}"
echo ""
echo "To expose the webhook externally, you have several options:"
echo ""
echo "Option 1: Port-forward (for testing)"
echo "  kubectl port-forward -n flux-system svc/${NOTIFICATION_SERVICE} 8080:80"
echo "  Webhook URL: http://localhost:8080${WEBHOOK_PATH}"
echo ""
echo "Option 2: NodePort (for local clusters)"
echo "  kubectl patch svc ${NOTIFICATION_SERVICE} -n flux-system -p '{\"spec\":{\"type\":\"NodePort\"}}'"
echo "  Get NodePort: kubectl get svc ${NOTIFICATION_SERVICE} -n flux-system"
echo "  Webhook URL: http://<node-ip>:<node-port>${WEBHOOK_PATH}"
echo ""
echo "Option 3: LoadBalancer (for cloud clusters)"
echo "  kubectl patch svc ${NOTIFICATION_SERVICE} -n flux-system -p '{\"spec\":{\"type\":\"LoadBalancer\"}}'"
echo "  Get LoadBalancer IP: kubectl get svc ${NOTIFICATION_SERVICE} -n flux-system"
echo "  Webhook URL: http://<loadbalancer-ip>${WEBHOOK_PATH}"
echo ""
echo "Option 4: Ingress (recommended for production)"
echo "  Create an Ingress resource pointing to ${NOTIFICATION_SERVICE}"
echo "  Webhook URL: https://your-domain.com${WEBHOOK_PATH}"
echo ""
echo "=========================================="
echo "Configure Git Webhook"
echo "=========================================="
echo ""
echo "In your Git server (Gitea), configure a webhook:"
echo ""
echo "1. Go to: http://172.18.0.200:3000/nephio/focom-resources/settings/hooks"
echo "2. Click 'Add Webhook' → 'Gitea' (or 'Generic')"
echo "3. Set Payload URL to the webhook URL above"
echo "4. Set Secret to: $WEBHOOK_TOKEN"
echo "5. Select events: 'Push'"
echo "6. Set Active: Yes"
echo "7. Click 'Add Webhook'"
echo ""
echo "Test the webhook:"
echo "  curl -X POST <webhook-url> -H 'X-Signature: sha1=$WEBHOOK_TOKEN'"
echo ""
print_status "Webhook setup complete!"
echo ""
echo "Verify with:"
echo "  kubectl get receiver focom-resources -n flux-system"
echo "  kubectl describe receiver focom-resources -n flux-system"
