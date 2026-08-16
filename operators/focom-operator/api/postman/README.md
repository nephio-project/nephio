# FOCOM NBI Postman Collection

This directory contains a comprehensive Postman collection for testing and demonstrating the FOCOM North Bound Interface (NBI) REST API.

## Collection Overview

The collection demonstrates the complete workflow for managing FOCOM resources:

1. **Health & Info** - API health checks and information
2. **OCloud Workflow** - Complete lifecycle of OCloud configuration
3. **TemplateInfo Workflow** - Complete lifecycle of TemplateInfo configuration
4. **FocomProvisioningRequest Workflow** - Complete lifecycle of provisioning requests
5. **Revision Management** - Working with resource revisions
6. **Draft Rejection Workflow** - Demonstrating draft rejection and state transitions
7. **Cleanup** - Optional cleanup of created resources

## Prerequisites

- Postman installed (Desktop or Web version)
- FOCOM Operator deployed and accessible
- API endpoint accessible (default: `http://localhost:8080`)

## Importing the Collection

### Method 1: Import from File

1. Open Postman
2. Click **Import** button
3. Select **File** tab
4. Choose `focom-nbi-collection.json`
5. Click **Import**

### Method 2: Import from URL (if hosted)

1. Open Postman
2. Click **Import** button
3. Select **Link** tab
4. Paste the URL to the collection
5. Click **Continue** and **Import**

## Configuration

### Collection Variables

The collection uses variables to store dynamic values:

| Variable | Description | Default Value | Auto-Set |
|----------|-------------|---------------|----------|
| `baseUrl` | API base URL | `http://localhost:8080` | No |
| `oCloudId` | OCloud resource ID | (empty) | Yes |
| `templateInfoId` | TemplateInfo resource ID | (empty) | Yes |
| `fprId` | FocomProvisioningRequest ID | (empty) | Yes |

### Setting the Base URL

If your API is not running on `localhost:8080`, update the `baseUrl` variable:

1. Click on the collection name
2. Go to **Variables** tab
3. Update the `baseUrl` **Current Value**
4. Click **Save**

Common base URLs:
- Local port-forward: `http://localhost:8080`
- NodePort: `http://<node-ip>:<node-port>`
- LoadBalancer: `http://<external-ip>:8080`
- Ingress: `https://focom-nbi.example.com`

## Usage Guide

### Quick Start - Complete Demo

Run the requests in order to see the complete workflow:

1. **Health & Info** → Health - Live
   - Verify API is accessible

2. **1. OCloud Workflow** → Run all requests in order
   - Creates, validates, and approves an OCloud configuration
   - The `oCloudId` is automatically saved for later use

3. **2. TemplateInfo Workflow** → Run all requests in order
   - Creates, validates, and approves a TemplateInfo configuration
   - The `templateInfoId` is automatically saved for later use

4. **3. FocomProvisioningRequest Workflow** → Run all requests in order
   - Creates, validates, and approves a provisioning request
   - Uses the previously created OCloud and TemplateInfo
   - The `fprId` is automatically saved for later use

5. **4. Revision Management** → Explore revision history
   - View all revisions of resources
   - Create new drafts from previous revisions

### Understanding the Workflow

Each resource type (OCloud, TemplateInfo, FocomProvisioningRequest) follows the same lifecycle:

```
1. Create Draft (POST /resource/draft)
   ↓
2. Get Draft (GET /resource/{id}/draft)
   ↓
3. Update Draft (PATCH /resource/{id}/draft) [Optional]
   ↓
4. Validate Draft (POST /resource/{id}/draft/validate)
   ↓
5. Approve Draft (POST /resource/{id}/draft/approve)
   ↓
6. Get Approved Resource (GET /resource/{id})
```

### Draft States

Resources go through these states:

- **DRAFT** - Initial state, can be modified
- **VALIDATED** - Passed validation, ready for approval
- **APPROVED** - Approved and active

State transitions:
- DRAFT → VALIDATED (via `/validate`)
- VALIDATED → DRAFT (via `/reject`)
- VALIDATED → APPROVED (via `/approve`)

### Automatic Variable Capture

The collection automatically captures resource IDs when you create resources:

- Creating an OCloud draft saves `oCloudId`
- Creating a TemplateInfo draft saves `templateInfoId`
- Creating an FPR draft saves `fprId`

These variables are then used in subsequent requests.

## Request Details

### 1. Health & Info

**Health - Live**
- Checks if the API is running
- Always returns 200 OK if service is up

**Health - Ready**
- Checks if the API and storage backend are ready
- Returns 503 if storage is not accessible

**API Info**
- Returns API metadata (name, version, description)

### 2. OCloud Workflow

**2.1 Create OCloud Draft**
- Creates a new OCloud configuration in draft state
- Requires: namespace, name, description, o2imsSecret
- Auto-saves `oCloudId` for subsequent requests

**2.2 Get OCloud Draft**
- Retrieves the current draft
- Shows current state (DRAFT, VALIDATED, or APPROVED)

**2.3 Update OCloud Draft**
- Modifies the draft (only works in DRAFT state)
- Example: Updates description

**2.4 Validate OCloud Draft**
- Validates the draft and changes state to VALIDATED
- Must be in DRAFT state

**2.5 Approve OCloud Draft**
- Approves the draft and creates v1 revision
- Must be in VALIDATED state
- Draft is removed after approval

**2.6 Get OCloud (Approved)**
- Retrieves the approved OCloud configuration
- Shows the latest approved revision

**2.7 List All OClouds**
- Lists all approved OCloud configurations

### 3. TemplateInfo Workflow

Similar to OCloud workflow but for TemplateInfo resources.

**Key Fields:**
- `templateName` - Name of the template
- `templateVersion` - Version identifier
- `templateParameterSchema` - JSON Schema for template parameters

**Example Schema:**
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

### 4. FocomProvisioningRequest Workflow

Creates a provisioning request that references an OCloud and TemplateInfo.

**Key Fields:**
- `oCloudId` - References an approved OCloud
- `oCloudNamespace` - Namespace of the OCloud
- `templateName` - Must match an approved TemplateInfo
- `templateVersion` - Must match an approved TemplateInfo
- `templateParameters` - Must conform to the TemplateInfo schema

**Validation:**
- Checks that referenced OCloud exists
- Checks that referenced TemplateInfo exists
- Validates template parameters against schema

### 5. Revision Management

**Get Revisions**
- Lists all approved revisions (v1, v2, v3, etc.)
- Shows revision history

**Create Draft from Revision**
- Creates a new draft based on a previous revision
- Useful for making changes to approved resources
- Cannot create draft if one already exists

### 6. Draft Rejection Workflow

Demonstrates the reject functionality:

1. Create a draft
2. Validate it (moves to VALIDATED state)
3. Reject it (moves back to DRAFT state)
4. Can now modify or delete the draft

### 7. Cleanup

Optional requests to delete created resources:

- Delete FPR first (has dependencies on OCloud and TemplateInfo)
- Then delete TemplateInfo
- Finally delete OCloud

**Note:** Deleting a resource removes all its revisions and drafts.

## Testing Scenarios

### Scenario 1: Happy Path

Run all requests in folders 1-3 in order to see the complete happy path.

### Scenario 2: Draft Modification

1. Create OCloud Draft
2. Get Draft (verify state is DRAFT)
3. Update Draft (modify description)
4. Get Draft (verify changes)
5. Validate Draft
6. Approve Draft

### Scenario 3: Validation Rejection

1. Create FPR Draft
2. Validate Draft (state → VALIDATED)
3. Reject Draft (state → DRAFT)
4. Update Draft (now possible again)
5. Validate Draft
6. Approve Draft

### Scenario 4: Revision History

1. Create and approve OCloud (creates v1)
2. Create draft from v1
3. Modify and approve (creates v2)
4. Get Revisions (see v1 and v2)
5. Create draft from v1 (rollback scenario)

### Scenario 5: Dependency Validation

1. Try to create FPR with non-existent OCloud ID
   - Should fail with dependency error
2. Try to delete OCloud that has FPRs
   - Should fail with dependency error

## Expected Responses

### Success Responses

- **201 Created** - Resource draft created
- **200 OK** - Request successful
- **202 Accepted** - Deletion request accepted

### Error Responses

- **400 Bad Request** - Invalid request body or parameters
- **404 Not Found** - Resource not found
- **409 Conflict** - Draft already exists or invalid state transition
- **500 Internal Server Error** - Server error

### Example Success Response

```json
{
  "oCloudId": "550e8400-e29b-41d4-a716-446655440000",
  "revisionId": "550e8400-e29b-41d4-a716-446655440001",
  "namespace": "focom-system",
  "name": "demo-ocloud-01",
  "description": "Demo OCloud for testing",
  "o2imsSecret": {
    "secretRef": {
      "name": "o2ims-credentials",
      "namespace": "focom-system"
    }
  },
  "oCloudRevisionState": "DRAFT"
}
```

### Example Error Response

```json
{
  "error": "Resource not found",
  "code": "NOT_FOUND",
  "details": "OCloud draft not found for ID: invalid-id",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Troubleshooting

### Connection Refused

**Problem:** Cannot connect to API

**Solutions:**
1. Verify operator is running: `kubectl get pods -n focom-operator-system`
2. Check port-forward is active: `kubectl port-forward -n focom-operator-system svc/focom-operator-controller-manager-nbi-service 8080:8080`
3. Update `baseUrl` variable if using different endpoint

### 404 Not Found

**Problem:** Resource not found

**Solutions:**
1. Verify resource was created successfully
2. Check that variable was saved (look in Console)
3. Manually set the variable if auto-capture failed

### 409 Conflict - Draft Already Exists

**Problem:** Cannot create draft because one exists

**Solutions:**
1. Delete existing draft first
2. Approve existing draft
3. Use a different resource ID

### 400 Bad Request - Invalid State

**Problem:** Cannot perform operation in current state

**Solutions:**
1. Check current state with GET request
2. Ensure correct state transition:
   - Update only works in DRAFT state
   - Validate only works in DRAFT state
   - Approve only works in VALIDATED state
   - Reject only works in VALIDATED state

### Dependency Errors

**Problem:** Referenced resource doesn't exist

**Solutions:**
1. Create OCloud before creating FPR
2. Create TemplateInfo before creating FPR
3. Verify OCloud ID and TemplateInfo name/version match

## Advanced Usage

### Running Collection with Newman

You can run the collection from command line using Newman:

```bash
# Install Newman
npm install -g newman

# Run collection
newman run focom-nbi-collection.json \
  --env-var "baseUrl=http://localhost:8080"

# Run with detailed output
newman run focom-nbi-collection.json \
  --env-var "baseUrl=http://localhost:8080" \
  --reporters cli,json \
  --reporter-json-export results.json
```

### Environment Files

Create environment files for different deployments:

**local.postman_environment.json**
```json
{
  "name": "Local",
  "values": [
    {"key": "baseUrl", "value": "http://localhost:8080", "enabled": true}
  ]
}
```

**staging.postman_environment.json**
```json
{
  "name": "Staging",
  "values": [
    {"key": "baseUrl", "value": "https://focom-nbi-staging.example.com", "enabled": true}
  ]
}
```

### Automated Testing

Use Postman's test scripts to add assertions:

```javascript
// Example test script
pm.test("Status code is 201", function () {
    pm.response.to.have.status(201);
});

pm.test("Response has oCloudId", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property('oCloudId');
});

pm.test("State is DRAFT", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData.oCloudRevisionState).to.eql('DRAFT');
});
```

## Additional Resources

- [OpenAPI Specification](../openapi/focom-nbi-api.yaml) - Complete API reference
- [Deployment Guide](../../docs/DEPLOYMENT.md) - How to deploy the operator
- [Testing Guide](../../docs/TESTING.md) - Comprehensive testing documentation

## Support

For issues with the Postman collection:
1. Verify API is accessible with curl
2. Check Postman console for detailed error messages
3. Review operator logs for server-side errors
4. Consult the OpenAPI specification for correct request format
