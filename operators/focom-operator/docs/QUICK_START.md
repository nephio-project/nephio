# FOCOM Operator Quick Start Guide

This guide will get you up and running with the FOCOM Operator in under 10 minutes.

## Prerequisites

- Kubernetes cluster (minikube, kind, or any K8s cluster)
- `kubectl` configured
- Docker installed
- `make` utility

## Step 1: Build and Deploy (5 minutes)

### Option A: Use Pre-built Image (Recommended)

```bash
# Clone the repository (if not already done)
cd focom-operator

# Set up Docker registry authentication for private images
make docker-auth REGISTRY_USER=your-username REGISTRY_PASSWORD=your-token

# Deploy with pre-built image
make deploy IMG=ghcr.io/your-org/focom-operator:latest IMAGE_PULL_SECRET=ghcr-secret
```

### Option B: Build Your Own Image

```bash
# Build the operator binary
make build

# Build Docker image (use your registry or localhost:5000 for local testing)
export IMG=localhost:5000/focom-operator:latest
make docker-build

# If using local registry, push the image
docker push $IMG

# Deploy to Kubernetes
make deploy IMG=$IMG
```

## Step 2: Verify Deployment (1 minute)

```bash
# Check if operator is running
kubectl get pods -n focom-operator-system

# You should see:
# NAME                                                READY   STATUS    RESTARTS   AGE
# focom-operator-controller-manager-xxxxxxxxx-xxxxx   1/1     Running   0          30s

# Check the service
kubectl get svc -n focom-operator-system

# You should see the NBI service on port 8080
```

## Step 3: Access the API (1 minute)

```bash
# Port forward the NBI service
kubectl port-forward -n focom-operator-system \
  svc/focom-operator-controller-manager-nbi-service 8080:8080
```

Keep this terminal open. The API is now available at `http://localhost:8080`

## Step 4: Test with Postman (3 minutes)

### Import the Collection

1. Open Postman
2. Click **Import**
3. Select `focom-operator/api/postman/focom-nbi-collection.json`
4. Click **Import**

### Run the Demo

1. Open the imported collection
2. Expand **Health & Info** folder
3. Click **Health - Live** and click **Send**
   - You should get a 200 OK response

4. Expand **1. OCloud Workflow** folder
5. Run each request in order:
   - **1.1 Create OCloud Draft** - Creates a draft
   - **1.2 Get OCloud Draft** - Retrieves the draft
   - **1.3 Update OCloud Draft** - Modifies the draft
   - **1.4 Validate OCloud Draft** - Validates the draft
   - **1.5 Approve OCloud Draft** - Approves the draft
   - **1.6 Get OCloud (Approved)** - Gets the approved resource
   - **1.7 List All OClouds** - Lists all OClouds

6. Repeat for **2. TemplateInfo Workflow**

7. Repeat for **3. FocomProvisioningRequest Workflow**

## What You Just Did

You successfully:

1. ✅ Deployed the FOCOM Operator to Kubernetes
2. ✅ Exposed the REST API
3. ✅ Created an OCloud configuration (draft → validate → approve)
4. ✅ Created a TemplateInfo configuration (draft → validate → approve)
5. ✅ Created a FocomProvisioningRequest (draft → validate → approve)

## Understanding the Workflow

Each resource follows this lifecycle:

```
CREATE DRAFT → UPDATE (optional) → VALIDATE → APPROVE
```

States:
- **DRAFT** - Can be modified
- **VALIDATED** - Ready for approval
- **APPROVED** - Active and immutable

## Next Steps

### Explore More Features

1. **Revision Management**
   - Run requests in **4. Revision Management** folder
   - See how to create drafts from previous revisions

2. **Draft Rejection**
   - Run requests in **5. Draft Rejection Workflow** folder
   - Learn how to reject validated drafts

3. **Cleanup**
   - Run requests in **6. Cleanup** folder
   - Delete created resources

### Try with Porch Storage

For production use with Git-backed persistence:

1. Follow [Porch Setup Guide](./porch-setup.md)
2. Configure operator to use Porch storage
3. See your resources stored in Git!

### Explore the API

- **OpenAPI Spec**: `focom-operator/api/openapi/focom-nbi-api.yaml`
- **Postman Collection**: `focom-operator/api/postman/focom-nbi-collection.json`
- **Postman Guide**: `focom-operator/api/postman/README.md`

## Common Commands

### Check Operator Logs

```bash
kubectl logs -n focom-operator-system \
  -l control-plane=controller-manager \
  -f
```

### Restart Operator

```bash
kubectl rollout restart deployment \
  focom-operator-controller-manager \
  -n focom-operator-system
```

### Check Storage Backend

```bash
# View operator configuration
kubectl get deployment \
  focom-operator-controller-manager \
  -n focom-operator-system \
  -o yaml | grep -A 10 env
```

### Access API from Pod

```bash
# Run a test pod
kubectl run -it --rm debug \
  --image=curlimages/curl \
  --restart=Never -- \
  curl http://focom-operator-controller-manager-nbi-service.focom-operator-system.svc.cluster.local:8080/health/live
```

## Troubleshooting

### Operator Not Starting

```bash
# Check pod status
kubectl describe pod -n focom-operator-system \
  -l control-plane=controller-manager

# Common issues:
# - Image pull error: Check IMG variable
# - CrashLoopBackOff: Check logs for errors
```

### API Not Accessible

```bash
# Verify service exists
kubectl get svc -n focom-operator-system

# Verify port-forward is running
# Re-run: kubectl port-forward -n focom-operator-system \
#   svc/focom-operator-controller-manager-nbi-service 8080:8080
```

### Postman Connection Error

1. Verify port-forward is running
2. Check `baseUrl` variable is set to `http://localhost:8080`
3. Try curl: `curl http://localhost:8080/health/live`

## Clean Up

When you're done testing:

```bash
# Stop port-forward (Ctrl+C in the terminal)

# Undeploy operator
make undeploy

# Remove CRDs (WARNING: Deletes all custom resources)
make uninstall

# Or delete the entire namespace
kubectl delete namespace focom-operator-system
```

## What's Next?

### For Development

- [Testing Guide](./TESTING.md) - Run unit and integration tests
- [Deployment Guide](./DEPLOYMENT.md) - Advanced deployment options
- [Porch Setup](./porch-setup.md) - Set up Git-backed storage

### For Production

1. Set up Porch storage backend
2. Configure proper RBAC
3. Set up monitoring and logging
4. Configure ingress for external access
5. Set up TLS/SSL certificates

## Getting Help

- **Documentation**: Check `focom-operator/docs/` directory
- **Logs**: `kubectl logs -n focom-operator-system -l control-plane=controller-manager`
- **API Reference**: See OpenAPI spec in `api/openapi/focom-nbi-api.yaml`

## Summary

You now have a working FOCOM Operator deployment with:

- ✅ REST API accessible at `http://localhost:8080`
- ✅ InMemory storage backend (for testing)
- ✅ Postman collection ready to use
- ✅ Complete workflow demonstrated

**Time to complete**: ~10 minutes

**Next recommended step**: Explore the Postman collection and try creating your own resources!
