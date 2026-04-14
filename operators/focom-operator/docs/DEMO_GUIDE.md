# FOCOM Operator Demo Guide

This guide will help you deploy the FOCOM Operator and demonstrate its REST API functionality using Postman.

## 📦 What You'll Need

- Kubernetes cluster (minikube, kind, or any K8s cluster)
- `kubectl` configured to access your cluster
- Docker installed
- Postman installed
- 15-20 minutes

## 🎯 What You'll Demonstrate

By the end of this demo, you'll have:

1. ✅ Deployed FOCOM Operator to Kubernetes
2. ✅ Created an OCloud configuration (draft → validate → approve)
3. ✅ Created a TemplateInfo configuration (draft → validate → approve)
4. ✅ Created a FocomProvisioningRequest (draft → validate → approve)
5. ✅ Demonstrated the complete REST API workflow

## 📋 Step-by-Step Demo

### Part 1: Deployment (5 minutes)

#### 1.1 Build the Operator

```bash
cd focom-operator

# Build the binary
make build

# Build Docker image
export IMG=localhost:5000/focom-operator:latest
make docker-build

# Push to registry (if using remote registry)
docker push $IMG
```

#### 1.2 Deploy to Kubernetes

```bash
# Deploy CRDs and operator
make deploy IMG=$IMG

# Verify deployment
kubectl get pods -n focom-operator-system

# Expected output:
# NAME                                                READY   STATUS    RESTARTS   AGE
# focom-operator-controller-manager-xxxxxxxxx-xxxxx   1/1     Running   0          30s
```

#### 1.3 Expose the API

```bash
# Port forward the NBI service
kubectl port-forward -n focom-operator-system \
  svc/focom-operator-controller-manager-nbi-service 8080:8080

# Keep this terminal open!
```

#### 1.4 Verify API is Accessible

Open a new terminal:

```bash
# Test health endpoint
curl http://localhost:8080/health/live

# Expected response:
# {"service":"focom-nbi","status":"ok","timestamp":"..."}
```

### Part 2: Postman Setup (2 minutes)

#### 2.1 Import the Collection

1. Open Postman
2. Click **Import** button (top left)
3. Click **Upload Files**
4. Navigate to `focom-operator/api/postman/`
5. Select `focom-nbi-collection.json`
6. Click **Import**

#### 2.2 Verify Configuration

1. Click on the collection name "FOCOM NBI API - Complete Demo"
2. Go to **Variables** tab
3. Verify `baseUrl` is set to `http://localhost:8080`
4. If different, update the **Current Value**
5. Click **Save**

### Part 3: API Demo (10 minutes)

#### 3.1 Health Check (30 seconds)

**Folder:** Health & Info

1. Click **Health - Live**
2. Click **Send**
3. **Show:** Status 200 OK, service is running

**Demo Point:** "The API is accessible and the operator is healthy"

#### 3.2 OCloud Workflow (3 minutes)

**Folder:** 1. OCloud Workflow

**Demo Point:** "OCloud represents an O-Cloud configuration with O2IMS credentials"

**3.2.1 Create Draft**
1. Click **1.1 Create OCloud Draft**
2. **Show the request body:**
   ```json
   {
     "namespace": "focom-system",
     "name": "demo-ocloud-01",
     "description": "Demo OCloud for testing",
     "o2imsSecret": {
       "secretRef": {
         "name": "o2ims-credentials",
         "namespace": "focom-system"
       }
     }
   }
   ```
3. Click **Send**
4. **Show response:** Status 201, `oCloudId` is generated, state is `DRAFT`
5. **Point out:** The ID is automatically saved for later use

**3.2.2 Get Draft**
1. Click **1.2 Get OCloud Draft**
2. **Show:** URL uses `{{oCloudId}}` variable
3. Click **Send**
4. **Show:** Same draft data, state is still `DRAFT`

**3.2.3 Update Draft**
1. Click **1.3 Update OCloud Draft**
2. **Show:** We're updating the description
3. Click **Send**
4. **Show:** Description is updated, state is still `DRAFT`

**3.2.4 Validate Draft**
1. Click **1.4 Validate OCloud Draft**
2. Click **Send**
3. **Show:** State changed to `VALIDATED`
4. **Demo Point:** "Validation checks the configuration is correct"

**3.2.5 Approve Draft**
1. Click **1.5 Approve OCloud Draft**
2. Click **Send**
3. **Show:** State changed to `APPROVED`, `revisionId` is `v1`
4. **Demo Point:** "Approval creates the first revision and makes it active"

**3.2.6 Get Approved Resource**
1. Click **1.6 Get OCloud (Approved)**
2. Click **Send**
3. **Show:** The approved configuration

**3.2.7 List All**
1. Click **1.7 List All OClouds**
2. Click **Send**
3. **Show:** Array with our OCloud

#### 3.3 TemplateInfo Workflow (3 minutes)

**Folder:** 2. TemplateInfo Workflow

**Demo Point:** "TemplateInfo defines a template with its parameter schema"

**3.3.1 Create Draft**
1. Click **2.1 Create TemplateInfo Draft**
2. **Show the request body:**
   - `templateName` and `templateVersion` identify the template
   - `templateParameterSchema` is a JSON Schema
3. **Show the schema:**
   ```json
   {
     "type": "object",
     "properties": {
       "cpu": {"type": "string"},
       "memory": {"type": "string"},
       "replicas": {"type": "integer"},
       "storage": {"type": "string"}
     },
     "required": ["cpu", "memory", "replicas"]
   }
   ```
4. Click **Send**
5. **Show:** `templateInfoId` is generated and saved

**3.3.2 Validate and Approve**
1. Click **2.3 Validate TemplateInfo Draft**
2. Click **Send**
3. **Show:** State is `VALIDATED`
4. Click **2.4 Approve TemplateInfo Draft**
5. Click **Send**
6. **Show:** State is `APPROVED`, revision is `v1`

**3.3.3 List All**
1. Click **2.5 List All TemplateInfos**
2. Click **Send**
3. **Show:** Our template is listed

#### 3.4 FocomProvisioningRequest Workflow (3 minutes)

**Folder:** 3. FocomProvisioningRequest Workflow

**Demo Point:** "FPR creates a provisioning request that references OCloud and TemplateInfo"

**3.4.1 Create Draft**
1. Click **3.1 Create FPR Draft**
2. **Show the request body:**
   - `oCloudId` references our OCloud (using variable)
   - `templateName` and `templateVersion` reference our TemplateInfo
   - `templateParameters` must match the schema
3. **Show the parameters:**
   ```json
   {
     "cpu": "4",
     "memory": "8Gi",
     "replicas": 3,
     "storage": "100Gi"
   }
   ```
4. Click **Send**
5. **Show:** `provisioningRequestId` is generated
6. **Demo Point:** "The API validates that referenced resources exist"

**3.4.2 Update Parameters**
1. Click **3.3 Update FPR Draft**
2. **Show:** We're increasing resources
3. Click **Send**
4. **Show:** Parameters are updated

**3.4.3 Validate and Approve**
1. Click **3.4 Validate FPR Draft**
2. Click **Send**
3. **Show:** State is `VALIDATED`
4. **Demo Point:** "Validation checks parameters against the template schema"
5. Click **3.5 Approve FPR Draft**
6. Click **Send**
7. **Show:** State is `APPROVED`, revision is `v1`

**3.4.4 Get and List**
1. Click **3.6 Get FPR (Approved)**
2. Click **Send**
3. **Show:** The approved provisioning request
4. Click **3.7 List All FPRs**
5. Click **Send**
6. **Show:** All provisioning requests

#### 3.5 Bonus: Revision Management (Optional, 2 minutes)

**Folder:** 4. Revision Management

**Demo Point:** "Resources maintain revision history"

1. Click **4.1 Get FPR Revisions**
2. Click **Send**
3. **Show:** Array with v1 revision
4. Click **4.2 Create Draft from Revision**
5. Click **Send**
6. **Show:** New draft created from v1
7. **Demo Point:** "You can create new drafts from any previous revision"

#### 3.6 Bonus: Draft Rejection (Optional, 2 minutes)

**Folder:** 5. Draft Rejection Workflow

**Demo Point:** "Validated drafts can be rejected back to draft state"

1. Click **5.1 Create New FPR Draft**
2. Click **Send**
3. Click **5.2 Validate Draft**
4. Click **Send**
5. **Show:** State is `VALIDATED`
6. Click **5.3 Reject Draft**
7. Click **Send**
8. **Show:** State is back to `DRAFT`
9. **Demo Point:** "Rejection allows you to make changes after validation"

### Part 4: Verification (Optional, 2 minutes)

#### 4.1 Check Kubernetes Resources

```bash
# View CRDs
kubectl get crds | grep focom

# View custom resources (if using InMemory, these won't exist)
kubectl get oclouds -A
kubectl get templateinfos -A
kubectl get focomprovisioningrequests -A
```

#### 4.2 Check Operator Logs

```bash
# View operator logs
kubectl logs -n focom-operator-system \
  -l control-plane=controller-manager \
  --tail=50
```

#### 4.3 Check Porch Storage (if configured)

```bash
# List PackageRevisions in Porch
kubectl get packagerevisions -n porch-system

# Check Git repository
# Navigate to your Git repository and show the stored packages
```

## 🎤 Demo Script

Here's a suggested script for presenting:

### Introduction (1 minute)

> "Today I'll demonstrate the FOCOM Operator's REST API. FOCOM manages O-Cloud provisioning requests through a North Bound Interface. The API follows a draft-validate-approve workflow for all resources."

### OCloud Demo (3 minutes)

> "First, let's create an OCloud configuration. This represents an O-Cloud with its O2IMS credentials."
> 
> [Create draft]
> "We create a draft - notice it gets a unique ID and starts in DRAFT state."
> 
> [Update draft]
> "We can modify the draft as needed."
> 
> [Validate]
> "Validation checks the configuration and moves it to VALIDATED state."
> 
> [Approve]
> "Approval creates version 1 and makes it active. The draft is removed."

### TemplateInfo Demo (2 minutes)

> "Next, we define a template. TemplateInfo includes a JSON Schema that defines what parameters are allowed."
> 
> [Show schema]
> "This schema requires cpu, memory, and replicas parameters."
> 
> [Create, validate, approve]
> "Same workflow - draft, validate, approve."

### FPR Demo (3 minutes)

> "Now we create a provisioning request. It references the OCloud and TemplateInfo we just created."
> 
> [Show references]
> "Notice it uses the OCloud ID and template name/version."
> 
> [Show parameters]
> "The parameters must match the template schema."
> 
> [Create, validate, approve]
> "The API validates that referenced resources exist and parameters match the schema."

### Conclusion (1 minute)

> "We've demonstrated the complete workflow:
> - Created an OCloud configuration
> - Defined a template with parameter schema
> - Created a provisioning request with validation
> 
> All resources follow the same draft-validate-approve pattern, ensuring controlled changes with validation at each step."

## 🎯 Key Points to Emphasize

1. **Draft-Validate-Approve Workflow**
   - All resources follow the same pattern
   - Drafts can be modified
   - Validation ensures correctness
   - Approval creates immutable revisions

2. **Dependency Validation**
   - FPR validates OCloud exists
   - FPR validates TemplateInfo exists
   - Parameters validated against schema

3. **Revision Management**
   - Each approval creates a new revision (v1, v2, v3...)
   - Can create drafts from any revision
   - Maintains complete history

4. **State Transitions**
   - DRAFT → VALIDATED (via validate)
   - VALIDATED → APPROVED (via approve)
   - VALIDATED → DRAFT (via reject)

5. **RESTful API**
   - Standard HTTP methods (GET, POST, PATCH, DELETE)
   - JSON request/response
   - Proper status codes
   - Clear error messages

## 🐛 Troubleshooting During Demo

### API Not Accessible

```bash
# Check port-forward is running
ps aux | grep port-forward

# Restart if needed
kubectl port-forward -n focom-operator-system \
  svc/focom-operator-controller-manager-nbi-service 8080:8080
```

### Request Fails

1. Check the error message in Postman
2. Show operator logs:
   ```bash
   kubectl logs -n focom-operator-system -l control-plane=controller-manager --tail=20
   ```
3. Verify previous steps completed successfully

### Variable Not Set

1. Show Postman Console (View → Show Postman Console)
2. Manually set the variable:
   - Click collection → Variables tab
   - Set Current Value
   - Save

## 📚 Additional Resources

- **Quick Start Guide**: `docs/QUICK_START.md`
- **Deployment Guide**: `docs/DEPLOYMENT.md`
- **Postman Guide**: `api/postman/README.md`
- **OpenAPI Spec**: `api/openapi/focom-nbi-api.yaml`
- **Test Script**: `scripts/test-api.sh`

## 🎉 Demo Checklist

Before the demo:
- [ ] Kubernetes cluster is accessible
- [ ] Operator is deployed and running
- [ ] Port-forward is active
- [ ] Postman collection is imported
- [ ] Health endpoint returns 200 OK
- [ ] Practiced the workflow once

During the demo:
- [ ] Explain the workflow concept
- [ ] Show request bodies
- [ ] Point out state transitions
- [ ] Highlight validation
- [ ] Show dependency checking
- [ ] Demonstrate revision management

After the demo:
- [ ] Answer questions
- [ ] Share documentation links
- [ ] Offer to show Porch storage (if configured)
- [ ] Demonstrate cleanup (optional)

## 🧹 Cleanup After Demo

```bash
# Stop port-forward (Ctrl+C in the terminal)

# Undeploy operator
make undeploy

# Remove CRDs (WARNING: Deletes all resources)
make uninstall

# Or delete entire namespace
kubectl delete namespace focom-operator-system
```

---

**Ready to demo?** Follow the steps above and you'll have a successful demonstration!

**Need practice?** Run through the workflow 2-3 times before the actual demo.

**Questions?** Check the troubleshooting section or review the documentation.
