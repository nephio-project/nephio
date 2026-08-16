# FOCOM Operator User Guide

This guide explains how to use the FOCOM NBI API to manage O-Cloud infrastructure and deploy clusters.

---

## Understanding FOCOM Resources

FOCOM manages three types of resources:

| Resource | Purpose | Created By | Frequency |
|----------|---------|------------|-----------|
| **OCloud** | FOCOM's knowledge about an O-Cloud (endpoint, credentials) | Administrator | Once per O-Cloud |
| **TemplateInfo** | Cached metadata about MIT templates available on O-Cloud | Administrator | Per template discovered |
| **FocomProvisioningRequest** | Requests cluster deployment | User | Each deployment |

### The Relationship

```
OCloud + TemplateInfo → FocomProvisioningRequest → Cluster Deployment
(O-CLOUD INFO) (TEMPLATE CACHE)  (DEPLOY REQUEST)      (RESULT)
```

---

## Administrator Setup

### Step 1: Register O-Clouds

Register each O-Cloud infrastructure where you want to deploy clusters.

#### 1.1 Create O2IMS Credentials Secret

First, create a Kubernetes Secret containing the O2IMS endpoint and credentials:

```bash
kubectl create secret generic edge-cloud-west-credentials \
  --from-literal=endpoint=https://ocloud-west-ims.example.com \
  --from-literal=token=your-bearer-token-here \
  -n focom-system
```

**Secret Format:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: edge-cloud-west-credentials
  namespace: focom-system
type: Opaque
stringData:
  endpoint: "https://ocloud-west-ims.example.com"
  token: "bearer-token-here"
  # OR for basic auth:
  # username: "admin"
  # password: "password"
```

#### 1.2 Create OCloud via API

```bash
# Create draft
curl -X POST http://localhost:8080/api/v1/o-clouds/draft \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "focom-system",
    "name": "edge-cloud-west",
    "o2imsSecret": {
      "secretRef": {
        "name": "edge-cloud-west-credentials",
        "namespace": "focom-system"
      }
    }
  }'

# Validate
curl -X POST http://localhost:8080/api/v1/o-clouds/edge-cloud-west/draft/validate

# Approve
curl -X POST http://localhost:8080/api/v1/o-clouds/edge-cloud-west/draft/approve
```

#### 1.3 Verify OCloud

```bash
# List all OClouds
curl http://localhost:8080/api/v1/o-clouds

# Get specific OCloud
curl http://localhost:8080/api/v1/o-clouds/edge-cloud-west
```

### Step 2: Create Cluster Templates

Create templates that define different types of clusters you want to deploy.

```bash
# Create draft
curl -X POST http://localhost:8080/api/v1/template-infos/draft \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "focom-system",
    "name": "edge-cluster-template",
    "version": "1.0.0",
    "templateParameters": {
      "clusterType": "edge",
      "defaultWorkerNodes": 3,
      "features": ["SRIOV", "DPDK"]
    }
  }'

# Validate
curl -X POST http://localhost:8080/api/v1/template-infos/edge-cluster-template/draft/validate

# Approve
curl -X POST http://localhost:8080/api/v1/template-infos/edge-cluster-template/draft/approve
```

#### Verify Template

```bash
# List all templates
curl http://localhost:8080/api/v1/template-infos

# Get specific template
curl http://localhost:8080/api/v1/template-infos/edge-cluster-template
```

---

## User Operations

### Deploy a Cluster

Once OClouds and TemplateInfos are set up, users can request cluster deployments.

#### Step 1: Create Provisioning Request

```bash
curl -X POST http://localhost:8080/api/v1/focom-provisioning-requests/draft \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "focom-system",
    "name": "deploy-edge-cluster-001",
    "ocloudRef": {
      "name": "edge-cloud-west"
    },
    "templateInfoRef": {
      "name": "edge-cluster-template"
    },
    "userCredentials": {
      "username": "admin",
      "roleBindings": [
        {
          "role": "cluster-admin",
          "namespace": "*"
        }
      ]
    },
    "capabilityParameters": {
      "clusterType": "edge",
      "highAvailability": true,
      "networkFunctions": ["5G-CU", "5G-DU"],
      "specialFeatures": ["SRIOV", "DPDK"]
    },
    "capacityParameters": {
      "workerNodes": 3,
      "controlPlaneNodes": 3,
      "compute": {
        "cpuCores": 64,
        "memoryGB": 256
      },
      "storage": {
        "persistentVolumeGB": 1000
      },
      "network": {
        "bandwidth": "10Gbps"
      }
    }
  }'
```

#### Step 2: Validate Request

```bash
curl -X POST http://localhost:8080/api/v1/focom-provisioning-requests/deploy-edge-cluster-001/draft/validate
```

The validation checks:
- ✅ Referenced OCloud exists
- ✅ Referenced TemplateInfo exists
- ✅ User credentials are well-formed
- ✅ Capability parameters are valid
- ✅ Capacity parameters are valid

#### Step 3: Approve Request

```bash
curl -X POST http://localhost:8080/api/v1/focom-provisioning-requests/deploy-edge-cluster-001/draft/approve
```

Once approved:
1. FOCOM stores the request in Git via Porch
2. ConfigSync/Flux syncs the CR to the cluster
3. SBI team's component picks up the CR
4. SBI calls O2IMS to provision the cluster
5. O2IMS deploys the actual Kubernetes cluster

#### Step 4: Monitor Status

```bash
# Get provisioning request status
curl http://localhost:8080/api/v1/focom-provisioning-requests/deploy-edge-cluster-001

# Check Kubernetes CR status
kubectl get focomprovisioningrequest deploy-edge-cluster-001 -n focom-system -o yaml
```

Look for status conditions:
```yaml
status:
  conditions:
    - type: Validated
      status: "True"
      reason: "AllChecksPass"
    - type: ReadyForProvisioning
      status: "True"
      reason: "SBICanProceed"
  phase: "Validated"
```

---

## Common Workflows

### Deploy Multiple Clusters to Same O-Cloud

```bash
# All use the same OCloud and Template
curl -X POST .../focom-provisioning-requests/draft -d '{"name": "cluster-001", ...}'
curl -X POST .../focom-provisioning-requests/draft -d '{"name": "cluster-002", ...}'
curl -X POST .../focom-provisioning-requests/draft -d '{"name": "cluster-003", ...}'
```

### Deploy to Multiple O-Clouds

```bash
# Create requests for different O-Clouds
curl -X POST .../focom-provisioning-requests/draft -d '{
  "name": "west-cluster",
  "ocloudRef": {"name": "edge-cloud-west"},
  ...
}'

curl -X POST .../focom-provisioning-requests/draft -d '{
  "name": "east-cluster",
  "ocloudRef": {"name": "edge-cloud-east"},
  ...
}'
```

### Deploy Different Cluster Sizes

```bash
# Small cluster
curl -X POST .../focom-provisioning-requests/draft -d '{
  "name": "small-cluster",
  "templateInfoRef": {"name": "small-edge-template"},
  "capacityParameters": {"workerNodes": 3, ...}
}'

# Large cluster
curl -X POST .../focom-provisioning-requests/draft -d '{
  "name": "large-cluster",
  "templateInfoRef": {"name": "large-edge-template"},
  "capacityParameters": {"workerNodes": 10, ...}
}'
```

---

## Update and Delete Operations

### Update a Provisioning Request (Before Approval)

```bash
# Update draft
curl -X PUT http://localhost:8080/api/v1/focom-provisioning-requests/deploy-edge-cluster-001/draft \
  -H "Content-Type: application/json" \
  -d '{
    "capacityParameters": {
      "workerNodes": 5  # Changed from 3 to 5
    }
  }'

# Re-validate and approve
curl -X POST .../deploy-edge-cluster-001/draft/validate
curl -X POST .../deploy-edge-cluster-001/draft/approve
```

### Delete a Provisioning Request

```bash
curl -X DELETE http://localhost:8080/api/v1/focom-provisioning-requests/deploy-edge-cluster-001
```

**Note:** This deletes the FOCOM request. The actual cluster deletion is handled by O2IMS.

---

## Troubleshooting

### Validation Fails

**Problem:** Validation returns an error

**Check:**
```bash
# Verify OCloud exists
curl http://localhost:8080/api/v1/o-clouds/edge-cloud-west

# Verify TemplateInfo exists
curl http://localhost:8080/api/v1/template-infos/edge-cluster-template

# Check validation error message
curl -X POST .../draft/validate
```

### Request Not Picked Up by SBI

**Problem:** FPR approved but cluster not deploying

**Check:**
```bash
# Check CR status
kubectl get focomprovisioningrequest deploy-edge-cluster-001 -n focom-system -o yaml

# Check if CR has ReadyForProvisioning condition
kubectl get focomprovisioningrequest deploy-edge-cluster-001 -n focom-system \
  -o jsonpath='{.status.conditions[?(@.type=="ReadyForProvisioning")].status}'

# Check SBI team's component logs
kubectl logs -n sbi-system -l app=sbi-controller
```

### O2IMS Credentials Invalid

**Problem:** SBI can't connect to O2IMS

**Check:**
```bash
# Verify secret exists
kubectl get secret edge-cloud-west-credentials -n focom-system

# Check secret contents
kubectl get secret edge-cloud-west-credentials -n focom-system -o yaml

# Verify endpoint and token are correct
kubectl get secret edge-cloud-west-credentials -n focom-system \
  -o jsonpath='{.data.endpoint}' | base64 -d
```

---

## Best Practices

### 1. Naming Conventions

- **OClouds:** Use location-based names (e.g., `edge-cloud-west`, `core-cloud-us-east`)
- **TemplateInfos:** Use descriptive names (e.g., `small-edge-template`, `large-core-template`)
- **FPRs:** Use deployment-specific names (e.g., `deploy-edge-cluster-001`, `prod-cluster-west`)

### 2. Resource Organization

- Keep all resources in the same namespace (e.g., `focom-system`)
- Use labels for grouping related resources
- Document template parameters clearly

### 3. Security

- Store O2IMS credentials in Kubernetes Secrets
- Use RBAC to control who can create/approve requests
- Rotate credentials regularly

### 4. Validation

- Always validate before approving
- Review validation errors carefully
- Test with small deployments first

---

## API Reference

For complete API documentation, see:
- [OpenAPI Specification](../api/openapi/focom-nbi-api.yaml)
- [Postman Collection](../api/postman/)

---

## Next Steps

- [Architecture Documentation](ARCHITECTURE.md) - Understand the system design
- [Deployment Guide](DEPLOYMENT.md) - Deploy the FOCOM operator
- [Testing Guide](TESTING.md) - Test your setup

