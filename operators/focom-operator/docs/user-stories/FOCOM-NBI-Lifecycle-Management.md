# User Story - FOCOM NBI Lifecycle Management

**Feature Overview**

The FOCOM (Federated O-Cloud Cluster Orchestration Management) North Bound Interface (NBI) provides a REST API for managing the complete lifecycle of FOCOM infrastructure resources. This interface enables administrators and orchestrators to declaratively manage O-Cloud configurations, cluster templates, and provisioning requests through a Git-backed, revision-controlled system using Nephio Porch and GitOps synchronization (ConfigSync/Flux).

The FOCOM NBI implements a draft-validate-approve workflow that ensures safe infrastructure-as-code management, with Git serving as the source of truth for FOCOM resources. This approach provides full audit trails, self-healing capabilities, while maintaining clear separation between the northbound management interface and southbound O2IMS integration.

**Stakeholders**

* Infrastructure Administrators - Managing O-Cloud configurations and cluster templates
* Service Orchestrators - Creating and managing cluster provisioning requests  
* Platform Operators - Monitoring and maintaining FOCOM infrastructure
* O-RAN Architecture Teams - Defining standards for FOCOM integration

**Design and architectural considerations**

### Core Architecture Principles

The FOCOM NBI is designed around several key architectural principles:

**1. Git as Source of Truth**
- All FOCOM resources are stored in Git repositories via Nephio Porch
- Git provides immutable revision history and audit trails
- ConfigSync or Flux automatically synchronizes Git state to Kubernetes CRs
- Self-healing: manual changes to CRs are automatically reverted to Git state

**2. Draft-Validate-Approve Workflow**
- Resources progress through lifecycle states: Draft → Validated → Approved
- Draft state allows iterative editing without affecting production
- Validation ensures referential integrity and parameter correctness
- Approval commits resources to Git and triggers deployment

**3. Resource Separation and Cached Orchestration**
- **OCloud**: FOCOM's knowledge about an O-Cloud, including O2IMS endpoint and credentials
- **TemplateInfo**: FOCOM's cached metadata about MIT templates available on a specific O-Cloud
- **FocomProvisioningRequest**: Combines OCloud + TemplateInfo for actual deployment
- OClouds and TemplateInfos represent FOCOM's cached view of O-Cloud capabilities, maintained and updated as the underlying O-Cloud resources change

**4. RESTful API Design**
- Standard HTTP methods (GET, POST, PATCH, DELETE)
- JSON request/response format
- Consistent endpoint structure across resource types
- OpenAPI 3.0 specification for documentation and code generation

**5. Kubernetes-Native Integration**
- Resources stored as Kubernetes Custom Resources (CRDs)
- Rest API access to CRs via Porch PackageRevision synchronization
- Rest API can query state of Kubernetes resources
- CRs are automatically managed by ConfigSync or Flux
- Integrates with existing Kubernetes tooling and workflows - through kubernetes api client querying resources
- Compatible with GitOps tools (ConfigSync, Flux)

### Resource Model

The FOCOM NBI manages three core resource types:

#### OCloud - "FOCOM's Knowledge About an O-Cloud"

O-Clouds represent the underlying platform (hardware, software, resources) that hosts O-Cloud Clusters,
where these clusters are provisioned or updated via Provisioning Request from the FOCOM in the SMO.
The FOCOM interacts with the O-Cloud for infrastructure and cluster management via the O2-IMS interface.
Each O-Cloud can host multiple independent clusters, where each is usually dedicated to different network functions.
A given O-Cloud can potentially be hosted in one or more physical sites, and physical sites can host all or
parts of one or multiple O-Clouds.

**What it contains:**
- A reference to a Kubernetes Secret with O2IMS API endpoint and credentials
- Metadata (name, namespace) to identify this O-Cloud

**What it does NOT contain:**
- No actual infrastructure details (those are queried from O2IMS)
- No template information (those are separate TemplateInfo resources)
- No deployment history (those are FocomProvisioningRequest resources)

**OCloud and TemplateInfo Relationship:**
When you create a FocomProvisioningRequest, you specify:
- Which OCloud to deploy to (by referencing the OCloud name)
- Which template to use (by referencing template name/version)

The OCloud name acts as the "handle" or "identifier" that ties together:
1. The infrastructure endpoint (via o2imsSecret)
2. The templates available on that O-Cloud (TemplateInfo resources)
3. The deployment requests targeting that O-Cloud (FocomProvisioningRequest resources)

**Key Fields:**
- `name`: Human-readable identifier (e.g., "cloud-west")
- `namespace`: Kubernetes namespace
- `o2imsSecret`: Reference to Secret containing O2IMS endpoint URL and authentication token

**Lifecycle:** Created once by administrators, reused for multiple deployments

**Example:**
```yaml
apiVersion: focom.nephio.org/v1alpha1
kind: OCloud
metadata:
  name: cloud-west
  namespace: focom-system
spec:
  o2imsSecret:
    secretRef:
      name: cloud-west-credentials
      namespace: focom-system
```

#### TemplateInfo - "Cached MIT Template Metadata"
Contains FOCOM's cached metadata about MIT templates available on a specific O-Cloud. The actual MIT templates are defined and owned by the O-Cloud.

**Key Fields:**
- `name`: Template identifier
- `templateName`: Name of the underlying template
- `templateVersion`: Version of the template
- `templateParameterSchema`: JSON schema defining valid parameters

**Lifecycle:** Created once by administrators, reused for multiple deployments

**Example:**
```yaml
apiVersion: provisioning.oran.org/v1alpha1
kind: TemplateInfo
metadata:
  name: edge-cluster-template
  namespace: focom-system
spec:
  templateName: edge-cluster-template
  templateVersion: "1.0.0"
  templateParameterSchema: |
    {
      "type": "object",
      "properties": {
        "cpu": {"type": "string"},
        "memory": {"type": "string"},
        "replicas": {"type": "integer"}
      }
    }
```

#### FocomProvisioningRequest - "Deploy This Cluster"
The actual request to deploy a cluster using a specific template on a specific O-Cloud.

**Key Fields:**
- `ocloudRef`: Reference to target OCloud
- `templateInfoRef`: Reference to TemplateInfo
- `templateParameters`: Instance-specific parameter values

**Lifecycle:** Created each time a user wants to deploy a new cluster

**Example:**
```yaml
apiVersion: focom.nephio.org/v1alpha1
kind: FocomProvisioningRequest
metadata:
  name: deploy-edge-cluster-001
  namespace: focom-system
spec:
  ocloudRef:
    name: cloud-west
  templateInfoRef:
    name: edge-cluster-template
  templateParameters:
    cpu: "4"
    memory: "8Gi"
    replicas: 3
```

### Technology Stack

**REST API Layer:**
- Go programming language
- Gin HTTP framework
- OpenAPI 3.0 specification
- JSON request/response format

**Storage Layer:**
- Nephio Porch for Git-backed storage
- Kubernetes PackageRevision CRDs
- Git repositories for persistent storage
- Revision history and audit trails

**Synchronization Layer:**
- ConfigSync (Google Anthos Config Management) - default
- Flux CD - alternative option
- 15-second poll interval (ConfigSync)
- Automatic CR creation and self-healing

**Kubernetes Integration:**
- Custom Resource Definitions (CRDs)
- RBAC for authorization
- Service accounts for authentication
- Namespace-based resource isolation

### Integration with O2IMS and Southbound Interface

While the FOCOM NBI is designed to eventually integrate with O2IMS and southbound interfaces, this user story focuses on the northbound lifecycle management capabilities. The relationship with O2IMS is as follows:

**Current Scope (This User Story):**
- REST API for FOCOM resource lifecycle management
- Git-backed storage and revision control
- Draft-validate-approve workflow
- Kubernetes CR management via GitOps

**Future Integration (Out of Scope):**
- Southbound interface to O2IMS provisioning API
- Cluster deployment via O2IMS
- Infrastructure inventory queries
- Capacity validation against available resources

**Interface Point:**
The FocomProvisioningRequest CR serves as the contract between FOCOM NBI and future southbound components. The `o2imsSecret` field in OCloud provides the endpoint and credentials for O2IMS integration when implemented.


### Porch and GitOps Integration

The FOCOM NBI leverages Nephio Porch for Git-backed storage, providing several key benefits:

**Porch PackageRevision Lifecycle:**
```
Draft → Proposed → Published
  ↓        ↓          ↓
 Edit   Validate   Immutable
```

**Git Repository Structure:**
```
focom-resources.git/
├── demo-ocloud-01-v1/
│   ├── Kptfile
│   └── ocloud.yaml
├── demo-ocloud-01-v2/
│   ├── Kptfile
│   └── ocloud.yaml
├── edge-template-v1/
│   ├── Kptfile
│   └── templateinfo.yaml
└── deployment-01-v1/
    ├── Kptfile
    └── focomprovisioningrequest.yaml
```

**GitOps Synchronization Flow:**
1. API approves draft → Porch commits to Git
2. ConfigSync/Flux polls Git (every 15s)
3. ConfigSync/Flux detects changes
4. ConfigSync/Flux applies CRs to cluster
5. ConfigSync/Flux adds management annotations
6. Self-healing: reverts manual CR changes

**Benefits:**
- **Audit Trail**: Full history in Git commits
- **Disaster Recovery**: Restore from Git repository
- **Multi-Cluster**: Same Git repo syncs to multiple clusters
- **Compliance**: Git provides audit trail for compliance requirements
- **Rollback**: Can revert to any previous revision
- **Self-Healing**: Automatic restoration of deleted/modified CRs


**Assumptions**

The following assumptions are relevant to this use case:

1. **Kubernetes Cluster Available** – A Kubernetes cluster is running and accessible for deploying the FOCOM operator and managing resources.
2. **Porch Installed** – Nephio Porch is installed and configured with at least one Git repository for storing FOCOM resources.
3. **GitOps Tool Installed** – Either ConfigSync or Flux CD is installed and configured to synchronize from the Porch-managed Git repository.
4. **Git Repository Access** – The FOCOM operator has appropriate credentials to access the Git repository via Porch.
5. **Pre-existing upstream repo** - There should be a pre-existing upstream repo for Porch to sync to e.g. "focom-resources".
6. **RBAC Configured** – Kubernetes RBAC is configured to allow the FOCOM operator to manage PackageRevisions and Custom Resources.
7. **Network Connectivity** – The FOCOM NBI API is accessible to clients (administrators, orchestrators, automation tools).

**Prerequisites**

1. **Kubernetes Cluster** – Version 1.11.3+ with kubectl access
2. **Nephio Porch** – Installed and configured with a Git repository
3. **GitOps Synchronization** – ConfigSync or Flux CD installed and configured
4. **Git Repository** – Accessible Git repository for storing FOCOM resources
5. **FOCOM Operator Deployed** – FOCOM operator running in the cluster
6. **CRDs Installed** – OCloud, TemplateInfo, and FocomProvisioningRequest CRDs installed
7. **Service Account** – Service account with appropriate RBAC permissions for Porch and CR management

**Requirements**

1. The FOCOM NBI SHALL provide a REST API for creating, reading, updating, and deleting OCloud configurations.
2. The FOCOM NBI SHALL provide a REST API for creating, reading, updating, and deleting TemplateInfo configurations.
3. The FOCOM NBI SHALL provide a REST API for creating, reading, updating, and deleting FocomProvisioningRequest resources.
4. The FOCOM NBI SHALL implement a draft-validate-approve workflow for all resource types.
5. The FOCOM NBI SHALL store all approved resources in Git via Nephio Porch.
6. The FOCOM NBI SHALL maintain immutable revision history for all resources.
7. The FOCOM NBI SHALL validate referential integrity (e.g., FocomProvisioningRequest references existing OCloud and TemplateInfo).
8. The FOCOM NBI SHALL support creating new drafts from previous revisions.
9. The FOCOM NBI SHALL provide health check endpoints for monitoring.
10. The FOCOM NBI SHALL return standardized error responses with appropriate HTTP status codes.


**Acceptance criteria (Definition of done)**

### General API Behavior

1. The REST API SHALL be accessible on port 8080 with base path `/api/v1`.
2. All API endpoints SHALL accept and return JSON-formatted data.
3. All API endpoints SHALL return appropriate HTTP status codes (200, 201, 400, 404, 409, 500).
4. The API SHALL provide OpenAPI 3.0 specification at `/api/info`.
5. The API SHALL provide health check endpoints at `/health/live` and `/health/ready`.

### OCloud Lifecycle Management

6. The API SHALL support registering a new target O-Cloud IMS with its endpoint and credentials by creating an OCloud draft via `POST /api/v1/o-clouds/draft`.
7. The API SHALL support retrieving the current draft for an OCloud via `GET /api/v1/o-clouds/{id}/draft`.
8. The API SHALL support updating the current draft for an OCloud via `PATCH /api/v1/o-clouds/{id}/draft`.
9. The API SHALL support deleting the current for an OCloud via `DELETE /api/v1/o-clouds/{id}/draft`.
10. The API SHALL support validating the current draft for an OCloud via `POST /api/v1/o-clouds/{id}/draft/validate`.
11. The API SHALL support rejecting validated OCloud drafts via `POST /api/v1/o-clouds/{id}/ draft/reject`.
12. The API SHALL support approving a validated draft for an OCloud via `POST /api/v1/o-clouds/{id}/draft/approve`.
13. The API SHALL support listing the latest approved revisions of all OClouds via `GET /api/v1/o-clouds`.
14. The API SHALL support retrieving the latest approved revision of a specific OCloud via `GET /api/v1/o-clouds/{id}`. The response shall contain the availability status of the target O-Cloud IMS.
15. The API SHALL support deleting the registered information for a specific OCloud via `DELETE /api/v1/o-clouds/{id}`.
16. The API SHALL support listing all previously approved revisions for an OCloud via `GET /api/v1/o-clouds/{id}/revisions`.
17. The API SHALL support retrieving a specific previously approved OCloud revision via `GET /api/v1/o-clouds/{id}/revisions/{revisionId}`.
18. The API SHALL support creating a new OCloud draft from a previously approved OCloud revision via `POST /api/v1/o-clouds/{id}/revisions/{revisionId}/draft`.

### TemplateInfo Lifecycle Management

19. The API SHALL support registering information about a new O-Cloud Managed Infrastructure Template (MIT) by creating a TemplateInfo draft via `POST /api/v1/template-infos/draft`.
20. The API SHALL support retrieving the current TemplateInfo draft for a specific O-Cloud MIT via `GET /api/v1/template-infos/{id}/draft`.
21. The API SHALL support updating the current TemplateInfo draft for a specific O-Cloud MIT via `PATCH /api/v1/template-infos/{id}/draft`.
22. The API SHALL support deleting the current TemplateInfo draft for a specific O-Cloud MIT via `DELETE /api/v1/template-infos/{id}/draft`.
23. The API SHALL support validating the current TemplateInfo draft for a specific O-Cloud MIT via `POST /api/v1/template-infos/{id}/draft/validate`.
24. The API SHALL support rejecting the current validated TemplateInfo draft for a specific O-Cloud MIT via `POST /api/v1/template-infos/{id}/draft/reject`.
25. The API SHALL support approving a validated TemplateInfo draft via `POST /api/v1/template-infos/{id}/draft/approve`.
26. The API SHALL support listing the latest approved revisions of all TemplateInfos via `GET /api/v1/template-infos`.
27. The API SHALL support retrieving the latest approved revision of a specific TemplateInfo via `GET /api/v1/template-infos/{id}`.
28. The API SHALL support deleting the registered TemplateInfo information for a specific O-Cloud MIT via `DELETE /api/v1/template-infos/{id}`.
29. The API SHALL support listing all previously approved revisions for a TemplateInfo via `GET /api/v1/template-infos/{id}/revisions`.
30. The API SHALL support retrieving a specific previously approved TemplateInfo revision via `GET /api/v1/template-infos/{id}/revisions/{revisionId}`.
31. The API SHALL support creating a new TemplateInfo draft from a previously approved TemplateInfo revision via `POST /api/v1/template-infos/{id}/revisions/{revisionId}/draft`.


### FocomProvisioningRequest Lifecycle Management

32. The API SHALL support the definition of a new O2ims Provisioning Request by creating a FocomProvisioningRequest draft via `POST /api/v1/focom-provisioning-requests/draft`.
33. The API SHALL support retrieving the current FocomProvisioningRequest draft for a specific O2ims Provisioning Request via `GET /api/v1/focom-provisioning-requests/{id}/draft`.
34. The API SHALL support updating the current FocomProvisioningRequest draft for a specific O2ims Provisioning Request via `PATCH /api/v1/focom-provisioning-requests/{id}/draft`.
35. The API SHALL support deleting the current FocomProvisioningRequest draft for a specific O2ims Provisioning Request via `DELETE /api/v1/focom-provisioning-requests/{id}/draft`.
36. The API SHALL support validating the current FocomProvisioningRequest draft for a specific O2ims Provisioning Request via `POST /api/v1/focom-provisioning-requests/{id}/draft/validate`.
37. The API SHALL support rejecting the current validated FocomProvisioningRequest draft for a specific O2ims Provisioning Request via `POST /api/v1/focom-provisioning-requests/{id}/draft/reject`.
38. The API SHALL support approving a validated FocomProvisioningRequest draft via `POST /api/v1/focom-provisioning-requests/{id}/draft/approve`. This will trigger FOCOM to issue the southbound O2ims Provisioning Request towards the O-Cloud.
39. The API SHALL support listing the latest approved revisions of all FocomProvisioningRequests via `GET /api/v1/focom-provisioning-requests`.
40. The API SHALL support retrieving the latest approved revision of a specific FocomProvisioningRequest via `GET /api/v1/focom-provisioning-requests/{id}`. The response shall contain the processing status of the corresponding O-Cloud O2ims Provisioning Request.
41. The API SHALL support deleting a FocomProvisioningRequests via `DELETE /api/v1/focom-provisioning-requests/{id}`. This will trigger FOCOM to delete the corresponding southbound O2ims Provisioning Request in the O-Cloud.
42. The API SHALL support listing all previously approved revisions for a FocomProvisioningRequest via `GET /api/v1/focom-provisioning-requests/{id}/revisions`.
43. The API SHALL support retrieving a specific previously approved FocomProvisioningRequest revision via `GET /api/v1/focom-provisioning-requests/{id}/revisions/{revisionId}`.
44. The API SHALL support creating a new FocomProvisioningRequest draft from previously approved FocomProvisioningRequest revision via `POST /api/v1/focom-provisioning-requests/{id}/revisions/{revisionId}/draft`.

### Draft Workflow Behavior

45. Draft resources SHALL be editable via PATCH operations.
46. Validated drafts SHALL NOT be editable until rejected back to Draft state.
47. Approved resources SHALL be immutable and stored as new revisions.
48. The API SHALL return HTTP 409 Conflict when attempting to update a validated draft.
49. The API SHALL return HTTP 400 Bad Request when attempting to approve a non-validated draft.
50. The API SHALL return HTTP 400 Bad Request when attempting to reject a non-validated draft.

### Validation Behavior

51. Validation of FocomProvisioningRequest SHALL verify that the referenced OCloud exists.
52. Validation of FocomProvisioningRequest SHALL verify that the referenced TemplateInfo exists and execute schema validation of the FocomProvisioningRequest templateParameters based on the parameter schema in the TemplateInfo.
53. Validation SHALL return descriptive error messages for validation failures.
54. Successful validation SHALL transition the draft from Draft to Validated state.


### Git Storage and Revision Management

55. Approved resources SHALL be stored in Git via Porch PackageRevisions.
56. Each approval SHALL create a new immutable revision (v1, v2, v3, etc.).
57. Git commits SHALL contain both Kptfile and resource YAML.
58. Revision history SHALL be queryable via the revisions endpoints.
59. Previous revisions SHALL be retrievable and usable as basis for new drafts.

### GitOps Synchronization

60. Approved resources SHALL be synchronized to Kubernetes CRs via ConfigSync or Flux.
61. Synchronized CRs SHALL have ConfigSync/Flux management annotations.
62. Manual changes to CRs SHALL be automatically reverted to Git state (self-healing).
63. Synchronization SHALL occur within 15 seconds of Git commit (ConfigSync default).

### Error Handling

64. The API SHALL return HTTP 404 Not Found for non-existent resources.
65. The API SHALL return HTTP 400 Bad Request for invalid request payloads.
66. The API SHALL return HTTP 500 Internal Server Error for server-side failures.
67. Error responses SHALL include descriptive error messages and error codes.
68. Error responses SHALL include timestamps.

### Resource Deletion

69. Deleting a resource SHALL delete all its revisions from Git.
70. Deleting a resource SHALL trigger CR deletion via GitOps synchronization.
71. The API SHALL return HTTP 202 Accepted for deletion requests.
72. Draft deletion SHALL remove only the draft PackageRevision.

### Performance and Reliability

73. Draft operations SHALL complete within 1 second under normal conditions.
74. Approval operations SHALL complete within 2 seconds under normal conditions.
75. List operations SHALL complete within 1 second for up to 100 resources.
76. The API SHALL handle concurrent requests safely.
77. The API SHALL maintain data consistency across Porch, Git, and Kubernetes.

**Priority**

High - This is a foundational capability for FOCOM infrastructure management.


**What's not in scope**

The following items are explicitly out of scope for this user story:

1. **Southbound O2IMS Integration** – Communication with O2IMS provisioning API for actual cluster deployment
2. **Cluster Deployment** – Actual Kubernetes cluster creation and configuration
3. **Infrastructure Inventory** – Querying O2IMS inventory API for available capacity
4. **Capacity Validation** – Validating requested capacity against available infrastructure resources
5. **Template Parameter Validation** – Deep validation of template parameters against schemas
6. **Multi-Site Deployment** – Coordinating deployments across multiple O-Cloud sites
7. **REST API Authentication** – Authentication for external clients calling the REST API (OAuth, OIDC, API keys, etc.)
8. **REST API Authorization** – Fine-grained authorization for REST API endpoints beyond network-level controls
9. **Webhook Notifications** – Real-time notifications for resource state changes
10. **Metrics and Monitoring** – Detailed metrics collection and monitoring dashboards
11. **Template Management** – Creation and management of underlying cluster templates
12. **Backup and Restore** – Automated backup and restore procedures (Git provides this inherently)

**References**

- OpenAPI Specification: `focom-operator/api/openapi/focom-nbi-api.yaml`
- Postman Collection: `focom-operator/api/postman/focom-nbi-collection.json`
- Architecture Documentation: `focom-operator/docs/ARCHITECTURE.md`
- User Guide: `focom-operator/docs/USER_GUIDE.md`
- Deployment Guide: `focom-operator/docs/DEPLOYMENT.md`
- Nephio Porch Documentation: https://github.com/nephio-project/porch
- ConfigSync Documentation: https://cloud.google.com/anthos-config-management/docs/config-sync-overview
- Flux CD Documentation: https://fluxcd.io/docs/


**Component Architecture**

The FOCOM NBI architecture consists of several integrated components working together to provide Git-backed, revision-controlled infrastructure management.

### High-Level Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     External Clients                             │
│         (curl, UI, Automation Tools, Orchestrators)              │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTP REST (JSON)
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   FOCOM NBI REST API                             │
│                  (Gin HTTP Server :8080)                         │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   OCloud     │  │ TemplateInfo │  │     FPR      │          │
│  │   Handler    │  │   Handler    │  │   Handler    │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                  │                  │                   │
│         └──────────────────┼──────────────────┘                  │
│                            │                                      │
│                            ▼                                      │
│                  ┌──────────────────┐                            │
│                  │ PorchStorage     │                            │
│                  │ Interface        │                            │
│                  └──────────────────┘                            │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTP REST
                             ▼
                ┌──────────────────────────┐
                │   Kubernetes API Server  │
                │  (Porch Extension API)   │
                └────────────┬─────────────┘
                             │
                             ▼
                ┌──────────────────────────┐
                │    Porch Controller      │
                │  (PackageRevision CRD)   │
                └────────────┬─────────────┘
                             │ Git Sync
                             ▼
                ┌──────────────────────────┐
                │      Git Repository      │
                │  (focom-resources.git)   │
                └────────────┬─────────────┘
                             │ Poll (15s)
                             ▼
                ┌──────────────────────────┐
                │   ConfigSync/Flux        │
                │ (config-management-sys)  │
                └────────────┬─────────────┘
                             │ Apply
                             ▼
                ┌──────────────────────────┐
                │  Kubernetes CRs          │
                │  (OCloud, TemplateInfo,  │
                │   FocomProvisioningReq)  │
                └──────────────────────────┘
```


### Component Descriptions

#### 1. FOCOM NBI REST API

**Technology:** Go + Gin HTTP framework  
**Port:** 8080  
**Purpose:** Provides HTTP REST API for managing FOCOM resources

**Key Responsibilities:**
- Accept and validate HTTP requests
- Implement draft-validate-approve workflow
- Interact with Porch storage layer
- Return standardized JSON responses
- Provide health check endpoints
- Serve OpenAPI specification

**API Endpoints:**
- `/api/v1/o-clouds/*` - OCloud management
- `/api/v1/template-infos/*` - TemplateInfo management
- `/api/v1/focom-provisioning-requests/*` - FPR management
- `/health/live` - Liveness probe
- `/health/ready` - Readiness probe
- `/api/info` - API information and OpenAPI spec

#### 2. PorchStorage Interface

**Purpose:** Abstraction layer for Git-backed storage via Porch

**Key Operations:**
- `CreateDraft()` - Create new draft PackageRevision
- `GetDraft()` - Retrieve draft PackageRevision
- `UpdateDraft()` - Update draft PackageRevision
- `DeleteDraft()` - Delete draft PackageRevision
- `ValidateDraft()` - Transition draft to Proposed state
- `RejectDraft()` - Transition Proposed back to Draft
- `ApproveDraft()` - Publish PackageRevision to Git
- `List()` - List approved resources
- `Get()` - Get specific approved resource
- `Delete()` - Delete all revisions of a resource
- `GetRevisions()` - List all revisions
- `GetRevision()` - Get specific revision
- `CreateDraftFromRevision()` - Create new draft from previous revision

**Implementation Details:**
- Communicates with Kubernetes API server
- Manages PackageRevision and PackageRevisionResources
- Handles Porch lifecycle states (Draft → Proposed → Published)
- Generates unique revision identifiers (v1, v2, v3, etc.)


#### 3. Nephio Porch

**Purpose:** Kubernetes-native package management with Git backend

**Key Concepts:**
- **PackageRevision:** Kubernetes CR representing a versioned package
- **Lifecycle States:** Draft → Proposed → Published
- **Workspace:** Temporary Git branch for draft/proposed revisions
- **Repository:** Git repository configuration

**Package Structure:**
```
resource-id-v1/
├── Kptfile          # Required by Porch (package metadata)
└── resource.yaml    # Actual FOCOM resource data
```

**Responsibilities:**
- Manage PackageRevision lifecycle
- Synchronize with Git repository
- Create workspace branches for drafts
- Merge to main branch on approval
- Maintain package metadata

#### 4. Git Repository

**Purpose:** Persistent, version-controlled storage for all FOCOM resources

**Structure:**
```
focom-resources.git/
├── demo-ocloud-01-v1/
│   ├── Kptfile
│   └── ocloud.yaml
├── demo-ocloud-01-v2/
│   ├── Kptfile
│   └── ocloud.yaml
├── edge-template-v1/
│   ├── Kptfile
│   └── templateinfo.yaml
└── deployment-01-v1/
    ├── Kptfile
    └── focomprovisioningrequest.yaml
```

**Benefits:**
- Immutable revision history
- Full audit trail with commit messages
- Disaster recovery capability
- Multi-cluster synchronization source
- Standard Git workflows (branch, merge, revert)


#### 5. ConfigSync / Flux CD

**Purpose:** Automatically synchronize Git resources to Kubernetes CRs

**ConfigSync (Default):**
- Google Anthos Config Management component
- Polls Git every 15 seconds
- Automatic namespace creation
- Self-healing (reverts manual changes)
- Drift detection and reporting

**Flux CD (Alternative):**
- CNCF GitOps tool
- Webhook-based or polling synchronization
- Kustomize and Helm support
- Multi-tenancy capabilities

**Configuration Example (ConfigSync):**
```yaml
apiVersion: configsync.gke.io/v1beta1
kind: RootSync
metadata:
  name: focom-resources
  namespace: config-management-system
spec:
  sourceFormat: unstructured
  git:
    repo: http://gitea.default.svc.cluster.local:3000/nephio/focom-resources.git
    branch: main
    dir: /
    auth: token
    secretRef:
      name: gitea-secret
```

**Annotations Added to CRs:**
```yaml
metadata:
  annotations:
    configmanagement.gke.io/managed: enabled
    configmanagement.gke.io/source-path: demo-ocloud-01-v1/ocloud.yaml
    configsync.gke.io/git-context: '{"repo":"...","branch":"main"}'
```

#### 6. Kubernetes Custom Resources

**Purpose:** Runtime representation of FOCOM resources

**CRDs:**
- `OCloud` (focom.nephio.org/v1alpha1)
- `TemplateInfo` (provisioning.oran.org/v1alpha1)
- `FocomProvisioningRequest` (focom.nephio.org/v1alpha1)

**Management:**
- Created and updated by ConfigSync/Flux
- Git is the source of truth
- Manual changes are automatically reverted
- Status subresource for reconciler updates


### Sequence Diagrams

#### Create and Approve Flow

```
Client                 API              PorchStorage      K8s API          Porch           Git            ConfigSync      K8s CR
  │                     │                     │               │               │               │                │              │
  │ POST /o-clouds/draft │                     │               │               │               │                │              │
  ├────────────────────>│                     │               │               │               │                │              │
  │                     │ CreateDraft()       │               │               │               │                │              │
  │                     ├────────────────────>│               │               │               │                │              │
  │                     │                     │ POST PackageRevision          │               │                │              │
  │                     │                     │ (lifecycle=Draft)             │               │                │              │
  │                     │                     ├──────────────>│               │               │                │              │
  │                     │                     │               │ Create workspace              │                │              │
  │                     │                     │               ├──────────────>│               │                │              │
  │                     │                     │ PUT PackageRevisionResources  │               │                │              │
  │                     │                     ├──────────────>│               │               │                │              │
  │                     │                     │               │ Commit to workspace           │                │              │
  │                     │                     │               ├──────────────────────────────>│                │              │
  │                     │ Draft created       │               │               │               │                │              │
  │ 201 Created         │<────────────────────┤               │               │               │                │              │
  │<────────────────────┤                     │               │               │               │                │              │
  │                     │                     │               │               │               │                │              │
  │ POST /draft/validate│                     │               │               │               │                │              │
  ├────────────────────>│                     │               │               │               │                │              │
  │                     │ ValidateDraft()     │               │               │               │                │              │
  │                     ├────────────────────>│               │               │               │                │              │
  │                     │                     │ PUT PackageRevision           │               │                │              │
  │                     │                     │ (lifecycle=Proposed)          │               │                │              │
  │                     │                     ├──────────────>│               │               │                │              │
  │ 200 OK              │<────────────────────┤               │               │               │                │              │
  │<────────────────────┤                     │               │               │               │                │              │
  │                     │                     │               │               │               │                │              │
  │ POST /draft/approve │                     │               │               │               │                │              │
  ├────────────────────>│                     │               │               │               │                │              │
  │                     │ ApproveDraft()      │               │               │               │                │              │
  │                     ├────────────────────>│               │               │               │                │              │
  │                     │                     │ PUT PackageRevision           │               │                │              │
  │                     │                     │ (lifecycle=Published, rev=v1) │               │                │              │
  │                     │                     ├──────────────>│               │               │                │              │
  │                     │                     │               │ Merge to main │               │                │              │
  │                     │                     │               ├──────────────────────────────>│                │              │
  │                     │                     │               │ Create package dir (id-v1/)   │                │              │
  │                     │                     │               ├──────────────────────────────>│                │              │
  │ 200 OK              │<────────────────────┤               │               │               │                │              │
  │<────────────────────┤                     │               │               │               │                │              │
  │                     │                     │               │               │               │                │              │
  │                     │                     │               │               │               │ Poll (15s)     │              │
  │                     │                     │               │               │               │<───────────────┤              │
  │                     │                     │               │               │               │ New commit     │              │
  │                     │                     │               │               │               ├───────────────>│              │
  │                     │                     │               │               │               │ Read YAML      │              │
  │                     │                     │               │               │               │<───────────────┤              │
  │                     │                     │               │               │               │                │ Apply CR     │
  │                     │                     │               │               │               │                ├─────────────>│
  │                     │                     │               │               │               │                │ Add annotations
  │                     │                     │               │               │               │                ├─────────────>│
```


#### Self-Healing Flow

```
User                   K8s CR           ConfigSync         Git
  │                      │                  │               │
  │ kubectl delete ocloud│                  │               │
  ├─────────────────────>│                  │               │
  │ OCloud deleted       │                  │               │
  │<─────────────────────┤                  │               │
  │                      │                  │               │
  │                      │                  │ Poll (15s)    │
  │                      │                  ├──────────────>│
  │                      │                  │ No changes    │
  │                      │                  │<──────────────┤
  │                      │ Check CR exists  │               │
  │                      │<─────────────────┤               │
  │                      │ CR not found     │               │
  │                      │ (drift detected) │               │
  │                      │─────────────────>│               │
  │                      │                  │ Read YAML     │
  │                      │                  ├──────────────>│
  │                      │                  │<──────────────┤
  │                      │ Recreate CR      │               │
  │                      │<─────────────────┤               │
  │                      │ Add annotations  │               │
  │                      │<─────────────────┤               │
  │                      │                  │               │
  │ (CR restored from Git)                  │               │
```

### Workflow Examples

#### Administrator Setup Workflow

1. **Reference to pre-existing O2IMS Credentials Secret**
There should be a pre-existing secret in the inventory for authentication to O2IMS. This is the **cloud-west**
referred to in the next step.

2. **Register OCloud**
```bash
POST /api/v1/o-clouds/draft
{
  "namespace": "focom-system",
  "name": "cloud-west",
  "o2imsSecret": {
    "secretRef": {
      "name": "cloud-west-creds",
      "namespace": "focom-system"
    }
  }
}

POST /api/v1/o-clouds/cloud-west/draft/validate
POST /api/v1/o-clouds/cloud-west/draft/approve
```

3. **Create Cluster Template**
```bash
POST /api/v1/template-infos/draft
{
  "namespace": "focom-system",
  "name": "edge-cluster-template",
  "templateName": "edge-cluster",
  "templateVersion": "1.0.0",
  "templateParameterSchema": "{...}"
}

POST /api/v1/template-infos/edge-cluster-template/draft/validate
POST /api/v1/template-infos/edge-cluster-template/draft/approve
```


#### User Cluster Deployment Workflow

1. **Create Provisioning Request**
```bash
POST /api/v1/focom-provisioning-requests/draft
{
  "namespace": "focom-system",
  "name": "deploy-cluster-001",
  "ocloudRef": {"name": "cloud-west"},
  "templateInfoRef": {"name": "edge-cluster-template"},
  "templateParameters": {
    "cpu": "4",
    "memory": "8Gi",
    "replicas": 3
  }
}
```

2. **Validate and Approve**
```bash
POST /api/v1/focom-provisioning-requests/deploy-cluster-001/draft/validate
POST /api/v1/focom-provisioning-requests/deploy-cluster-001/draft/approve
```

3. **Monitor Status**
```bash
GET /api/v1/focom-provisioning-requests/deploy-cluster-001

# Check Kubernetes CR
kubectl get focomprovisioningrequest deploy-cluster-001 -n focom-system
```
The response shall contain the processing status of the corresponding O-Cloud O2ims Provisioning Request.

#### Update Workflow

1. **Create Draft from Current Version**
```bash
GET /api/v1/o-clouds/cloud-west/revisions
# Returns: [v1, v2]

POST /api/v1/o-clouds/cloud-west/revisions/v2/draft
```

2. **Update Draft**
```bash
PATCH /api/v1/o-clouds/cloud-west/draft
{
  "description": "Updated description"
}
```

3. **Validate and Approve**
```bash
POST /api/v1/o-clouds/cloud-west/draft/validate
POST /api/v1/o-clouds/cloud-west/draft/approve
# Creates v3
```


### Data Flow and Transformations

#### REST API Format (JSON)

**Client Request:**
```json
POST /api/v1/o-clouds/draft
{
  "namespace": "focom-system",
  "name": "demo-ocloud",
  "description": "Demo OCloud",
  "o2imsSecret": {
    "secretRef": {
      "name": "credentials",
      "namespace": "focom-system"
    }
  }
}
```

#### Internal Storage Format (Kubernetes YAML in Git)

**Git Repository (demo-ocloud-v1/ocloud.yaml):**
```yaml
apiVersion: focom.nephio.org/v1alpha1
kind: OCloud
metadata:
  name: demo-ocloud
  namespace: focom-system
spec:
  id: demo-ocloud
  revisionId: v1
  name: demo-ocloud
  description: Demo OCloud
  state: APPROVED
  o2imsSecret:
    secretRef:
      name: credentials
      namespace: focom-system
```

#### Kubernetes CR (Applied by ConfigSync)

**Cluster State:**
```yaml
apiVersion: focom.nephio.org/v1alpha1
kind: OCloud
metadata:
  name: demo-ocloud
  namespace: focom-system
  annotations:
    configmanagement.gke.io/managed: enabled
    configmanagement.gke.io/source-path: demo-ocloud-v1/ocloud.yaml
    configsync.gke.io/git-context: '{"repo":"http://...","branch":"main"}'
spec:
  id: demo-ocloud
  revisionId: v1
  name: demo-ocloud
  description: Demo OCloud
  state: APPROVED
  o2imsSecret:
    secretRef:
      name: credentials
      namespace: focom-system
status:
  # Populated by FOCOM operator reconciler (future)
  conditions: []
```


### Security Architecture

#### FOCOM Operator Authentication (to Kubernetes API)

The FOCOM operator authenticates to the Kubernetes API to manage PackageRevisions and CRs:

**In-Cluster Deployment:**
- Service account token mounted at `/var/run/secrets/kubernetes.io/serviceaccount/token`
- Automatic token rotation by Kubernetes

**Local Development:**
- Token from `KUBECONFIG` file
- Token from `TOKEN` environment variable
- Token from file path specified in `TOKEN` environment variable

#### FOCOM Operator Authorization (Kubernetes RBAC)

Kubernetes RBAC controls what the FOCOM operator can do:

```yaml
# Porch PackageRevision management
apiGroups: ["porch.kpt.dev"]
resources: ["packagerevisions", "packagerevisionresources"]
verbs: ["get", "list", "create", "update", "delete", "patch"]

# FOCOM Custom Resources
apiGroups: ["focom.nephio.org", "provisioning.oran.org"]
resources: ["oclouds", "templateinfoes", "focomprovisioningrequests"]
verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Repository access
apiGroups: ["porch.kpt.dev"]
resources: ["repositories"]
verbs: ["get", "list"]
```

#### REST API Client Authentication

**Current State:** The REST API (port 8080) has **no authentication**. Any client with network access can make requests.

**Mitigation:** Deploy behind network security controls (firewall, network policies, service mesh).

**Future:** External authentication mechanisms (OAuth, OIDC, API keys) are out of scope for this user story.

#### Data Protection

- **Secrets:** O2IMS credentials stored in Kubernetes Secrets
- **Git:** No sensitive data in Git (only references to Secrets)
- **Audit:** Full audit trail via Git commit history


### Deployment Architecture

#### Kubernetes Resources

```
focom-operator-system/
├── Deployment: focom-operator-controller-manager
│   └── Container: manager (FOCOM NBI API + reconcilers)
├── Service: focom-operator-controller-manager-metrics-service
├── ServiceAccount: focom-operator-controller-manager
├── ClusterRole: focom-operator-manager-role
└── ClusterRoleBinding: focom-operator-manager-rolebinding

config-management-system/
├── RootSync: focom-resources
│   └── Spec: Git repo configuration
└── Secret: gitea-secret (Git credentials)

focom-system/
├── OCloud CRs (managed by ConfigSync)
├── TemplateInfo CRs (managed by ConfigSync)
├── FocomProvisioningRequest CRs (managed by ConfigSync)
└── Secrets (O2IMS credentials)
```

#### Configuration

**Environment Variables:**
```bash
NBI_STORAGE_BACKEND=porch          # Storage backend (porch or memory)
NBI_STAGE=2                        # Stage identifier
PORCH_NAMESPACE=default            # Namespace for PackageRevisions
PORCH_REPOSITORY=focom-resources   # Porch repository name
KUBERNETES_BASE_URL=https://...    # K8s API URL (auto-detected)
TOKEN=/path/to/token               # Auth token (auto-detected)
```

**ConfigMap (if needed):**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: focom-nbi-config
  namespace: focom-operator-system
data:
  storage-backend: "porch"
  porch-namespace: "default"
  porch-repository: "focom-resources"
```


### Performance Characteristics

#### Latency

- **Draft operations** (Create, Update, Get): ~100-500ms
  - Kubernetes API calls to manage PackageRevisions
  - Network latency to Kubernetes API server
  
- **Validation**: ~200-500ms
  - Kubernetes API call to update lifecycle state
  - Referential integrity checks
  
- **Approval**: ~500ms-1s
  - Kubernetes API call to publish PackageRevision
  - Porch commits to Git repository
  
- **CR creation** (after approval): 0-15s
  - ConfigSync poll interval (default 15s)
  - Flux can be faster with webhooks
  
- **List operations**: ~100-300ms
  - Kubernetes API list call
  - Scales with number of resources

#### Scalability

- **Resources per repository:** Thousands (limited by Git performance)
- **Concurrent operations:** Limited by Kubernetes API server capacity
- **Revision history:** Unlimited (stored in Git)
- **API throughput:** Hundreds of requests per second (limited by Gin framework and K8s API)

#### Reliability

- **Git as source of truth:** Survives cluster failures and restarts
- **Self-healing:** ConfigSync automatically restores CRs from Git
- **Audit trail:** Full history in Git commits with timestamps
- **Backup:** Git repository can be backed up and replicated
- **Multi-cluster:** Same Git repo can sync to multiple clusters


### Monitoring and Observability

#### Health Checks

```bash
# API liveness probe
curl http://localhost:8080/health/live
# Returns: {"status": "ok"}

# API readiness probe
curl http://localhost:8080/health/ready
# Returns: {"status": "ok", "porch": "connected"}

# ConfigSync status
kubectl get rootsync focom-resources -n config-management-system
# Check: SYNCERRORCOUNT, RENDERINGERRORCOUNT, SOURCEERRORCOUNT

# Porch connectivity
kubectl get repositories -n default
# Verify repository is Ready
```

#### Logs

```bash
# FOCOM operator logs (includes NBI API)
kubectl logs -n focom-operator-system \
  -l control-plane=controller-manager \
  --tail=100 -f

# ConfigSync logs
kubectl logs -n config-management-system \
  -l app=reconciler-manager \
  --tail=100 -f

# Porch logs
kubectl logs -n porch-system \
  -l app=porch-server \
  --tail=100 -f
```

#### Metrics

**ConfigSync Metrics:**
- `SYNCERRORCOUNT`: Number of sync errors
- `RENDERINGERRORCOUNT`: Number of rendering errors
- `SOURCEERRORCOUNT`: Number of source errors
- Last sync timestamp

**Porch Metrics:**
- PackageRevision counts by lifecycle state
- Repository sync status
- Git operation latency

**API Metrics:**
- Request count by endpoint
- Request latency by endpoint
- Error rate by status code
- Active connections


### Advantages of This Architecture

1. **GitOps Native:** Git is the source of truth for FOCOM resources
2. **Self-Healing:** ConfigSync/Flux automatically restores deleted or modified CRs
3. **Audit Trail:** Full history in Git commits with author, timestamp, and message
4. **Disaster Recovery:** Complete system state can be restored from Git
5. **No Custom Sync Code:** Leverages battle-tested ConfigSync or Flux
7. **Declarative:** Resources defined in YAML, not imperative API calls
8. **Revision History:** Immutable versions stored in Git (v1, v2, v3, etc.)
9. **Rollback:** Can revert to any previous revision by creating draft from old version
10. **Compliance:** Git provides audit trail for compliance and regulatory requirements
11. **Safe Workflow:** Draft-validate-approve prevents accidental production changes
12. **Referential Integrity:** Validation ensures referenced resources exist
13. **Standard Tools:** Works with standard Git, Kubernetes, and GitOps tools
14. **Extensible:** Easy to add new resource types following same pattern

### Future Enhancements

The following enhancements are planned for future releases:

1. **Webhook Notifications:** Real-time notifications for CR creation (reduce 15s delay)
2. **Metrics Dashboard:** Grafana dashboard for sync latency and API metrics
3. **Multi-Repository Support:** Manage resources across multiple Git repositories
4. **Template-Based Provisioning:** PackageVariant for template instantiation
5. **Advanced Caching:** Kubernetes watch API for real-time updates
6. **Batch Operations:** Bulk create/update/delete operations
7. **Resource Dependencies:** Automatic ordering of resource creation
8. **Validation Webhooks:** Admission webhooks for additional validation
9. **Status Aggregation:** Aggregate status from multiple resources
10. **Performance Optimizations:** Caching, connection pooling, batch operations

---

**Document Status:** Draft - Subject to change based on stakeholder feedback

**Last Updated:** 2025-12-09

**Version:** 1.0

