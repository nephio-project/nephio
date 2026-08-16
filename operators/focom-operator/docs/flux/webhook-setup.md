# Flux Webhook Setup Guide

## Overview

Webhooks enable instant synchronization instead of waiting for the 15-second polling interval. When you approve a draft in the FOCOM API, Porch writes to Git, Git triggers the webhook, and Flux immediately syncs the changes.

**Latency Comparison:**
- **Without webhook:** 0-15 seconds (polling interval)
- **With webhook:** <1 second (instant)

## Important: Webhook + Polling Work Together

**⚠️ Key Point:** Webhooks do NOT replace polling - they work alongside it!

When you enable webhooks:
- ✅ **Webhook triggers instant sync** - Most changes sync in <1 second
- ✅ **Polling continues every 15 seconds** - Provides reliable fallback
- ✅ **Both mechanisms run simultaneously** - This is intentional and recommended

**Why keep both?**
1. **Redundancy** - If webhook fails, polling catches changes within 15 seconds
2. **Reliability** - Webhook might miss events (network issues, Gitea restart, etc.)
3. **Drift detection** - Polling catches manual changes to Git
4. **Best practice** - Flux recommends this dual-mechanism approach

**Resource impact:** Minimal - polling is lightweight and both mechanisms coexist efficiently.

**To reduce polling frequency (optional):**
```yaml
# In focom-resources-gitrepository.yaml
spec:
  interval: 1m  # Reduce from 15s to 1 minute
```

This gives you instant webhook sync with a 1-minute fallback instead of 15 seconds.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  FOCOM API                                                       │
│  - Approve draft                                                 │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Porch                                                           │
│  - Write to Git                                                  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Git Server (Gitea)                                              │
│  - Commit received                                               │
│  - Trigger webhook ──────────────────────┐                      │
└──────────────────────────────────────────┼──────────────────────┘
                                           │
                                           │ HTTP POST
                                           │
                                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  Flux Notification Controller                                    │
│  - Receive webhook                                               │
│  - Validate token                                                │
│  - Trigger reconciliation ────────────┐                         │
└───────────────────────────────────────┼─────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────┐
│  Flux Source Controller                                          │
│  - Fetch from Git immediately                                    │
│  - Update artifact                                               │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Flux Kustomize Controller                                       │
│  - Apply changes immediately                                     │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Kubernetes CRs                                                  │
│  - OCloud, TemplateInfo, FPR created instantly                   │
└─────────────────────────────────────────────────────────────────┘
```

## Quick Setup

### Step 1: Run the Setup Script

```bash
cd focom-operator
./config/flux/setup-webhook.sh
```

This script will:
1. Generate a secure webhook token
2. Create the webhook-token secret
3. Create the Flux Receiver resource
4. Display the webhook URL and configuration instructions

### Step 2: Expose the Webhook Endpoint

Choose one of these methods based on your environment:

#### Option A: Port-Forward (Testing/Development)

```bash
# Forward the notification-controller service
kubectl port-forward -n flux-system svc/notification-controller 8080:80

# Webhook URL: http://localhost:8080/hook/<receiver-id>
```

**Pros:** Quick and easy for testing
**Cons:** Only accessible from your machine, stops when terminal closes

#### Option B: NodePort (Local Clusters)

```bash
# Patch the service to use NodePort
kubectl patch svc notification-controller -n flux-system \
  -p '{"spec":{"type":"NodePort"}}'

# Get the NodePort
kubectl get svc notification-controller -n flux-system

# Webhook URL: http://<node-ip>:<node-port>/hook/<receiver-id>
```

**Pros:** Accessible from network, persistent
**Cons:** Requires node IP to be accessible from Git server

#### Option C: LoadBalancer (Cloud Clusters)

```bash
# Patch the service to use LoadBalancer
kubectl patch svc notification-controller -n flux-system \
  -p '{"spec":{"type":"LoadBalancer"}}'

# Get the LoadBalancer IP
kubectl get svc notification-controller -n flux-system

# Webhook URL: http://<loadbalancer-ip>/hook/<receiver-id>
```

**Pros:** Automatic external IP, persistent
**Cons:** Requires cloud provider support, may incur costs

#### Option D: Ingress (Production)

Create an Ingress resource:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: flux-webhook
  namespace: flux-system
spec:
  rules:
  - host: flux-webhook.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: notification-controller
            port:
              number: 80
```

**Pros:** Production-ready, HTTPS support, custom domain
**Cons:** Requires Ingress controller and DNS configuration

### Step 3: Configure Git Webhook

#### For Gitea (Your Setup)

1. **Navigate to webhook settings:**
   ```
   http://172.18.0.200:3000/nephio/focom-resources/settings/hooks
   ```

2. **Click "Add Webhook" → "Gitea"**

3. **Configure webhook:**
   - **Payload URL:** `http://<webhook-url>/hook/<receiver-id>`
   - **HTTP Method:** POST
   - **POST Content Type:** application/json
   - **Secret:** `<webhook-token>` (from setup script output)
   - **Trigger On:** Push events
   - **Active:** ✓ Yes

4. **Click "Add Webhook"**

5. **Test the webhook:**
   - Click on the webhook you just created
   - Click "Test Delivery"
   - Check for a successful response (200 OK)

#### For GitHub

```yaml
# Webhook settings
Payload URL: http://<webhook-url>/hook/<receiver-id>
Content type: application/json
Secret: <webhook-token>
Events: Just the push event
Active: Yes
```

#### For GitLab

```yaml
# Webhook settings
URL: http://<webhook-url>/hook/<receiver-id>
Secret Token: <webhook-token>
Trigger: Push events
Enable SSL verification: No (if using HTTP)
```

## Verification

### Check Receiver Status

```bash
# Get Receiver status
kubectl get receiver focom-resources -n flux-system

# Expected output:
# NAME              READY   STATUS
# focom-resources   True    Receiver initialized for path: /hook/...

# Detailed status
kubectl describe receiver focom-resources -n flux-system
```

### Test the Webhook

#### Manual Test

```bash
# Get webhook URL and token from setup script output
WEBHOOK_URL="http://localhost:8080/hook/<receiver-id>"
WEBHOOK_TOKEN="<your-token>"

# Send test webhook
curl -X POST $WEBHOOK_URL \
  -H "X-Signature: sha1=$WEBHOOK_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"ref":"refs/heads/main"}'

# Expected response: {"status":"success"}
```

#### End-to-End Test

```bash
# 1. Create and approve a draft
curl -X POST http://localhost:8080/api/v1/o-clouds/draft \
  -H "Content-Type: application/json" \
  -d '{"namespace": "focom-system", "name": "webhook-test", "description": "Test webhook"}'

curl -X POST http://localhost:8080/api/v1/o-clouds/webhook-test/draft/approve

# 2. Check how long it takes for CR to appear
time kubectl wait --for=condition=Ready ocloud webhook-test -n focom-system --timeout=30s

# With webhook: Should be <2 seconds
# Without webhook: Could be up to 15 seconds
```

### Monitor Webhook Events

```bash
# Watch Receiver events
kubectl get events -n flux-system --field-selector involvedObject.name=focom-resources --watch

# Watch notification-controller logs
kubectl logs -n flux-system -l app=notification-controller -f

# Watch for reconciliation triggers
kubectl logs -n flux-system -l app=source-controller -f | grep "focom-resources"
```

## Troubleshooting

### Webhook Not Triggering

**Check Receiver status:**
```bash
kubectl describe receiver focom-resources -n flux-system
```

**Common issues:**
- Receiver not ready
- Webhook URL incorrect
- Token mismatch
- Network connectivity

### Git Server Can't Reach Webhook

**Problem:** Git server (Gitea) can't reach the webhook URL

**Solutions:**

1. **If using port-forward:** Git server must be able to reach your machine
   ```bash
   # Use NodePort or LoadBalancer instead
   ```

2. **If using NodePort:** Ensure node IP is accessible from Git server
   ```bash
   # Test connectivity from Git server
   curl http://<node-ip>:<node-port>/hook/<receiver-id>
   ```

3. **If using LoadBalancer:** Ensure LoadBalancer IP is accessible
   ```bash
   # Check LoadBalancer status
   kubectl get svc notification-controller -n flux-system
   ```

### Token Mismatch

**Problem:** Webhook returns 401 Unauthorized

**Solution:**
```bash
# Regenerate token
kubectl delete secret webhook-token -n flux-system
./config/flux/setup-webhook.sh

# Update Git webhook with new token
```

### Webhook Delivers but No Sync

**Check GitRepository reconciliation:**
```bash
kubectl describe gitrepository focom-resources -n flux-system
```

**Force reconciliation:**
```bash
kubectl annotate gitrepository focom-resources -n flux-system \
  reconcile.fluxcd.io/requestedAt="$(date +%s)"
```

### Receiver Not Created

**Check notification-controller:**
```bash
kubectl get pods -n flux-system -l app=notification-controller

# If not running, check logs
kubectl logs -n flux-system -l app=notification-controller
```

## Understanding Webhook + Polling Coexistence

### How They Work Together

When you enable webhooks, **both webhook and polling mechanisms operate simultaneously**:

```
Webhook Path (instant):
Git Push → Webhook → Flux → Sync (<1s)

Polling Path (fallback):
Flux checks Git every 15s → Sync if changed
```

### Why Both?

This dual-mechanism approach is **intentional and recommended** by Flux:

| Scenario | Webhook | Polling | Result |
|----------|---------|---------|--------|
| **Normal operation** | ✅ Triggers | ⏸️ Skips (no change) | Instant sync |
| **Webhook fails** | ❌ Missed | ✅ Catches it | Sync within 15s |
| **Network issue** | ❌ Can't reach | ✅ Works | Reliable fallback |
| **Gitea restart** | ❌ Lost events | ✅ Detects changes | No missed syncs |
| **Manual Git change** | ❌ No trigger | ✅ Detects | Drift detection |

### Observing Both Mechanisms

You can see both in action:

```bash
# Watch for webhook triggers
kubectl logs -n flux-system -l app=notification-controller -f | grep "Handling webhook"

# Watch for polling
kubectl logs -n flux-system -l app=source-controller -f | grep "focom-resources"
```

**What you'll see:**
- Webhook triggers: Immediate reconciliation after Git push
- Polling: Regular checks every 15 seconds (usually finds no changes)

### Adjusting Polling Frequency

If you want to reduce polling frequency (webhook handles most syncs):

```yaml
# focom-resources-gitrepository.yaml
spec:
  interval: 1m  # Reduce from 15s to 1 minute
```

**Recommendation:**
- **Keep 15s** - Fast fallback, minimal resource impact
- **Use 1m** - Slightly lower resource usage, still good fallback
- **Don't disable** - Polling provides essential reliability

### Resource Impact

**With webhook + 15s polling:**
- CPU: Negligible increase
- Memory: ~5-10 MB additional
- Network: Minimal (only Git metadata checks)

**Conclusion:** The overhead is so small that keeping both is always recommended.

## Advanced Configuration

### Custom Webhook Path

```yaml
apiVersion: notification.toolkit.fluxcd.io/v1
kind: Receiver
metadata:
  name: focom-resources
  namespace: flux-system
spec:
  type: generic
  secretRef:
    name: webhook-token
  resources:
    - apiVersion: source.toolkit.fluxcd.io/v1
      kind: GitRepository
      name: focom-resources
      namespace: flux-system
```

### Multiple Receivers

You can create multiple receivers for different repositories:

```bash
# Create receiver for another repo
kubectl apply -f - <<EOF
apiVersion: notification.toolkit.fluxcd.io/v1
kind: Receiver
metadata:
  name: other-repo
  namespace: flux-system
spec:
  type: generic
  secretRef:
    name: webhook-token
  resources:
    - apiVersion: source.toolkit.fluxcd.io/v1
      kind: GitRepository
      name: other-repo
EOF
```

### Webhook with HTTPS

For production, use HTTPS:

1. **Set up Ingress with TLS:**
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: flux-webhook
  namespace: flux-system
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - flux-webhook.yourdomain.com
    secretName: flux-webhook-tls
  rules:
  - host: flux-webhook.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: notification-controller
            port:
              number: 80
```

2. **Configure Git webhook with HTTPS URL**

## Performance Impact

### Latency Comparison

| Scenario | Without Webhook | With Webhook |
|----------|----------------|--------------|
| **Approve Draft** | 0-15s | <1s |
| **Update Resource** | 0-15s | <1s |
| **Delete Resource** | 0-15s | <1s |

### Resource Usage

Webhooks have minimal resource impact:
- **notification-controller:** +10-20 MB memory
- **Network:** Minimal (only webhook HTTP requests)

## Security Considerations

### Token Security

- **Generate strong tokens:** Use the setup script (generates SHA-256 hash)
- **Rotate tokens regularly:** Regenerate and update Git webhooks
- **Store securely:** Tokens are stored in Kubernetes secrets

### Network Security

- **Use HTTPS in production:** Encrypt webhook traffic
- **Restrict access:** Use network policies to limit who can reach notification-controller
- **Validate signatures:** Flux validates webhook signatures using the token

### Example Network Policy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: notification-controller
  namespace: flux-system
spec:
  podSelector:
    matchLabels:
      app: notification-controller
  policyTypes:
  - Ingress
  ingress:
  - from:
    - ipBlock:
        cidr: 172.18.0.200/32  # Your Git server IP
    ports:
    - protocol: TCP
      port: 80
```

## Uninstalling Webhook

To remove webhook support:

```bash
# Delete Receiver
kubectl delete receiver focom-resources -n flux-system

# Delete webhook token secret
kubectl delete secret webhook-token -n flux-system

# Remove webhook from Git server
# (Go to Git webhook settings and delete)

# Flux will fall back to polling (15s interval)
```

## See Also

- [Flux Receiver Documentation](https://fluxcd.io/flux/components/notification/receiver/)
- [Flux Webhook Guide](https://fluxcd.io/flux/guides/webhook-receivers/)
- [README.md](README.md) - Main Flux configuration
- [TESTING_GUIDE.md](TESTING_GUIDE.md) - Testing procedures
