# FOCOM Resource Relationships - Visual Guide

## Conceptual Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         FOCOM Management Cluster                         │
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │  OCloud: edge-cloud-west                                      │    │
│  │  ┌──────────────────────────────────────────────────────────┐  │    │
│  │  │  Spec:                                                     │  │    │
│  │  │    o2imsSecret:                                            │  │    │
│  │  │      secretRef:                                            │  │    │
│  │  │        name: cloud-west-o2ims-creds                            │  │    │
│  │  └──────────────────────────────────────────────────────────┘  │    │
│  │         │                                                        │    │
│  │         │ References                                            │    │
│  │         ▼                                                        │    │
│  │  ┌──────────────────────────────────────────────────────────┐  │    │
│  │  │  Secret: cloud-west-o2ims-creds                              │  │    │
│  │  │    endpoint: https://o2ims.cloud-west.example.com            │  │    │
│  │  │    token: bearer-abc123...                               │  │    │
│  │  └──────────────────────────────────────────────────────────┘  │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │  TemplateInfo: edge-cluster-small                              │    │
│  │  ┌──────────────────────────────────────────────────────────┐  │    │
│  │  │  Spec:                                                     │  │    │
│  │  │    templateName: edge-cluster-small                        │  │    │
│  │  │    templateVersion: "1.0.0"                                │  │    │
│  │  │    templateParameterSchema: {...}                          │  │    │
│  │  └──────────────────────────────────────────────────────────┘  │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │  TemplateInfo: edge-cluster-large                              │    │
│  │  ┌──────────────────────────────────────────────────────────┐  │    │
│  │  │  Spec:                                                     │  │    │
│  │  │    templateName: edge-cluster-large                        │  │    │
│  │  │    templateVersion: "2.0.0"                                │  │    │
│  │  │    templateParameterSchema: {...}                          │  │    │
│  │  └──────────────────────────────────────────────────────────┘  │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │  FocomProvisioningRequest: deploy-cluster-001                  │    │
│  │  ┌──────────────────────────────────────────────────────────┐  │    │
│  │  │  Spec:                                                     │  │    │
│  │  │    oCloudId: edge-cloud-west        ◄─── References OCloud│  │
│  │  │    templateName: edge-cluster-small   ◄─── References Template│ │
│  │  │    templateVersion: "1.0.0"                                │  │    │
│  │  │    templateParameters:                                     │  │    │
│  │  │      cpu: "4"                                              │  │    │
│  │  │      memory: "8Gi"                                         │  │    │
│  │  │      replicas: 3                                           │  │    │
│  │  └──────────────────────────────────────────────────────────┘  │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                           │
└───────────────────────────────────┬───────────────────────────────────────┘
                                    │
                                    │ O2IMS API Call
                                    │ (using endpoint + token from secret)
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Remote O-Cloud: West Data Center                    │
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │  O2IMS API Server                                               │    │
│  │  - Receives cluster creation request                            │    │
│  │  - Uses template: edge-cluster-small v1.0.0                     │    │
│  │  - Applies parameters: cpu=4, memory=8Gi, replicas=3            │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │  Cluster Templates (stored on O-Cloud)                          │    │
│  │  - edge-cluster-small v1.0.0                                    │    │
│  │  - edge-cluster-large v2.0.0                                    │    │
│  │  - core-cluster v1.5.0                                          │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │  Infrastructure Resources                                       │    │
│  │  - Compute nodes                                                │    │
│  │  - Storage                                                      │    │
│  │  - Network                                                      │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                           │
└─────────────────────────────────────────────────────────────────────────┘
```

## Relationship Summary

```
OCloud (edge-cloud-west)
    │
    ├─── Contains: O2IMS endpoint + credentials
    │
    ├─── Associated with: TemplateInfo resources
    │    ├─── edge-cluster-small v1.0.0
    │    ├─── edge-cluster-large v2.0.0
    │    └─── core-cluster v1.5.0
    │
    └─── Referenced by: FocomProvisioningRequest resources
         ├─── deploy-cluster-001 (uses edge-cluster-small)
         ├─── deploy-cluster-002 (uses edge-cluster-large)
         └─── deploy-cluster-003 (uses edge-cluster-small)
```

## Complete Example: Deploy a Cluster to West

### Step 1: Administrator Registers the O-Cloud

```bash
# Create secret with O2IMS credentials
kubectl create secret generic cloud-west-o2ims-creds \
  --from-literal=endpoint=https://o2ims.cloud-west.example.com \
  --from-literal=token=bearer-abc123xyz789 \
  -n focom-system
```

```bash
# Create OCloud resource via REST API
POST /api/v1/o-clouds/draft
Content-Type: application/json

{
  "name": "edge-cloud-cloud-west",
  "namespace": "focom-system",
  "description": "West Edge Data Center",
  "o2imsSecret": {
    "secretRef": {
      "name": "cloud-west-o2ims-creds",
      "namespace": "focom-system"
    }
  }
}

# Validate and approve
POST /api/v1/o-clouds/edge-cloud-west/draft/validate
POST /api/v1/o-clouds/edge-cloud-west/draft/approve

# Optional: Delete draft if you want to start over
# DELETE /api/v1/o-clouds/edge-cloud-west/draft
```

**Result in Kubernetes:**
```yaml
apiVersion: focom.nephio.org/v1alpha1
kind: OCloud
metadata:
  name: edge-cloud-west
  namespace: focom-system
spec:
  o2imsSecret:
    secretRef:
      name: cloud-west-o2ims-creds
      namespace: focom-system
```

### Step 2: Administrator Registers Available Templates

```bash
# Create TemplateInfo for small edge clusters
POST /api/v1/template-infos/draft
Content-Type: application/json

{
  "name": "edge-cluster-small",
  "namespace": "focom-system",
  "templateName": "edge-cluster-small",
  "templateVersion": "1.0.0",
  "templateParameterSchema": {
    "type": "object",
    "properties": {
      "cpu": {
        "type": "string",
        "description": "CPU cores per node",
        "enum": ["2", "4", "8"]
      },
      "memory": {
        "type": "string",
        "description": "Memory per node",
        "enum": ["4Gi", "8Gi", "16Gi"]
      },
      "replicas": {
        "type": "integer",
        "description": "Number of worker nodes",
        "minimum": 1,
        "maximum": 10
      }
    },
    "required": ["cpu", "memory", "replicas"]
  }
}

# Validate and approve
POST /api/v1/template-infos/edge-cluster-small/draft/validate
POST /api/v1/template-infos/edge-cluster-small/draft/approve
```

**Result in Kubernetes:**
```yaml
apiVersion: provisioning.oran.org/v1alpha1
kind: TemplateInfo
metadata:
  name: edge-cluster-small
  namespace: focom-system
spec:
  templateName: edge-cluster-small
  templateVersion: "1.0.0"
  templateParameterSchema: |
    {
      "type": "object",
      "properties": {
        "cpu": {"type": "string", "enum": ["2", "4", "8"]},
        "memory": {"type": "string", "enum": ["4Gi", "8Gi", "16Gi"]},
        "replicas": {"type": "integer", "minimum": 1, "maximum": 10}
      },
      "required": ["cpu", "memory", "replicas"]
    }
```

```bash
# Create TemplateInfo for large edge clusters
POST /api/v1/template-infos/draft
Content-Type: application/json

{
  "name": "edge-cluster-large",
  "namespace": "focom-system",
  "templateName": "edge-cluster-large",
  "templateVersion": "2.0.0",
  "templateParameterSchema": {
    "type": "object",
    "properties": {
      "cpu": {
        "type": "string",
        "enum": ["16", "32", "64"]
      },
      "memory": {
        "type": "string",
        "enum": ["32Gi", "64Gi", "128Gi"]
      },
      "replicas": {
        "type": "integer",
        "minimum": 3,
        "maximum": 50
      },
      "storageClass": {
        "type": "string",
        "enum": ["fast-ssd", "standard"]
      }
    },
    "required": ["cpu", "memory", "replicas", "storageClass"]
  }
}

# Validate and approve
POST /api/v1/template-infos/edge-cluster-large/draft/validate
POST /api/v1/template-infos/edge-cluster-large/draft/approve
```

### Step 3: User Deploys a Cluster

```bash
# Create provisioning request
POST /api/v1/focom-provisioning-requests/draft
Content-Type: application/json

{
  "name": "deploy-cluster-001",
  "namespace": "focom-system",
  "description": "Small edge cluster for 5G RAN workloads",
  "oCloudId": "edge-cloud-west",
  "oCloudNamespace": "focom-system",
  "templateName": "edge-cluster-small",
  "templateVersion": "1.0.0",
  "templateParameters": {
    "cpu": "4",
    "memory": "8Gi",
    "replicas": 3
  }
}

# Validate (checks that OCloud and TemplateInfo exist, validates parameters against schema)
POST /api/v1/focom-provisioning-requests/deploy-cluster-001/draft/validate

# Approve (commits to Git and triggers southbound O2IMS provisioning request)
POST /api/v1/focom-provisioning-requests/deploy-cluster-001/draft/approve
```

**Result in Kubernetes:**
```yaml
apiVersion: focom.nephio.org/v1alpha1
kind: FocomProvisioningRequest
metadata:
  name: deploy-cluster-001
  namespace: focom-system
spec:
  oCloudId: edge-cloud-west
  oCloudNamespace: focom-system
  name: deploy-cluster-001
  description: Small edge cluster for 5G RAN workloads
  templateName: edge-cluster-small
  templateVersion: "1.0.0"
  templateParameters:
    cpu: "4"
    memory: "8Gi"
    replicas: 3
status:
  phase: Provisioning
  message: Cluster creation in progress
  lastUpdated: "2025-01-15T14:30:00Z"
```

### Step 4: FOCOM Processes the Request

**What happens internally:**

1. **Lookup OCloud:**
   - Find OCloud resource "edge-cloud-west"
   - Retrieve secret "cloud-west-o2ims-creds"
   - Extract endpoint: `https://o2ims.cloud-west.example.com`
   - Extract token: `bearer-abc123xyz789`

2. **Lookup TemplateInfo:**
   - Find TemplateInfo "edge-cluster-small" version "1.0.0"
   - Validate parameters against templateParameterSchema
   - Confirm: cpu="4" ✓, memory="8Gi" ✓, replicas=3 ✓

3. **Call O2IMS API:**
   ```http
   POST https://o2ims.cloud-west.example.com/o2ims/v1/clusters
   Authorization: Bearer bearer-abc123xyz789
   Content-Type: application/json

   {
     "templateId": "edge-cluster-small",
     "templateVersion": "1.0.0",
     "parameters": {
       "cpu": "4",
       "memory": "8Gi",
       "replicas": 3
     }
   }
   ```

4. **Track Status:**
   - Update FocomProvisioningRequest status
   - Poll O2IMS for deployment progress
   - Update status: Provisioning → Ready → Active

### Step 5: Multiple Deployments to Same O-Cloud

```bash
# Deploy another cluster using different template
POST /api/v1/focom-provisioning-requests/draft
{
  "name": "deploy-cluster-002",
  "oCloudId": "edge-cloud-west",        # Same O-Cloud
  "templateName": "edge-cluster-large",    # Different template
  "templateVersion": "2.0.0",
  "templateParameters": {
    "cpu": "32",
    "memory": "64Gi",
    "replicas": 5,
    "storageClass": "fast-ssd"
  }
}

# Deploy another small cluster
POST /api/v1/focom-provisioning-requests/draft
{
  "name": "deploy-cluster-003",
  "oCloudId": "edge-cloud-west",        # Same O-Cloud
  "templateName": "edge-cluster-small",    # Same template as 001
  "templateVersion": "1.0.0",
  "templateParameters": {
    "cpu": "8",                            # Different parameters
    "memory": "16Gi",
    "replicas": 5
  }
}
```

**Result:**
```
OCloud: edge-cloud-west
    │
    ├─── TemplateInfo: edge-cluster-small v1.0.0
    │    ├─── Used by: deploy-cluster-001 (cpu=4, memory=8Gi, replicas=3)
    │    └─── Used by: deploy-cluster-003 (cpu=8, memory=16Gi, replicas=5)
    │
    └─── TemplateInfo: edge-cluster-large v2.0.0
         └─── Used by: deploy-cluster-002 (cpu=32, memory=64Gi, replicas=5)
```

### Step 6: Monitor Provisioning Status

```bash
# Get OCloud status (includes O2IMS availability status)
GET /api/v1/o-clouds/edge-cloud-west

# Response includes availability status from O2IMS
{
  "id": "edge-cloud-west",
  "name": "edge-cloud-west",
  "namespace": "focom-system",
  "state": "APPROVED",
  "o2imsAvailability": "available",  # Status from O2IMS
  ...
}

# Get provisioning request status (includes O2IMS provisioning status)
GET /api/v1/focom-provisioning-requests/deploy-cluster-001

# Response includes provisioning status from O2IMS
{
  "id": "deploy-cluster-001",
  "name": "deploy-cluster-001",
  "state": "APPROVED",
  "provisioningStatus": {
    "phase": "Active",              # Status from O2IMS
    "message": "Cluster is running",
    "lastUpdated": "2025-01-15T14:45:00Z"
  },
  ...
}
```

### Step 7: Delete Provisioning Request (Decommission Cluster)

```bash
# Delete provisioning request (triggers southbound O2IMS deletion)
DELETE /api/v1/focom-provisioning-requests/deploy-cluster-001

# Response: 202 Accepted
{
  "message": "FocomProvisioningRequest decommissioning initiated"
}

# This triggers:
# 1. Deletion from Git via Porch
# 2. GitOps sync removes Kubernetes CR
# 3. Southbound O2IMS deletion request to decommission the cluster
```

## Key Relationships Explained

### 1. OCloud → Secret (1:1)
- Each OCloud references exactly one Secret
- Secret contains O2IMS endpoint and credentials
- Secret is reused for all deployments to that O-Cloud

### 2. OCloud → TemplateInfo (1:N)
- One O-Cloud can have many TemplateInfo resources
- TemplateInfo describes what templates are available on that O-Cloud
- Currently implicit association (by naming convention)
- Future: could add explicit `oCloudRef` field to TemplateInfo

### 3. OCloud → FocomProvisioningRequest (1:N)
- One O-Cloud can have many deployment requests
- Each request explicitly references the OCloud via `oCloudId`
- This is how FOCOM knows where to send the deployment request

### 4. TemplateInfo → FocomProvisioningRequest (1:N)
- One template can be used by many deployment requests
- Each request references template via `templateName` + `templateVersion`
- Each request provides custom `templateParameters`

### 5. FocomProvisioningRequest → OCloud + TemplateInfo (N:1:1)
- Each request references exactly one OCloud
- Each request references exactly one TemplateInfo (name + version)
- This is the "glue" that ties everything together

## Summary

**OCloud** = Target Infrastructure (WHERE to deploy)
- Contains: O2IMS API credentials
- Reused for: Multiple templates and deployments

**TemplateInfo** = Template metadata (WHAT can be added to the MIT template)
- Contains: Template name, version, parameter schema
- Reused for: Multiple deployments with different parameters
- Represents a subset of parameters available for the actual MIT template
- This is what the o-cloud needs to know to populate the MIT template.

**FocomProvisioningRequest** = Deployment request (Deploy THIS, HERE, with THESE settings)
- References: One OCloud + One TemplateInfo
- Contains: Custom parameters for this specific deployment
- Creates: One cluster on the target O-Cloud
