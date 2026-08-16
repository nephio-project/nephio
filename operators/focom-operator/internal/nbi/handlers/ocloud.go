/*
Copyright 2026 The Nephio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/integration"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/storage"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OCloudHandler handles HTTP requests for OCloud resources
type OCloudHandler struct {
	BaseHandler
	storage          storage.StorageInterface
	integration      integration.OperatorIntegration // Optional operator integration
	defaultNamespace string                          // Default namespace from config
}

// NewOCloudHandler creates a new OCloud handler
func NewOCloudHandler(storage storage.StorageInterface, defaultNamespace string) *OCloudHandler {
	return &OCloudHandler{
		BaseHandler:      *NewBaseHandler(),
		storage:          storage,
		integration:      nil, // No operator integration
		defaultNamespace: defaultNamespace,
	}
}

// NewOCloudHandlerWithIntegration creates a new OCloud handler with operator integration
func NewOCloudHandlerWithIntegration(storage storage.StorageInterface, integration integration.OperatorIntegration, defaultNamespace string) *OCloudHandler {
	return &OCloudHandler{
		BaseHandler:      *NewBaseHandler(),
		storage:          storage,
		integration:      integration,
		defaultNamespace: defaultNamespace,
	}
}

// CreateDraft creates a new OCloud draft
// POST /o-clouds/draft
func (h *OCloudHandler) CreateDraft(c *gin.Context) {
	var req struct {
		Namespace   string                `json:"namespace"` // DEPRECATED: Use FOCOM_NAMESPACE environment variable instead
		Name        string                `json:"name" binding:"required"`
		Description string                `json:"description" binding:"required"`
		O2IMSSecret models.O2IMSSecretRef `json:"o2imsSecret" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Invalid request body", err.Error())
		return
	}

	// Determine namespace: request body (deprecated) takes precedence over default for backward compatibility
	namespace := h.defaultNamespace
	if req.Namespace != "" {
		namespace = req.Namespace
		// Log deprecation warning
		log.Log.WithName("ocloud-handler").Info(
			"DEPRECATION WARNING: 'namespace' field in request body is deprecated. Use FOCOM_NAMESPACE environment variable instead.",
			"resource", "OCloud",
			"name", req.Name,
			"namespace", req.Namespace,
		)
	}

	// Create new OCloud data
	oCloudData := models.NewOCloudData(namespace, req.Name, req.Description, req.O2IMSSecret)

	// Store the draft
	if err := h.storage.CreateDraft(c.Request.Context(), storage.ResourceTypeOCloud, oCloudData); err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to create OCloud draft", err.Error())
		return
	}

	h.SendCreated(c, oCloudData)
}

// GetDraft retrieves an OCloud draft
// GET /o-clouds/{oCloudId}/draft
func (h *OCloudHandler) GetDraft(c *gin.Context) {
	oCloudID := c.Param("oCloudId")
	if oCloudID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing oCloudId parameter", "")
		return
	}

	draft, err := h.storage.GetDraft(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "OCloud draft not found", err.Error())
		return
	}

	h.SendOK(c, draft)
}

// UpdateDraft updates an OCloud draft
// PATCH /o-clouds/{oCloudId}/draft
func (h *OCloudHandler) UpdateDraft(c *gin.Context) {
	oCloudID := c.Param("oCloudId")
	if oCloudID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing oCloudId parameter", "")
		return
	}

	var updateReq models.OCloudDataUpdate
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Invalid request body", err.Error())
		return
	}

	// Get existing draft
	existingDraft, err := h.storage.GetDraft(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "OCloud draft not found", err.Error())
		return
	}

	// Storage now returns models.OCloudData after conversion
	oCloudData, ok := existingDraft.(*models.OCloudData)
	if !ok {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Invalid draft data type", "")
		return
	}

	// Check if draft is in a state that allows updates
	if oCloudData.State == models.StateValidated {
		h.SendConflict(c, models.ErrorCodeInvalidState, "Cannot update validated draft", "Draft must be in DRAFT state to allow updates")
		return
	}

	// Apply updates
	oCloudData.Update(updateReq.Name, updateReq.Description, updateReq.O2IMSSecret)

	// Update the draft in storage (storage layer will handle type conversion)
	if err := h.storage.UpdateDraft(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID, oCloudData); err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to update OCloud draft", err.Error())
		return
	}

	h.SendOK(c, oCloudData)
}

// DeleteDraft deletes an OCloud draft
// DELETE /o-clouds/{oCloudId}/draft
func (h *OCloudHandler) DeleteDraft(c *gin.Context) {
	oCloudID := c.Param("oCloudId")
	if oCloudID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing oCloudId parameter", "")
		return
	}

	if err := h.storage.DeleteDraft(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID); err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "OCloud draft not found", err.Error())
		return
	}

	h.SendNoContent(c)
}

// ValidateDraft validates an OCloud draft
// POST /o-clouds/{oCloudId}/draft/validate
func (h *OCloudHandler) ValidateDraft(c *gin.Context) {
	oCloudID := c.Param("oCloudId")
	if oCloudID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing oCloudId parameter", "")
		return
	}

	if err := h.storage.ValidateDraft(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Validation failed", err.Error())
		return
	}

	// Get the validated draft to return
	draft, err := h.storage.GetDraft(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to retrieve validated draft", err.Error())
		return
	}

	validationResult := models.NewValidationResult(true, nil, nil)
	response := map[string]interface{}{
		"validationResult": validationResult,
		"draft":            draft,
	}

	h.SendOK(c, response)
}

// ApproveDraft approves an OCloud draft
// POST /o-clouds/{oCloudId}/draft/approve
func (h *OCloudHandler) ApproveDraft(c *gin.Context) {
	oCloudID := c.Param("oCloudId")
	if oCloudID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing oCloudId parameter", "")
		return
	}

	// Validate dependencies before approval (for FPR resources)
	// For OCloud, no dependencies to validate, but we keep the pattern consistent

	if err := h.storage.ApproveDraft(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, err.Error(), "")
		return
	}

	// Get the approved resource
	approvedResource, err := h.storage.Get(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to retrieve approved resource", err.Error())
		return
	}

	// NOTE: CR creation is now handled by ConfigSync (Git → Kubernetes sync)
	// ConfigSync watches the Git repository and automatically creates CRs when
	// PackageRevisions are Published. This provides true GitOps with Git as source of truth.
	//
	// The following code is commented out to avoid duplicate CR creation:
	//
	// // Create Kubernetes CR if operator integration is available
	// if h.integration != nil {
	// 	oCloudData, ok := approvedResource.(*models.OCloudData)
	// 	if !ok {
	// 		h.SendInternalError(c, models.ErrorCodeInternalError, "Invalid resource data type", "")
	// 		return
	// 	}
	//
	// 	if err := h.integration.CreateOCloudCR(c.Request.Context(), oCloudData); err != nil {
	// 		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to create OCloud CR", err.Error())
	// 		return
	// 	}
	// }

	h.SendOK(c, approvedResource)
}

// RejectDraft rejects an OCloud draft
// POST /o-clouds/{oCloudId}/draft/reject
func (h *OCloudHandler) RejectDraft(c *gin.Context) {
	oCloudID := c.Param("oCloudId")
	if oCloudID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing oCloudId parameter", "")
		return
	}

	// Parse rejection request body
	var rejectReq struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&rejectReq); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Invalid request body", err.Error())
		return
	}

	if err := h.storage.RejectDraft(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Rejection failed", err.Error())
		return
	}

	// Get the rejected draft to return
	draft, err := h.storage.GetDraft(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to retrieve rejected draft", err.Error())
		return
	}

	// Return response in the format expected by tests
	response := map[string]interface{}{
		"rejected": true,
		"reason":   rejectReq.Reason,
		"draft":    draft,
	}

	h.SendOK(c, response)
}

// ListOClouds lists all approved OCloud configurations
// GET /o-clouds
func (h *OCloudHandler) ListOClouds(c *gin.Context) {
	resources, err := h.storage.List(c.Request.Context(), storage.ResourceTypeOCloud)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to list OClouds", err.Error())
		return
	}

	// Convert to OCloudInfo format with status information
	var oCloudInfos []models.OCloudInfo
	for _, resource := range resources {
		if oCloudData, ok := resource.(*models.OCloudData); ok {
			oCloudInfo := models.OCloudInfo{
				OCloudData:   oCloudData,
				OCloudStatus: &models.OCloudStatus{Message: "Active"},
			}
			oCloudInfos = append(oCloudInfos, oCloudInfo)
		}
	}

	// Return array directly as per OpenAPI spec
	h.SendOK(c, oCloudInfos)
}

// GetOCloud retrieves a specific approved OCloud configuration
// GET /o-clouds/{oCloudId}
func (h *OCloudHandler) GetOCloud(c *gin.Context) {
	oCloudID := c.Param("oCloudId")
	if oCloudID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing oCloudId parameter", "")
		return
	}

	resource, err := h.storage.Get(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "OCloud not found", err.Error())
		return
	}

	oCloudData, ok := resource.(*models.OCloudData)
	if !ok {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Invalid resource data type", "")
		return
	}

	// Return the resource data directly for individual GET requests
	// This matches the expectation in cross_resource_integration_test.go
	h.SendOK(c, oCloudData)
}

// DeleteOCloud deletes an approved OCloud configuration
// DELETE /o-clouds/{oCloudId}
func (h *OCloudHandler) DeleteOCloud(c *gin.Context) {
	oCloudID := c.Param("oCloudId")
	if oCloudID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing oCloudId parameter", "")
		return
	}

	// Get the resource first to have complete data for dependency validation
	resource, err := h.storage.Get(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "OCloud not found", err.Error())
		return
	}

	// Check for dependencies before deletion
	if err := h.storage.ValidateDependencies(c.Request.Context(), storage.ResourceTypeOCloud, resource); err != nil {
		h.SendConflict(c, models.ErrorCodeDependency, "OCloud cannot be deleted: it is referenced by FocomProvisioningRequest resources", err.Error())
		return
	}

	if err := h.storage.Delete(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID); err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "OCloud not found", err.Error())
		return
	}

	h.SendAccepted(c, map[string]string{"message": "OCloud deletion initiated"})
}

// GetRevisions lists all revisions for an OCloud
// GET /o-clouds/{oCloudId}/revisions
func (h *OCloudHandler) GetRevisions(c *gin.Context) {
	oCloudID := c.Param("oCloudId")
	if oCloudID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing oCloudId parameter", "")
		return
	}

	revisions, err := h.storage.GetRevisions(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "OCloud revisions not found", err.Error())
		return
	}

	// Transform to revision format with resourceId and revisionId
	var revisionList []map[string]interface{}
	for _, rev := range revisions {
		if oCloudData, ok := rev.(*models.OCloudData); ok {
			revisionList = append(revisionList, map[string]interface{}{
				"resourceId":  oCloudData.ID,
				"revisionId":  oCloudData.RevisionID,
				"name":        oCloudData.Name,
				"description": oCloudData.Description,
				"state":       oCloudData.State,
				"createdAt":   oCloudData.CreatedAt,
				"updatedAt":   oCloudData.UpdatedAt,
			})
		}
	}

	// Return array directly as per OpenAPI spec
	h.SendOK(c, revisionList)
}

// GetRevision retrieves a specific revision of an OCloud
// GET /o-clouds/{oCloudId}/revisions/{revisionId}
func (h *OCloudHandler) GetRevision(c *gin.Context) {
	oCloudID := c.Param("oCloudId")
	revisionID := c.Param("revisionId")

	if oCloudID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing oCloudId parameter", "")
		return
	}
	if revisionID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing revisionId parameter", "")
		return
	}

	revision, err := h.storage.GetRevision(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID, revisionID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "OCloud revision not found", err.Error())
		return
	}

	h.SendOK(c, revision)
}

// CreateDraftFromRevision creates a new draft from a specific revision
// POST /o-clouds/{oCloudId}/revisions/{revisionId}/draft
func (h *OCloudHandler) CreateDraftFromRevision(c *gin.Context) {
	oCloudID := c.Param("oCloudId")
	revisionID := c.Param("revisionId")

	if oCloudID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing oCloudId parameter", "")
		return
	}
	if revisionID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing revisionId parameter", "")
		return
	}

	if err := h.storage.CreateDraftFromRevision(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID, revisionID); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Failed to create draft from revision", err.Error())
		return
	}

	// Get the newly created draft
	draft, err := h.storage.GetDraft(c.Request.Context(), storage.ResourceTypeOCloud, oCloudID)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to retrieve created draft", err.Error())
		return
	}

	h.SendCreated(c, draft)
}

// RegisterRoutes registers all OCloud routes with the router
func (h *OCloudHandler) RegisterRoutes(router gin.IRouter) {
	oCloudGroup := router.Group("/o-clouds")
	{
		// Draft management
		oCloudGroup.POST("/draft", h.CreateDraft)
		oCloudGroup.GET("/:oCloudId/draft", h.GetDraft)
		oCloudGroup.PATCH("/:oCloudId/draft", h.UpdateDraft)
		oCloudGroup.DELETE("/:oCloudId/draft", h.DeleteDraft)
		oCloudGroup.POST("/:oCloudId/draft/validate", h.ValidateDraft)
		oCloudGroup.POST("/:oCloudId/draft/approve", h.ApproveDraft)
		oCloudGroup.POST("/:oCloudId/draft/reject", h.RejectDraft)

		// Approved resource management
		oCloudGroup.GET("", h.ListOClouds)
		oCloudGroup.GET("/:oCloudId", h.GetOCloud)
		oCloudGroup.DELETE("/:oCloudId", h.DeleteOCloud)

		// Revision management
		oCloudGroup.GET("/:oCloudId/revisions", h.GetRevisions)
		oCloudGroup.GET("/:oCloudId/revisions/:revisionId", h.GetRevision)
		oCloudGroup.POST("/:oCloudId/revisions/:revisionId/draft", h.CreateDraftFromRevision)
	}
}
