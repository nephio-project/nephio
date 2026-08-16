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
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/integration"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/storage"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TemplateInfoHandler handles HTTP requests for TemplateInfo resources
type TemplateInfoHandler struct {
	BaseHandler
	storage          storage.StorageInterface
	integration      integration.OperatorIntegration // Optional operator integration
	defaultNamespace string                          // Default namespace from config
}

// NewTemplateInfoHandler creates a new TemplateInfo handler
func NewTemplateInfoHandler(storage storage.StorageInterface, defaultNamespace string) *TemplateInfoHandler {
	return &TemplateInfoHandler{
		BaseHandler:      *NewBaseHandler(),
		storage:          storage,
		integration:      nil, // No operator integration
		defaultNamespace: defaultNamespace,
	}
}

// NewTemplateInfoHandlerWithIntegration creates a new TemplateInfo handler with operator integration
func NewTemplateInfoHandlerWithIntegration(storage storage.StorageInterface, integration integration.OperatorIntegration, defaultNamespace string) *TemplateInfoHandler {
	return &TemplateInfoHandler{
		BaseHandler:      *NewBaseHandler(),
		storage:          storage,
		integration:      integration,
		defaultNamespace: defaultNamespace,
	}
}

// validateTemplateParameterSchema validates that the template parameter schema is valid JSON or YAML
func (h *TemplateInfoHandler) validateTemplateParameterSchema(schema string) error {
	if schema == "" {
		return &models.ValidationError{
			Field:   "templateParameterSchema",
			Message: "Template parameter schema cannot be empty",
		}
	}

	// Try to parse as JSON first
	var jsonData interface{}
	if err := json.Unmarshal([]byte(schema), &jsonData); err != nil {
		// If JSON parsing fails, check if it looks like malformed JSON
		trimmed := strings.TrimSpace(schema)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			// Looks like JSON but failed to parse - reject it
			return &models.ValidationError{
				Field:   "templateParameterSchema",
				Message: "Template parameter schema must be valid JSON: " + err.Error(),
			}
		}
		// Otherwise, accept it as YAML (basic validation)
		// For now, we'll accept any non-JSON string as potential YAML
		// In a production system, you might want to add proper YAML validation
		return nil
	}

	// Basic JSON Schema validation - recursively check for valid types
	validTypes := map[string]bool{
		"object": true, "array": true, "string": true,
		"number": true, "integer": true, "boolean": true, "null": true,
	}

	var validateSchema func(interface{}) error
	validateSchema = func(data interface{}) error {
		if schemaObj, ok := data.(map[string]interface{}); ok {
			// Check type field if it exists
			if schemaType, exists := schemaObj["type"]; exists {
				if typeStr, ok := schemaType.(string); ok {
					if !validTypes[typeStr] {
						return &models.ValidationError{
							Field:   "templateParameterSchema",
							Message: "Invalid JSON Schema type: " + typeStr,
						}
					}
				}
			}

			// Recursively validate properties
			if properties, exists := schemaObj["properties"]; exists {
				if propsObj, ok := properties.(map[string]interface{}); ok {
					for _, propSchema := range propsObj {
						if err := validateSchema(propSchema); err != nil {
							return err
						}
					}
				}
			}

			// Recursively validate items (for arrays)
			if items, exists := schemaObj["items"]; exists {
				if err := validateSchema(items); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := validateSchema(jsonData); err != nil {
		return err
	}

	return nil
}

// CreateDraft creates a new TemplateInfo draft
// POST /template-infos/draft
func (h *TemplateInfoHandler) CreateDraft(c *gin.Context) {
	var req struct {
		Namespace               string `json:"namespace"` // DEPRECATED: Use FOCOM_NAMESPACE environment variable instead
		Name                    string `json:"name" binding:"required"`
		Description             string `json:"description" binding:"required"`
		TemplateName            string `json:"templateName" binding:"required"`
		TemplateVersion         string `json:"templateVersion" binding:"required"`
		TemplateParameterSchema string `json:"templateParameterSchema" binding:"required"`
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
		log.Log.WithName("templateinfo-handler").Info(
			"DEPRECATION WARNING: 'namespace' field in request body is deprecated. Use FOCOM_NAMESPACE environment variable instead.",
			"resource", "TemplateInfo",
			"name", req.Name,
			"namespace", req.Namespace,
		)
	}

	// Validate template parameter schema
	if err := h.validateTemplateParameterSchema(req.TemplateParameterSchema); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Invalid template parameter schema", err.Error())
		return
	}

	// Create new TemplateInfo data
	templateInfoData := models.NewTemplateInfoData(
		namespace,
		req.Name,
		req.Description,
		req.TemplateName,
		req.TemplateVersion,
		req.TemplateParameterSchema,
	)

	// Store the draft
	if err := h.storage.CreateDraft(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoData); err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to create TemplateInfo draft", err.Error())
		return
	}

	h.SendCreated(c, templateInfoData)
}

// GetDraft retrieves a TemplateInfo draft
// GET /template-infos/{templateInfoId}/draft
func (h *TemplateInfoHandler) GetDraft(c *gin.Context) {
	templateInfoID := c.Param("templateInfoId")
	if templateInfoID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing templateInfoId parameter", "")
		return
	}

	draft, err := h.storage.GetDraft(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "TemplateInfo draft not found", err.Error())
		return
	}

	h.SendOK(c, draft)
}

// UpdateDraft updates a TemplateInfo draft
// PATCH /template-infos/{templateInfoId}/draft
func (h *TemplateInfoHandler) UpdateDraft(c *gin.Context) {
	templateInfoID := c.Param("templateInfoId")
	if templateInfoID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing templateInfoId parameter", "")
		return
	}

	var updateReq models.TemplateInfoDataUpdate
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Invalid request body", err.Error())
		return
	}

	// Validate template parameter schema if provided
	if updateReq.TemplateParameterSchema != nil {
		if err := h.validateTemplateParameterSchema(*updateReq.TemplateParameterSchema); err != nil {
			h.SendBadRequest(c, models.ErrorCodeValidation, "Invalid template parameter schema", err.Error())
			return
		}
	}

	// Get existing draft
	existingDraft, err := h.storage.GetDraft(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "TemplateInfo draft not found", err.Error())
		return
	}

	templateInfoData, ok := existingDraft.(*models.TemplateInfoData)
	if !ok {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Invalid draft data type", "")
		return
	}

	// Check if draft is in a state that allows updates
	if templateInfoData.State == models.StateValidated {
		h.SendConflict(c, models.ErrorCodeInvalidState, "Cannot update validated draft", "Draft must be in DRAFT state to allow updates")
		return
	}

	// Apply updates
	templateInfoData.Update(
		updateReq.Name,
		updateReq.Description,
		updateReq.TemplateName,
		updateReq.TemplateVersion,
		updateReq.TemplateParameterSchema,
	)

	// Update the draft in storage
	if err := h.storage.UpdateDraft(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID, templateInfoData); err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to update TemplateInfo draft", err.Error())
		return
	}

	h.SendOK(c, templateInfoData)
}

// DeleteDraft deletes a TemplateInfo draft
// DELETE /template-infos/{templateInfoId}/draft
func (h *TemplateInfoHandler) DeleteDraft(c *gin.Context) {
	templateInfoID := c.Param("templateInfoId")
	if templateInfoID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing templateInfoId parameter", "")
		return
	}

	if err := h.storage.DeleteDraft(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID); err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "TemplateInfo draft not found", err.Error())
		return
	}

	h.SendNoContent(c)
}

// ValidateDraft validates a TemplateInfo draft
// POST /template-infos/{templateInfoId}/draft/validate
func (h *TemplateInfoHandler) ValidateDraft(c *gin.Context) {
	templateInfoID := c.Param("templateInfoId")
	if templateInfoID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing templateInfoId parameter", "")
		return
	}

	// Get the draft first to perform additional validation
	draft, err := h.storage.GetDraft(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "TemplateInfo draft not found", err.Error())
		return
	}

	templateInfoData, ok := draft.(*models.TemplateInfoData)
	if !ok {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Invalid draft data type", "")
		return
	}

	// Perform template parameter schema validation
	if err := h.validateTemplateParameterSchema(templateInfoData.TemplateParameterSchema); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Validation failed", err.Error())
		return
	}

	if err := h.storage.ValidateDraft(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Validation failed", err.Error())
		return
	}

	// Get the validated draft to return
	validatedDraft, err := h.storage.GetDraft(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to retrieve validated draft", err.Error())
		return
	}

	validationResult := models.NewValidationResult(true, nil, nil)
	response := map[string]interface{}{
		"validationResult": validationResult,
		"draft":            validatedDraft,
	}

	h.SendOK(c, response)
}

// ApproveDraft approves a TemplateInfo draft
// POST /template-infos/{templateInfoId}/draft/approve
func (h *TemplateInfoHandler) ApproveDraft(c *gin.Context) {
	templateInfoID := c.Param("templateInfoId")
	if templateInfoID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing templateInfoId parameter", "")
		return
	}

	if err := h.storage.ApproveDraft(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Approval failed", err.Error())
		return
	}

	// Get the approved resource
	approvedResource, err := h.storage.Get(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to retrieve approved resource", err.Error())
		return
	}

	// Convert to TemplateInfo data for response and CR creation
	approvedTemplateInfoData, ok := approvedResource.(*models.TemplateInfoData)
	if !ok {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Invalid resource data type", "")
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
	// 	if err := h.integration.CreateTemplateInfoCR(c.Request.Context(), approvedTemplateInfoData); err != nil {
	// 		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to create TemplateInfo CR", err.Error())
	// 		return
	// 	}
	// }

	// Return approval response with expected fields
	response := map[string]interface{}{
		"approved":    true,
		"approvedAt":  approvedTemplateInfoData.UpdatedAt,
		"revisionId":  approvedTemplateInfoData.RevisionID,
		"id":          approvedTemplateInfoData.ID,
		"name":        approvedTemplateInfoData.Name,
		"description": approvedTemplateInfoData.Description,
		"state":       approvedTemplateInfoData.State,
	}

	h.SendOK(c, response)
}

// RejectDraft rejects a TemplateInfo draft
// POST /template-infos/{templateInfoId}/draft/reject
func (h *TemplateInfoHandler) RejectDraft(c *gin.Context) {
	templateInfoID := c.Param("templateInfoId")
	if templateInfoID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing templateInfoId parameter", "")
		return
	}

	// Parse the rejection request to get the reason
	var rejectReq struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&rejectReq); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Invalid request body", err.Error())
		return
	}

	if err := h.storage.RejectDraft(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Rejection failed", err.Error())
		return
	}

	// Return rejection response with expected fields
	response := map[string]interface{}{
		"rejected": true,
		"reason":   rejectReq.Reason,
	}

	h.SendOK(c, response)
}

// ListTemplateInfos lists all approved TemplateInfo configurations
// GET /template-infos
func (h *TemplateInfoHandler) ListTemplateInfos(c *gin.Context) {
	resources, err := h.storage.List(c.Request.Context(), storage.ResourceTypeTemplateInfo)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to list TemplateInfos", err.Error())
		return
	}

	// Convert to TemplateInfoInfo format with status information
	var templateInfoInfos []models.TemplateInfoInfo
	for _, resource := range resources {
		if templateInfoData, ok := resource.(*models.TemplateInfoData); ok {
			templateInfoInfo := models.TemplateInfoInfo{
				TemplateInfoData:   templateInfoData,
				TemplateInfoStatus: &models.TemplateInfoStatus{Message: "Active"},
			}
			templateInfoInfos = append(templateInfoInfos, templateInfoInfo)
		}
	}

	// Return array directly as per OpenAPI spec
	h.SendOK(c, templateInfoInfos)
}

// GetTemplateInfo retrieves a specific approved TemplateInfo configuration
// GET /template-infos/{templateInfoId}
func (h *TemplateInfoHandler) GetTemplateInfo(c *gin.Context) {
	templateInfoID := c.Param("templateInfoId")
	if templateInfoID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing templateInfoId parameter", "")
		return
	}

	resource, err := h.storage.Get(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "TemplateInfo not found", err.Error())
		return
	}

	templateInfoData, ok := resource.(*models.TemplateInfoData)
	if !ok {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Invalid resource data type", "")
		return
	}

	// Return the resource data directly for individual GET requests
	h.SendOK(c, templateInfoData)
}

// DeleteTemplateInfo deletes an approved TemplateInfo configuration
// DELETE /template-infos/{templateInfoId}
func (h *TemplateInfoHandler) DeleteTemplateInfo(c *gin.Context) {
	templateInfoID := c.Param("templateInfoId")
	if templateInfoID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing templateInfoId parameter", "")
		return
	}

	// Get the resource first to have complete data for dependency validation
	resource, err := h.storage.Get(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "TemplateInfo not found", err.Error())
		return
	}

	// Check for dependencies before deletion
	if err := h.storage.ValidateDependencies(c.Request.Context(), storage.ResourceTypeTemplateInfo, resource); err != nil {
		h.SendConflict(c, models.ErrorCodeDependency, "Cannot delete TemplateInfo: it is referenced by FocomProvisioningRequest resources", err.Error())
		return
	}

	if err := h.storage.Delete(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID); err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "TemplateInfo not found", err.Error())
		return
	}

	h.SendAccepted(c, map[string]string{"message": "TemplateInfo deletion initiated"})
}

// GetRevisions lists all revisions for a TemplateInfo
// GET /template-infos/{templateInfoId}/revisions
func (h *TemplateInfoHandler) GetRevisions(c *gin.Context) {
	templateInfoID := c.Param("templateInfoId")
	if templateInfoID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing templateInfoId parameter", "")
		return
	}

	revisions, err := h.storage.GetRevisions(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "TemplateInfo revisions not found", err.Error())
		return
	}

	// Return array directly as per OpenAPI spec
	h.SendOK(c, revisions)
}

// GetRevision retrieves a specific revision of a TemplateInfo
// GET /template-infos/{templateInfoId}/revisions/{revisionId}
func (h *TemplateInfoHandler) GetRevision(c *gin.Context) {
	templateInfoID := c.Param("templateInfoId")
	revisionID := c.Param("revisionId")

	if templateInfoID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing templateInfoId parameter", "")
		return
	}
	if revisionID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing revisionId parameter", "")
		return
	}

	revision, err := h.storage.GetRevision(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID, revisionID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "TemplateInfo revision not found", err.Error())
		return
	}

	h.SendOK(c, revision)
}

// CreateDraftFromRevision creates a new draft from a specific revision
// POST /template-infos/{templateInfoId}/revisions/{revisionId}/draft
func (h *TemplateInfoHandler) CreateDraftFromRevision(c *gin.Context) {
	templateInfoID := c.Param("templateInfoId")
	revisionID := c.Param("revisionId")

	if templateInfoID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing templateInfoId parameter", "")
		return
	}
	if revisionID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing revisionId parameter", "")
		return
	}

	if err := h.storage.CreateDraftFromRevision(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID, revisionID); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Failed to create draft from revision", err.Error())
		return
	}

	// Get the newly created draft
	draft, err := h.storage.GetDraft(c.Request.Context(), storage.ResourceTypeTemplateInfo, templateInfoID)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to retrieve created draft", err.Error())
		return
	}

	h.SendCreated(c, draft)
}

// RegisterRoutes registers all TemplateInfo routes with the router
func (h *TemplateInfoHandler) RegisterRoutes(router gin.IRouter) {
	templateInfoGroup := router.Group("/template-infos")
	{
		// Draft management
		templateInfoGroup.POST("/draft", h.CreateDraft)
		templateInfoGroup.GET("/:templateInfoId/draft", h.GetDraft)
		templateInfoGroup.PATCH("/:templateInfoId/draft", h.UpdateDraft)
		templateInfoGroup.DELETE("/:templateInfoId/draft", h.DeleteDraft)
		templateInfoGroup.POST("/:templateInfoId/draft/validate", h.ValidateDraft)
		templateInfoGroup.POST("/:templateInfoId/draft/approve", h.ApproveDraft)
		templateInfoGroup.POST("/:templateInfoId/draft/reject", h.RejectDraft)

		// Approved resource management
		templateInfoGroup.GET("", h.ListTemplateInfos)
		templateInfoGroup.GET("/:templateInfoId", h.GetTemplateInfo)
		templateInfoGroup.DELETE("/:templateInfoId", h.DeleteTemplateInfo)

		// Revision management
		templateInfoGroup.GET("/:templateInfoId/revisions", h.GetRevisions)
		templateInfoGroup.GET("/:templateInfoId/revisions/:revisionId", h.GetRevision)
		templateInfoGroup.POST("/:templateInfoId/revisions/:revisionId/draft", h.CreateDraftFromRevision)
	}
}
