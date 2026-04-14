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

package services

import (
	"context"
	"fmt"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/integration"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/storage"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/validation"
)

// FocomProvisioningRequestService provides business logic for FocomProvisioningRequest resource management
type FocomProvisioningRequestService struct {
	*BaseService
	storage               storage.StorageInterface
	validator             validation.Validator
	integration           integration.OperatorIntegration
	earlySchemaValidation bool
}

// NewFocomProvisioningRequestService creates a new FocomProvisioningRequest service
func NewFocomProvisioningRequestService(
	storage storage.StorageInterface,
	validator validation.Validator,
	integration integration.OperatorIntegration,
) *FocomProvisioningRequestService {
	return &FocomProvisioningRequestService{
		BaseService: NewBaseService(),
		storage:     storage,
		validator:   validator,
		integration: integration,
	}
}

// SetEarlySchemaValidation enables or disables schema validation during CreateDraft and UpdateDraft
func (s *FocomProvisioningRequestService) SetEarlySchemaValidation(enabled bool) {
	s.earlySchemaValidation = enabled
}

// CreateDraft creates a new FocomProvisioningRequest draft
func (s *FocomProvisioningRequestService) CreateDraft(ctx context.Context, fpr *models.FocomProvisioningRequestData) (*models.FocomProvisioningRequestData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	// Set initial state and timestamps (ID is already set by NewFocomProvisioningRequestData)
	fpr.State = models.StateDraft
	fpr.UpdateTimestamp()

	// Early schema validation if enabled
	if s.earlySchemaValidation {
		if err := s.validateTemplateParametersEarly(ctx, fpr); err != nil {
			return nil, err
		}
	}

	// Create draft in storage
	if err := s.storage.CreateDraft(ctx, storage.ResourceTypeFocomProvisioningRequest, fpr); err != nil {
		return nil, fmt.Errorf("failed to create FocomProvisioningRequest draft: %w", err)
	}

	return fpr, nil
}

// GetDraft retrieves a FocomProvisioningRequest draft
func (s *FocomProvisioningRequestService) GetDraft(ctx context.Context, id string) (*models.FocomProvisioningRequestData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	draft, err := s.storage.GetDraft(ctx, storage.ResourceTypeFocomProvisioningRequest, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get FocomProvisioningRequest draft: %w", err)
	}

	fpr, ok := draft.(*models.FocomProvisioningRequestData)
	if !ok {
		return nil, fmt.Errorf("invalid FocomProvisioningRequest draft data type")
	}

	return fpr, nil
}

// UpdateDraft updates an existing FocomProvisioningRequest draft
func (s *FocomProvisioningRequestService) UpdateDraft(ctx context.Context, id string, updates *models.FocomProvisioningRequestData) (*models.FocomProvisioningRequestData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	// Get existing draft
	existing, err := s.GetDraft(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if draft can be updated (not in VALIDATED state)
	if existing.State == models.StateValidated {
		return nil, ErrInvalidState
	}

	// Update allowed fields (only if they are provided)
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.OCloudID != "" {
		existing.OCloudID = updates.OCloudID
	}
	if updates.OCloudNamespace != "" {
		existing.OCloudNamespace = updates.OCloudNamespace
	}
	if updates.TemplateName != "" {
		existing.TemplateName = updates.TemplateName
	}
	if updates.TemplateVersion != "" {
		existing.TemplateVersion = updates.TemplateVersion
	}
	if updates.TemplateParameters != nil {
		existing.TemplateParameters = updates.TemplateParameters
	}
	existing.UpdateTimestamp()

	// Early schema validation if enabled
	if s.earlySchemaValidation {
		if err := s.validateTemplateParametersEarly(ctx, existing); err != nil {
			return nil, err
		}
	}

	// Update draft in storage
	if err := s.storage.UpdateDraft(ctx, storage.ResourceTypeFocomProvisioningRequest, id, existing); err != nil {
		return nil, fmt.Errorf("failed to update FocomProvisioningRequest draft: %w", err)
	}

	return existing, nil
}

// ValidateDraft validates a FocomProvisioningRequest draft including dependency validation
func (s *FocomProvisioningRequestService) ValidateDraft(ctx context.Context, id string) (*models.ValidationResult, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	// Get draft
	draft, err := s.GetDraft(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate the draft (basic validation)
	result := s.validator.ValidateFocomProvisioningRequest(ctx, draft)

	// If basic validation fails, return early
	if !result.Success {
		return result, nil
	}

	// Get TemplateInfo to validate template parameters against schema
	templateInfo, err := s.getTemplateInfo(ctx, draft.TemplateName, draft.TemplateVersion)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to find TemplateInfo %s:%s - %v", draft.TemplateName, draft.TemplateVersion, err))
		result.Success = false
		return result, nil
	}

	// Hard fail if TemplateInfo is not in APPROVED state
	if templateInfo.State != models.StateApproved {
		result.Errors = append(result.Errors, fmt.Sprintf("TemplateInfo %s:%s is not in APPROVED state (current state: %s)", draft.TemplateName, draft.TemplateVersion, templateInfo.State))
		result.Success = false
		return result, nil
	}

	// Validate template parameters against the actual schema
	templateParamResult := s.validator.ValidateTemplateParameters(ctx, draft.TemplateParameters, templateInfo.TemplateParameterSchema)

	// Merge validation results
	result.Errors = append(result.Errors, templateParamResult.Errors...)
	result.Warnings = append(result.Warnings, templateParamResult.Warnings...)
	result.SchemaErrors = append(result.SchemaErrors, templateParamResult.SchemaErrors...)
	result.Success = result.Success && templateParamResult.Success

	// If validation succeeds, update state to VALIDATED using storage layer method
	if result.Success {
		if err := s.storage.ValidateDraft(ctx, storage.ResourceTypeFocomProvisioningRequest, id); err != nil {
			return nil, fmt.Errorf("failed to update draft state to VALIDATED: %w", err)
		}
	}

	return result, nil
}

// getTemplateInfo finds a TemplateInfo by name and version
func (s *FocomProvisioningRequestService) getTemplateInfo(ctx context.Context, templateName, templateVersion string) (*models.TemplateInfoData, error) {
	// List all TemplateInfos
	templateInfos, err := s.storage.List(ctx, storage.ResourceTypeTemplateInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to list TemplateInfos: %w", err)
	}

	// Find matching TemplateInfo
	for _, resource := range templateInfos {
		if templateInfo, ok := resource.(*models.TemplateInfoData); ok {
			if templateInfo.TemplateName == templateName && templateInfo.TemplateVersion == templateVersion {
				return templateInfo, nil
			}
		}
	}

	return nil, fmt.Errorf("TemplateInfo not found: %s:%s", templateName, templateVersion)
}

// validateTemplateParametersEarly performs schema validation during create/update when early validation is enabled.
// It looks up the TemplateInfo and validates the FPR's templateParameters against its schema.
func (s *FocomProvisioningRequestService) validateTemplateParametersEarly(ctx context.Context, fpr *models.FocomProvisioningRequestData) error {
	templateInfo, err := s.getTemplateInfo(ctx, fpr.TemplateName, fpr.TemplateVersion)
	if err != nil {
		return fmt.Errorf("early schema validation failed: TemplateInfo %s:%s not found", fpr.TemplateName, fpr.TemplateVersion)
	}

	if templateInfo.State != models.StateApproved {
		return fmt.Errorf("early schema validation failed: TemplateInfo %s:%s is not in APPROVED state (current state: %s)", fpr.TemplateName, fpr.TemplateVersion, templateInfo.State)
	}

	result := s.validator.ValidateTemplateParameters(ctx, fpr.TemplateParameters, templateInfo.TemplateParameterSchema)
	if !result.Success {
		return &EarlyValidationError{Errors: result.Errors, SchemaErrors: result.SchemaErrors}
	}

	return nil
}

// RejectDraft rejects a FocomProvisioningRequest draft and resets it to DRAFT state
func (s *FocomProvisioningRequestService) RejectDraft(ctx context.Context, id string) error {
	if err := s.ValidateContext(ctx); err != nil {
		return err
	}

	// Get draft
	draft, err := s.GetDraft(ctx, id)
	if err != nil {
		return err
	}

	// Reset state to DRAFT
	draft.SetState(models.StateDraft)

	if err := s.storage.UpdateDraft(ctx, storage.ResourceTypeFocomProvisioningRequest, id, draft); err != nil {
		return fmt.Errorf("failed to reject FocomProvisioningRequest draft: %w", err)
	}

	return nil
}

// ApproveDraft approves a FocomProvisioningRequest draft with dependency validation and creates the corresponding Kubernetes CR
func (s *FocomProvisioningRequestService) ApproveDraft(ctx context.Context, id string) (*models.FocomProvisioningRequestData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	// Get draft
	draft, err := s.GetDraft(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if draft is in VALIDATED state
	if draft.State != models.StateValidated {
		return nil, ErrInvalidState
	}

	// Validate dependencies before approval (OCloud and TemplateInfo must exist)
	if err := s.validateDependenciesForApproval(ctx, draft); err != nil {
		return nil, fmt.Errorf("dependency validation failed: %w", err)
	}

	// Approve draft in storage (this creates the approved revision and updates state)
	if err := s.storage.ApproveDraft(ctx, storage.ResourceTypeFocomProvisioningRequest, id); err != nil {
		return nil, fmt.Errorf("failed to approve FocomProvisioningRequest draft: %w", err)
	}

	// Get the approved resource (now it should have the correct state)
	approvedResource, err := s.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve approved resource: %w", err)
	}

	// NOTE: CR creation is now handled by ConfigSync (Git → Kubernetes sync)
	// ConfigSync watches the Git repository and automatically creates CRs when
	// PackageRevisions are Published. This provides true GitOps with Git as source of truth.
	//
	// The following code is commented out to avoid duplicate CR creation:
	//
	// // Create Kubernetes custom resource using operator integration
	// if err := s.integration.CreateFocomProvisioningRequestCR(ctx, approvedResource); err != nil {
	// 	return nil, fmt.Errorf("failed to create FocomProvisioningRequest CR: %w", err)
	// }

	return approvedResource, nil
}

// validateDependenciesForApproval validates that required OCloud and TemplateInfo resources exist
func (s *FocomProvisioningRequestService) validateDependenciesForApproval(ctx context.Context, fpr *models.FocomProvisioningRequestData) error {
	// Check if referenced OCloud exists
	_, err := s.storage.Get(ctx, storage.ResourceTypeOCloud, fpr.OCloudID)
	if err != nil {
		return fmt.Errorf("referenced OCloud %s not found: %w", fpr.OCloudID, ErrDependencyNotFound)
	}

	// Check if referenced TemplateInfo exists by searching for matching template name and version
	templateInfos, err := s.storage.List(ctx, storage.ResourceTypeTemplateInfo)
	if err != nil {
		return fmt.Errorf("failed to list TemplateInfos for dependency validation: %w", err)
	}

	templateFound := false
	for _, resource := range templateInfos {
		templateInfo, ok := resource.(*models.TemplateInfoData)
		if !ok {
			continue
		}
		if templateInfo.TemplateName == fpr.TemplateName && templateInfo.TemplateVersion == fpr.TemplateVersion {
			templateFound = true
			break
		}
	}

	if !templateFound {
		return fmt.Errorf("referenced TemplateInfo %s:%s not found: %w", fpr.TemplateName, fpr.TemplateVersion, ErrDependencyNotFound)
	}

	return nil
}

// DeleteDraft deletes a FocomProvisioningRequest draft
func (s *FocomProvisioningRequestService) DeleteDraft(ctx context.Context, id string) error {
	if err := s.ValidateContext(ctx); err != nil {
		return err
	}

	if err := s.storage.DeleteDraft(ctx, storage.ResourceTypeFocomProvisioningRequest, id); err != nil {
		return fmt.Errorf("failed to delete FocomProvisioningRequest draft: %w", err)
	}

	return nil
}

// Get retrieves an approved FocomProvisioningRequest resource
func (s *FocomProvisioningRequestService) Get(ctx context.Context, id string) (*models.FocomProvisioningRequestData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	resource, err := s.storage.Get(ctx, storage.ResourceTypeFocomProvisioningRequest, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get FocomProvisioningRequest: %w", err)
	}

	fpr, ok := resource.(*models.FocomProvisioningRequestData)
	if !ok {
		return nil, fmt.Errorf("invalid FocomProvisioningRequest data type")
	}

	return fpr, nil
}

// List retrieves all approved FocomProvisioningRequest resources
func (s *FocomProvisioningRequestService) List(ctx context.Context) ([]*models.FocomProvisioningRequestData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	resources, err := s.storage.List(ctx, storage.ResourceTypeFocomProvisioningRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to list FocomProvisioningRequests: %w", err)
	}

	fprs := make([]*models.FocomProvisioningRequestData, len(resources))
	for i, resource := range resources {
		fpr, ok := resource.(*models.FocomProvisioningRequestData)
		if !ok {
			return nil, fmt.Errorf("invalid FocomProvisioningRequest data type at index %d", i)
		}
		fprs[i] = fpr
	}

	return fprs, nil
}

// GetRevisions retrieves all revisions for a FocomProvisioningRequest resource
func (s *FocomProvisioningRequestService) GetRevisions(ctx context.Context, id string) ([]*models.FocomProvisioningRequestData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	revisions, err := s.storage.GetRevisions(ctx, storage.ResourceTypeFocomProvisioningRequest, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get FocomProvisioningRequest revisions: %w", err)
	}

	fprs := make([]*models.FocomProvisioningRequestData, len(revisions))
	for i, revision := range revisions {
		fpr, ok := revision.(*models.FocomProvisioningRequestData)
		if !ok {
			return nil, fmt.Errorf("invalid FocomProvisioningRequest revision data type at index %d", i)
		}
		fprs[i] = fpr
	}

	return fprs, nil
}

// GetRevision retrieves a specific revision of a FocomProvisioningRequest resource
func (s *FocomProvisioningRequestService) GetRevision(ctx context.Context, id, revisionId string) (*models.FocomProvisioningRequestData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	revision, err := s.storage.GetRevision(ctx, storage.ResourceTypeFocomProvisioningRequest, id, revisionId)
	if err != nil {
		return nil, fmt.Errorf("failed to get FocomProvisioningRequest revision: %w", err)
	}

	fpr, ok := revision.(*models.FocomProvisioningRequestData)
	if !ok {
		return nil, fmt.Errorf("invalid FocomProvisioningRequest revision data type")
	}

	return fpr, nil
}

// CreateDraftFromRevision creates a new draft based on an existing revision
func (s *FocomProvisioningRequestService) CreateDraftFromRevision(ctx context.Context, id, revisionId string) (*models.FocomProvisioningRequestData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	// Create draft from revision in storage
	if err := s.storage.CreateDraftFromRevision(ctx, storage.ResourceTypeFocomProvisioningRequest, id, revisionId); err != nil {
		return nil, fmt.Errorf("failed to create FocomProvisioningRequest draft from revision: %w", err)
	}

	// Get the newly created draft
	return s.GetDraft(ctx, id)
}

// Delete deletes an approved FocomProvisioningRequest resource (triggers decommissioning)
func (s *FocomProvisioningRequestService) Delete(ctx context.Context, id string) error {
	if err := s.ValidateContext(ctx); err != nil {
		return err
	}

	// Get the resource first to ensure it exists
	_, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Delete the resource (this should trigger decommissioning in the operator)
	if err := s.storage.Delete(ctx, storage.ResourceTypeFocomProvisioningRequest, id); err != nil {
		return fmt.Errorf("failed to delete FocomProvisioningRequest: %w", err)
	}

	return nil
}
