# User Story - FOCOM NBI Lifecycle Management - PROPOSED

**Feature Overview**

The FOCOM (Federated O-Cloud Cluster Orchestration Management) North Bound Interface (NBI) provides a REST API for managing the complete lifecycle of FOCOM infrastructure resources. This interface enables administrators and orchestrators to declaratively manage O-Cloud configurations, cluster templates, and provisioning requests through a Git-backed, revision-controlled system.

The FOCOM NBI implements a draft-validate-approve workflow that ensures safe infrastructure-as-code management, with Git serving as the source of truth for FOCOM resources.

**Stakeholders**

* Infrastructure Administrators - Managing O-Cloud configurations and cluster templates
* Service Orchestrators - Creating and managing cluster provisioning requests  
* Platform Operators - Monitoring and maintaining FOCOM infrastructure
* O-RAN Architecture Teams - Defining standards for FOCOM integration

**User Stories**

**As an Infrastructure Administrator**
- I want to register O-Cloud configurations so that clusters can be deployed to specific infrastructure targets
- I want to define cluster templates so that standardized cluster types can be reused across deployments
- I want to manage these configurations through a safe draft-validate-approve workflow so that production changes are controlled

**As a Service Orchestrator**
- I want to create cluster provisioning requests by combining O-Cloud targets with cluster templates so that I can deploy clusters where needed
- I want to specify instance-specific parameters so that each deployment can be customized appropriately
- I want to track the complete lifecycle and revision history of my provisioning requests

**As a Platform Operator**
- I want all FOCOM resources stored in Git so that I have a complete audit trail and can recover from disasters
- I want automatic synchronization between Git and Kubernetes so that the system is self-healing
- I want to monitor the health and status of the FOCOM NBI API

**Core Requirements**

1. **Resource Management**: REST API for managing three core resource types:
   - **OCloud**: FOCOM's knowledge about an O-Cloud, including O2IMS endpoint and credentials. Also serves as the handle for associating TemplateInfo resources with that O-Cloud
   - **TemplateInfo**: FOCOM's cached metadata about MIT templates available on a specific O-Cloud. The actual MIT templates are defined and owned by the O-Cloud
   - **FocomProvisioningRequest**: Combines OCloud + TemplateInfo for actual deployment

2. **Draft-Validate-Approve Workflow**: All resources follow a controlled lifecycle:
   - **Draft**: Editable state for iterative development
   - **Validate**: Verify referential integrity and parameter correctness
   - **Approve**: Commit to Git and trigger deployment

3. **Git-Backed Storage**: 
   - Git serves as source of truth for FOCOM resources
   - Note that some data (like TemplateInfo) represents cached/aggregated information from other systems
   - Immutable revision history (v1, v2, v3, etc.)
   - Full audit trail with timestamps and change tracking

4. **GitOps Synchronization**:
   - Automatic sync from Git to Kubernetes Custom Resources
   - Self-healing: manual changes to CRs are automatically reverted
   - Integration with ConfigSync or Flux CD

5. **API Standards**:
   - RESTful HTTP API with JSON request/response
   - Standard HTTP methods and status codes
   - OpenAPI specification for documentation
   - Health check endpoints for monitoring

**Key Benefits**

- **Safety**: Draft-validate-approve prevents accidental production changes
- **Auditability**: Complete history in Git with author, timestamp, and change details
- **Reliability**: Self-healing system automatically restores from Git state
- **Cached Orchestration**: OClouds and TemplateInfos represent FOCOM's cached view of O-Cloud capabilities, maintained and updated as the underlying O-Cloud resources change
- **Disaster Recovery**: Complete system state recoverable from Git repository

**Acceptance Criteria Summary**

- REST API endpoints for all CRUD operations on OCloud, TemplateInfo, and FocomProvisioningRequest (including draft deletion)
- Draft-validate-approve workflow implementation for all resource types
- Git storage via Nephio Porch with immutable revision history
- GitOps synchronization with ConfigSync/Flux
- Referential integrity validation and template parameter schema validation
- OCloud GET operations return O2IMS availability status
- FocomProvisioningRequest GET operations return O2IMS provisioning status
- FocomProvisioningRequest approval triggers southbound O2IMS provisioning request
- FocomProvisioningRequest deletion triggers southbound O2IMS deletion request
- Health check and monitoring endpoints
- OpenAPI specification and documentation

**What's Not in Scope**

- Southbound O2IMS integration for actual cluster deployment (interface points defined, implementation deferred)
- Cluster lifecycle management (scaling, upgrading, monitoring)
- REST API authentication/authorization for external clients (currently no authentication - deploy behind network security)
- Cross-o-cloud deployment coordination

**Success Metrics**

- Infrastructure administrators can safely manage O-Cloud and template configurations
- Service orchestrators can efficiently create and track cluster provisioning requests
- Platform operators have full visibility and control over FOCOM infrastructure
- All changes are auditable and recoverable through Git history
- System demonstrates self-healing capabilities when CRs are manually modified

---

**Document Status:** Proposed - Subject to stakeholder review and approval

**Last Updated:** 2025-12-09

**Version:** 1.0 (Proposal)