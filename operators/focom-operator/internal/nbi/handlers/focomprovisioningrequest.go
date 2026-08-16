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
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/integration"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/services"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/storage"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/validation"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FocomProvisioningRequestHandler handles HTTP requests for FocomProvisioningRequest resources
type FocomProvisioningRequestHandler struct {
	BaseHandler
	storage          storage.StorageInterface
	service          *services.FocomProvisioningRequestService
	integration      integration.OperatorIntegration // Optional operator integration
	defaultNamespace string                          // Default namespace from config
}

// NewFocomProvisioningRequestHandler creates a new FocomProvisioningRequest handler
func NewFocomProvisioningRequestHandler(storage storage.StorageInterface, defaultNamespace string) *FocomProvisioningRequestHandler {
	return NewFocomProvisioningRequestHandlerWithOptions(storage, defaultNamespace, false)
}

// NewFocomProvisioningRequestHandlerWithOptions creates a new FPR handler with configuration options
func NewFocomProvisioningRequestHandlerWithOptions(storage storage.StorageInterface, defaultNamespace string, earlySchemaValidation bool) *FocomProvisioningRequestHandler {
	// Create validators
	schemaValidator := validation.NewJSONSchemaValidator()
	businessValidator := validation.NewBusinessRuleValidator()
	validator := validation.NewValidationService(schemaValidator, businessValidator)

	// Create service with proper validators and early validation flag
	service := services.NewFocomProvisioningRequestService(storage, validator, nil)
	service.SetEarlySchemaValidation(earlySchemaValidation)

	return &FocomProvisioningRequestHandler{
		BaseHandler:      *NewBaseHandler(),
		storage:          storage,
		service:          service,
		integration:      nil,
		defaultNamespace: defaultNamespace,
	}
}

// NewFocomProvisioningRequestHandlerWithIntegration creates a new FPR handler with operator integration
func NewFocomProvisioningRequestHandlerWithIntegration(storage storage.StorageInterface, integration integration.OperatorIntegration, defaultNamespace string, earlySchemaValidation bool) *FocomProvisioningRequestHandler {
	// Create validators
	schemaValidator := validation.NewJSONSchemaValidator()
	businessValidator := validation.NewBusinessRuleValidator()
	validator := validation.NewValidationService(schemaValidator, businessValidator)

	// Create service with proper validators and early validation flag
	service := services.NewFocomProvisioningRequestService(storage, validator, integration)
	service.SetEarlySchemaValidation(earlySchemaValidation)

	return &FocomProvisioningRequestHandler{
		BaseHandler:      *NewBaseHandler(),
		storage:          storage,
		service:          service,
		integration:      integration,
		defaultNamespace: defaultNamespace,
	}
}

// validateTemplateParameters is now handled by the validation service using the actual TemplateInfo schema
// This method is no longer needed as validation is done in the service layer

// validateDependencies validates that referenced OCloud and TemplateInfo exist
func (h *FocomProvisioningRequestHandler) validateDependencies(ctx *gin.Context, oCloudID, oCloudNamespace, templateName, templateVersion string) error {
	// Check if OCloud exists
	_, err := h.storage.Get(ctx.Request.Context(), storage.ResourceTypeOCloud, oCloudID)
	if err != nil {
		return &models.DependencyError{
			ResourceType:  models.ResourceTypeFocomProvisioningRequest,
			ResourceID:    "",
			DependentType: models.ResourceTypeOCloud,
			DependentID:   oCloudID,
			Message:       "Referenced OCloud does not exist",
		}
	}

	// Check if TemplateInfo exists by searching for matching template name and version
	templateInfos, err := h.storage.List(ctx.Request.Context(), storage.ResourceTypeTemplateInfo)
	if err != nil {
		return err
	}

	templateFound := false
	for _, resource := range templateInfos {
		if templateInfoData, ok := resource.(*models.TemplateInfoData); ok {
			if templateInfoData.TemplateName == templateName && templateInfoData.TemplateVersion == templateVersion {
				templateFound = true
				break
			}
		}
	}

	if !templateFound {
		return &models.DependencyError{
			ResourceType:  models.ResourceTypeFocomProvisioningRequest,
			ResourceID:    "",
			DependentType: models.ResourceTypeTemplateInfo,
			DependentID:   templateName + ":" + templateVersion,
			Message:       "Referenced TemplateInfo does not exist",
		}
	}

	return nil
}

// CreateDraft creates a new FocomProvisioningRequest draft
// POST /focom-provisioning-requests/draft
func (h *FocomProvisioningRequestHandler) CreateDraft(c *gin.Context) {
	var req struct {
		Namespace          string                 `json:"namespace"` // DEPRECATED: Use FOCOM_NAMESPACE environment variable instead
		Name               string                 `json:"name" binding:"required"`
		Description        string                 `json:"description" binding:"required"`
		OCloudID           string                 `json:"oCloudId" binding:"required"`
		OCloudNamespace    string                 `json:"oCloudNamespace"` // DEPRECATED: Use FOCOM_NAMESPACE environment variable instead
		TemplateName       string                 `json:"templateName" binding:"required"`
		TemplateVersion    string                 `json:"templateVersion" binding:"required"`
		TemplateParameters map[string]interface{} `json:"templateParameters" binding:"required"`
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
		log.Log.WithName("fpr-handler").Info(
			"DEPRECATION WARNING: 'namespace' field in request body is deprecated. Use FOCOM_NAMESPACE environment variable instead.",
			"resource", "FocomProvisioningRequest",
			"name", req.Name,
			"namespace", req.Namespace,
		)
	}

	// Determine oCloudNamespace: request body (deprecated) takes precedence over default for backward compatibility
	oCloudNamespace := h.defaultNamespace
	if req.OCloudNamespace != "" {
		oCloudNamespace = req.OCloudNamespace
		// Log deprecation warning
		log.Log.WithName("fpr-handler").Info(
			"DEPRECATION WARNING: 'oCloudNamespace' field in request body is deprecated. Use FOCOM_NAMESPACE environment variable instead.",
			"resource", "FocomProvisioningRequest",
			"name", req.Name,
			"oCloudNamespace", req.OCloudNamespace,
		)
	}

	// Create new FocomProvisioningRequest data
	fprData := models.NewFocomProvisioningRequestData(
		namespace,
		req.Name,
		req.Description,
		req.OCloudID,
		oCloudNamespace,
		req.TemplateName,
		req.TemplateVersion,
		req.TemplateParameters,
	)

	// Use service to create the draft
	createdFPR, err := h.service.CreateDraft(c.Request.Context(), fprData)
	if err != nil {
		var earlyErr *services.EarlyValidationError
		if errors.As(err, &earlyErr) {
			c.JSON(400, gin.H{
				"error":        "Schema validation failed",
				"errors":       earlyErr.Errors,
				"schemaErrors": earlyErr.SchemaErrors,
			})
			return
		}
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to create FocomProvisioningRequest draft", err.Error())
		return
	}

	h.SendCreated(c, createdFPR)
}

// GetDraft retrieves a FocomProvisioningRequest draft
// GET /focom-provisioning-requests/{provisioningRequestId}/draft
func (h *FocomProvisioningRequestHandler) GetDraft(c *gin.Context) {
	provisioningRequestID := c.Param("provisioningRequestId")
	if provisioningRequestID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing provisioningRequestId parameter", "")
		return
	}

	draft, err := h.service.GetDraft(c.Request.Context(), provisioningRequestID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "FocomProvisioningRequest draft not found", err.Error())
		return
	}

	h.SendOK(c, draft)
}

// UpdateDraft updates a FocomProvisioningRequest draft
// PATCH /focom-provisioning-requests/{provisioningRequestId}/draft
func (h *FocomProvisioningRequestHandler) UpdateDraft(c *gin.Context) {
	provisioningRequestID := c.Param("provisioningRequestId")
	if provisioningRequestID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing provisioningRequestId parameter", "")
		return
	}

	var updateReq models.FocomProvisioningRequestDataUpdate
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Invalid request body", err.Error())
		return
	}

	// Create update data - handle pointer fields
	updateData := &models.FocomProvisioningRequestData{}
	if updateReq.Name != nil {
		updateData.Name = *updateReq.Name
	}
	if updateReq.Description != nil {
		updateData.Description = *updateReq.Description
	}
	if updateReq.TemplateName != nil {
		updateData.TemplateName = *updateReq.TemplateName
	}
	if updateReq.TemplateVersion != nil {
		updateData.TemplateVersion = *updateReq.TemplateVersion
	}
	if updateReq.TemplateParameters != nil {
		updateData.TemplateParameters = updateReq.TemplateParameters
	}

	// Use service to update the draft
	updatedFPR, err := h.service.UpdateDraft(c.Request.Context(), provisioningRequestID, updateData)
	if err != nil {
		var earlyErr *services.EarlyValidationError
		if errors.As(err, &earlyErr) {
			c.JSON(400, gin.H{
				"error":        "Schema validation failed",
				"errors":       earlyErr.Errors,
				"schemaErrors": earlyErr.SchemaErrors,
			})
			return
		}
		// Check if it's a not found error by unwrapping the error chain
		var storageErr *storage.StorageError
		if errors.As(err, &storageErr) && storageErr.Code == storage.ErrorCodeNotFound {
			h.SendNotFound(c, models.ErrorCodeNotFound, "FocomProvisioningRequest draft not found", err.Error())
		} else {
			h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to update FocomProvisioningRequest draft", err.Error())
		}
		return
	}

	h.SendOK(c, updatedFPR)
}

// DeleteDraft deletes a FocomProvisioningRequest draft
// DELETE /focom-provisioning-requests/{provisioningRequestId}/draft
func (h *FocomProvisioningRequestHandler) DeleteDraft(c *gin.Context) {
	provisioningRequestID := c.Param("provisioningRequestId")
	if provisioningRequestID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing provisioningRequestId parameter", "")
		return
	}

	if err := h.service.DeleteDraft(c.Request.Context(), provisioningRequestID); err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "FocomProvisioningRequest draft not found", err.Error())
		return
	}

	h.SendNoContent(c)
}

// ValidateDraft validates a FocomProvisioningRequest draft
// POST /focom-provisioning-requests/{provisioningRequestId}/draft/validate
func (h *FocomProvisioningRequestHandler) ValidateDraft(c *gin.Context) {
	provisioningRequestID := c.Param("provisioningRequestId")
	if provisioningRequestID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing provisioningRequestId parameter", "")
		return
	}

	// Use service layer validation which includes template parameter validation against actual TemplateInfo schema
	validationResult, err := h.service.ValidateDraft(c.Request.Context(), provisioningRequestID)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Validation failed", err.Error())
		return
	}

	// If validation failed, return error response
	if !validationResult.Success {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Template parameters validation failed", validationResult.Errors[0])
		return
	}

	// Get the validated draft to return
	validatedDraft, err := h.service.GetDraft(c.Request.Context(), provisioningRequestID)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to retrieve validated draft", err.Error())
		return
	}

	response := map[string]interface{}{
		"validationResult": validationResult,
		"draft":            validatedDraft,
	}

	h.SendOK(c, response)
}

// ApproveDraft approves a FocomProvisioningRequest draft
// POST /focom-provisioning-requests/{provisioningRequestId}/draft/approve
func (h *FocomProvisioningRequestHandler) ApproveDraft(c *gin.Context) {
	provisioningRequestID := c.Param("provisioningRequestId")
	if provisioningRequestID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing provisioningRequestId parameter", "")
		return
	}

	// Get the draft first to perform dependency validation before approval
	draft, err := h.storage.GetDraft(c.Request.Context(), storage.ResourceTypeFocomProvisioningRequest, provisioningRequestID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "FocomProvisioningRequest draft not found", err.Error())
		return
	}

	fprData, ok := draft.(*models.FocomProvisioningRequestData)
	if !ok {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Invalid draft data type", "")
		return
	}

	// Validate dependencies before approval
	if err := h.validateDependencies(c, fprData.OCloudID, fprData.OCloudNamespace, fprData.TemplateName, fprData.TemplateVersion); err != nil {
		if depErr, ok := err.(*models.DependencyError); ok {
			// Return detailed error message for dependency failures
			errorMsg := depErr.Message
			switch depErr.DependentType {
			case models.ResourceTypeOCloud:
				errorMsg = "Cannot approve due to missing dependencies: OCloud " + depErr.DependentID + " not found"
			case models.ResourceTypeTemplateInfo:
				errorMsg = "Cannot approve due to missing dependencies: TemplateInfo " + depErr.DependentID + " version not found"
			}
			h.SendBadRequest(c, models.ErrorCodeDependency, errorMsg, err.Error())
		} else {
			h.SendBadRequest(c, models.ErrorCodeDependency, "Cannot approve due to missing dependencies", err.Error())
		}
		return
	}

	if err := h.storage.ApproveDraft(c.Request.Context(), storage.ResourceTypeFocomProvisioningRequest, provisioningRequestID); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Approval failed", err.Error())
		return
	}

	// Get the approved resource
	approvedResource, err := h.storage.Get(c.Request.Context(), storage.ResourceTypeFocomProvisioningRequest, provisioningRequestID)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to retrieve approved resource", err.Error())
		return
	}

	// Convert to FPR data for response and CR creation
	approvedFprData, ok := approvedResource.(*models.FocomProvisioningRequestData)
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
	// 	if err := h.integration.CreateFocomProvisioningRequestCR(c.Request.Context(), approvedFprData); err != nil {
	// 		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to create FocomProvisioningRequest CR", err.Error())
	// 		return
	// 	}
	// }

	// Return approval response with expected fields
	response := map[string]interface{}{
		"approved":    true,
		"approvedAt":  approvedFprData.UpdatedAt,
		"revisionId":  approvedFprData.RevisionID,
		"id":          approvedFprData.ID,
		"name":        approvedFprData.Name,
		"description": approvedFprData.Description,
		"state":       approvedFprData.State,
	}

	h.SendOK(c, response)
}

// RejectDraft rejects a FocomProvisioningRequest draft
// POST /focom-provisioning-requests/{provisioningRequestId}/draft/reject
func (h *FocomProvisioningRequestHandler) RejectDraft(c *gin.Context) {
	provisioningRequestID := c.Param("provisioningRequestId")
	if provisioningRequestID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing provisioningRequestId parameter", "")
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

	if err := h.storage.RejectDraft(c.Request.Context(), storage.ResourceTypeFocomProvisioningRequest, provisioningRequestID); err != nil {
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

// ListFocomProvisioningRequests lists all approved FocomProvisioningRequest configurations
// GET /focom-provisioning-requests
func (h *FocomProvisioningRequestHandler) ListFocomProvisioningRequests(c *gin.Context) {
	resources, err := h.storage.List(c.Request.Context(), storage.ResourceTypeFocomProvisioningRequest)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to list FocomProvisioningRequests", err.Error())
		return
	}

	// Convert to the format expected by tests (direct resource data)
	var fprList []interface{}
	for _, resource := range resources {
		if fprData, ok := resource.(*models.FocomProvisioningRequestData); ok {
			fprList = append(fprList, fprData)
		}
	}

	// Return array directly as per OpenAPI spec
	h.SendOK(c, fprList)
}

// GetFocomProvisioningRequest retrieves a specific approved FocomProvisioningRequest configuration
// GET /focom-provisioning-requests/{provisioningRequestId}
func (h *FocomProvisioningRequestHandler) GetFocomProvisioningRequest(c *gin.Context) {
	provisioningRequestID := c.Param("provisioningRequestId")
	if provisioningRequestID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing provisioningRequestId parameter", "")
		return
	}

	resource, err := h.storage.Get(c.Request.Context(), storage.ResourceTypeFocomProvisioningRequest, provisioningRequestID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "FocomProvisioningRequest not found", err.Error())
		return
	}

	fprData, ok := resource.(*models.FocomProvisioningRequestData)
	if !ok {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Invalid resource data type", "")
		return
	}

	// Return the resource data directly for individual GET requests
	h.SendOK(c, fprData)
}

// DeleteFocomProvisioningRequest deletes an approved FocomProvisioningRequest configuration
// DELETE /focom-provisioning-requests/{provisioningRequestId}
func (h *FocomProvisioningRequestHandler) DeleteFocomProvisioningRequest(c *gin.Context) {
	provisioningRequestID := c.Param("provisioningRequestId")
	if provisioningRequestID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing provisioningRequestId parameter", "")
		return
	}

	if err := h.storage.Delete(c.Request.Context(), storage.ResourceTypeFocomProvisioningRequest, provisioningRequestID); err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "FocomProvisioningRequest not found", err.Error())
		return
	}

	h.SendAccepted(c, map[string]string{"message": "FocomProvisioningRequest decommissioning initiated"})
}

// GetRevisions lists all revisions for a FocomProvisioningRequest
// GET /focom-provisioning-requests/{provisioningRequestId}/revisions
func (h *FocomProvisioningRequestHandler) GetRevisions(c *gin.Context) {
	provisioningRequestID := c.Param("provisioningRequestId")
	if provisioningRequestID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing provisioningRequestId parameter", "")
		return
	}

	revisions, err := h.storage.GetRevisions(c.Request.Context(), storage.ResourceTypeFocomProvisioningRequest, provisioningRequestID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "FocomProvisioningRequest revisions not found", err.Error())
		return
	}

	// Return array directly as per OpenAPI spec
	h.SendOK(c, revisions)
}

// GetRevision retrieves a specific revision of a FocomProvisioningRequest
// GET /focom-provisioning-requests/{provisioningRequestId}/revisions/{revisionId}
func (h *FocomProvisioningRequestHandler) GetRevision(c *gin.Context) {
	provisioningRequestID := c.Param("provisioningRequestId")
	revisionID := c.Param("revisionId")

	if provisioningRequestID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing provisioningRequestId parameter", "")
		return
	}
	if revisionID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing revisionId parameter", "")
		return
	}

	revision, err := h.storage.GetRevision(c.Request.Context(), storage.ResourceTypeFocomProvisioningRequest, provisioningRequestID, revisionID)
	if err != nil {
		h.SendNotFound(c, models.ErrorCodeNotFound, "FocomProvisioningRequest revision not found", err.Error())
		return
	}

	h.SendOK(c, revision)
}

// CreateDraftFromRevision creates a new draft from a specific revision
// POST /focom-provisioning-requests/{provisioningRequestId}/revisions/{revisionId}/draft
func (h *FocomProvisioningRequestHandler) CreateDraftFromRevision(c *gin.Context) {
	provisioningRequestID := c.Param("provisioningRequestId")
	revisionID := c.Param("revisionId")

	if provisioningRequestID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing provisioningRequestId parameter", "")
		return
	}
	if revisionID == "" {
		h.SendBadRequest(c, models.ErrorCodeBadRequest, "Missing revisionId parameter", "")
		return
	}

	if err := h.storage.CreateDraftFromRevision(c.Request.Context(), storage.ResourceTypeFocomProvisioningRequest, provisioningRequestID, revisionID); err != nil {
		h.SendBadRequest(c, models.ErrorCodeValidation, "Failed to create draft from revision", err.Error())
		return
	}

	// Get the newly created draft
	draft, err := h.storage.GetDraft(c.Request.Context(), storage.ResourceTypeFocomProvisioningRequest, provisioningRequestID)
	if err != nil {
		h.SendInternalError(c, models.ErrorCodeInternalError, "Failed to retrieve created draft", err.Error())
		return
	}

	h.SendCreated(c, draft)
}

// RegisterRoutes registers all FocomProvisioningRequest routes with the router
func (h *FocomProvisioningRequestHandler) RegisterRoutes(router gin.IRouter) {
	fprGroup := router.Group("/focom-provisioning-requests")
	{
		// Draft management
		fprGroup.POST("/draft", h.CreateDraft)
		fprGroup.GET("/:provisioningRequestId/draft", h.GetDraft)
		fprGroup.PATCH("/:provisioningRequestId/draft", h.UpdateDraft)
		fprGroup.DELETE("/:provisioningRequestId/draft", h.DeleteDraft)
		fprGroup.POST("/:provisioningRequestId/draft/validate", h.ValidateDraft)
		fprGroup.POST("/:provisioningRequestId/draft/approve", h.ApproveDraft)
		fprGroup.POST("/:provisioningRequestId/draft/reject", h.RejectDraft)

		// Approved resource management
		fprGroup.GET("", h.ListFocomProvisioningRequests)
		fprGroup.GET("/:provisioningRequestId", h.GetFocomProvisioningRequest)
		fprGroup.DELETE("/:provisioningRequestId", h.DeleteFocomProvisioningRequest)

		// Revision management
		fprGroup.GET("/:provisioningRequestId/revisions", h.GetRevisions)
		fprGroup.GET("/:provisioningRequestId/revisions/:revisionId", h.GetRevision)
		fprGroup.POST("/:provisioningRequestId/revisions/:revisionId/draft", h.CreateDraftFromRevision)
	}
}
