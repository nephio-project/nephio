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

// TemplateInfoService provides business logic for TemplateInfo resource management
type TemplateInfoService struct {
	*BaseService
	storage     storage.StorageInterface
	validator   validation.Validator
	integration integration.OperatorIntegration
}

// NewTemplateInfoService creates a new TemplateInfo service
func NewTemplateInfoService(
	storage storage.StorageInterface,
	validator validation.Validator,
	integration integration.OperatorIntegration,
) *TemplateInfoService {
	return &TemplateInfoService{
		BaseService: NewBaseService(),
		storage:     storage,
		validator:   validator,
		integration: integration,
	}
}

// CreateDraft creates a new TemplateInfo draft
func (s *TemplateInfoService) CreateDraft(ctx context.Context, templateInfo *models.TemplateInfoData) (*models.TemplateInfoData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	// Set initial state and timestamps (ID is already set by NewTemplateInfoData)
	templateInfo.State = models.StateDraft
	templateInfo.UpdateTimestamp()

	// Create draft in storage
	if err := s.storage.CreateDraft(ctx, storage.ResourceTypeTemplateInfo, templateInfo); err != nil {
		return nil, fmt.Errorf("failed to create TemplateInfo draft: %w", err)
	}

	return templateInfo, nil
}

// GetDraft retrieves a TemplateInfo draft
func (s *TemplateInfoService) GetDraft(ctx context.Context, id string) (*models.TemplateInfoData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	draft, err := s.storage.GetDraft(ctx, storage.ResourceTypeTemplateInfo, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get TemplateInfo draft: %w", err)
	}

	templateInfo, ok := draft.(*models.TemplateInfoData)
	if !ok {
		return nil, fmt.Errorf("invalid TemplateInfo draft data type")
	}

	return templateInfo, nil
}

// UpdateDraft updates an existing TemplateInfo draft
func (s *TemplateInfoService) UpdateDraft(ctx context.Context, id string, updates *models.TemplateInfoData) (*models.TemplateInfoData, error) {
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

	// Update allowed fields
	existing.Name = updates.Name
	existing.Description = updates.Description
	existing.TemplateName = updates.TemplateName
	existing.TemplateVersion = updates.TemplateVersion
	existing.TemplateParameterSchema = updates.TemplateParameterSchema
	existing.UpdateTimestamp()

	// Update draft in storage
	if err := s.storage.UpdateDraft(ctx, storage.ResourceTypeTemplateInfo, id, existing); err != nil {
		return nil, fmt.Errorf("failed to update TemplateInfo draft: %w", err)
	}

	return existing, nil
}

// ValidateDraft validates a TemplateInfo draft including template parameter schema validation
func (s *TemplateInfoService) ValidateDraft(ctx context.Context, id string) (*models.ValidationResult, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	// Get draft
	draft, err := s.GetDraft(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate the draft (includes template parameter schema validation)
	result := s.validator.ValidateTemplateInfo(ctx, draft)

	// If validation succeeds, update state to VALIDATED
	if result.Success {
		draft.SetState(models.StateValidated)
		if err := s.storage.UpdateDraft(ctx, storage.ResourceTypeTemplateInfo, id, draft); err != nil {
			return nil, fmt.Errorf("failed to update draft state: %w", err)
		}
	}

	return result, nil
}

// RejectDraft rejects a TemplateInfo draft and resets it to DRAFT state
func (s *TemplateInfoService) RejectDraft(ctx context.Context, id string) error {
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

	if err := s.storage.UpdateDraft(ctx, storage.ResourceTypeTemplateInfo, id, draft); err != nil {
		return fmt.Errorf("failed to reject TemplateInfo draft: %w", err)
	}

	return nil
}

// ApproveDraft approves a TemplateInfo draft and creates the corresponding Kubernetes CR
func (s *TemplateInfoService) ApproveDraft(ctx context.Context, id string) (*models.TemplateInfoData, error) {
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

	// Approve draft in storage (this creates the approved revision and updates state)
	if err := s.storage.ApproveDraft(ctx, storage.ResourceTypeTemplateInfo, id); err != nil {
		return nil, fmt.Errorf("failed to approve TemplateInfo draft: %w", err)
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
	// if err := s.integration.CreateTemplateInfoCR(ctx, approvedResource); err != nil {
	// 	return nil, fmt.Errorf("failed to create TemplateInfo CR: %w", err)
	// }

	return approvedResource, nil
}

// DeleteDraft deletes a TemplateInfo draft
func (s *TemplateInfoService) DeleteDraft(ctx context.Context, id string) error {
	if err := s.ValidateContext(ctx); err != nil {
		return err
	}

	if err := s.storage.DeleteDraft(ctx, storage.ResourceTypeTemplateInfo, id); err != nil {
		return fmt.Errorf("failed to delete TemplateInfo draft: %w", err)
	}

	return nil
}

// Get retrieves an approved TemplateInfo resource
func (s *TemplateInfoService) Get(ctx context.Context, id string) (*models.TemplateInfoData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	resource, err := s.storage.Get(ctx, storage.ResourceTypeTemplateInfo, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get TemplateInfo: %w", err)
	}

	templateInfo, ok := resource.(*models.TemplateInfoData)
	if !ok {
		return nil, fmt.Errorf("invalid TemplateInfo data type")
	}

	return templateInfo, nil
}

// List retrieves all approved TemplateInfo resources
func (s *TemplateInfoService) List(ctx context.Context) ([]*models.TemplateInfoData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	resources, err := s.storage.List(ctx, storage.ResourceTypeTemplateInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to list TemplateInfos: %w", err)
	}

	templateInfos := make([]*models.TemplateInfoData, len(resources))
	for i, resource := range resources {
		templateInfo, ok := resource.(*models.TemplateInfoData)
		if !ok {
			return nil, fmt.Errorf("invalid TemplateInfo data type at index %d", i)
		}
		templateInfos[i] = templateInfo
	}

	return templateInfos, nil
}

// GetRevisions retrieves all revisions for a TemplateInfo resource
func (s *TemplateInfoService) GetRevisions(ctx context.Context, id string) ([]*models.TemplateInfoData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	revisions, err := s.storage.GetRevisions(ctx, storage.ResourceTypeTemplateInfo, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get TemplateInfo revisions: %w", err)
	}

	templateInfos := make([]*models.TemplateInfoData, len(revisions))
	for i, revision := range revisions {
		templateInfo, ok := revision.(*models.TemplateInfoData)
		if !ok {
			return nil, fmt.Errorf("invalid TemplateInfo revision data type at index %d", i)
		}
		templateInfos[i] = templateInfo
	}

	return templateInfos, nil
}

// GetRevision retrieves a specific revision of a TemplateInfo resource
func (s *TemplateInfoService) GetRevision(ctx context.Context, id, revisionId string) (*models.TemplateInfoData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	revision, err := s.storage.GetRevision(ctx, storage.ResourceTypeTemplateInfo, id, revisionId)
	if err != nil {
		return nil, fmt.Errorf("failed to get TemplateInfo revision: %w", err)
	}

	templateInfo, ok := revision.(*models.TemplateInfoData)
	if !ok {
		return nil, fmt.Errorf("invalid TemplateInfo revision data type")
	}

	return templateInfo, nil
}

// CreateDraftFromRevision creates a new draft based on an existing revision
func (s *TemplateInfoService) CreateDraftFromRevision(ctx context.Context, id, revisionId string) (*models.TemplateInfoData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	// Create draft from revision in storage
	if err := s.storage.CreateDraftFromRevision(ctx, storage.ResourceTypeTemplateInfo, id, revisionId); err != nil {
		return nil, fmt.Errorf("failed to create TemplateInfo draft from revision: %w", err)
	}

	// Get the newly created draft
	return s.GetDraft(ctx, id)
}

// Delete deletes an approved TemplateInfo resource (with dependency validation)
func (s *TemplateInfoService) Delete(ctx context.Context, id string) error {
	if err := s.ValidateContext(ctx); err != nil {
		return err
	}

	// Get the resource to validate dependencies
	templateInfo, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Validate dependencies before deletion
	if err := s.storage.ValidateDependencies(ctx, storage.ResourceTypeTemplateInfo, templateInfo); err != nil {
		return fmt.Errorf("cannot delete TemplateInfo due to dependencies: %w", err)
	}

	if err := s.storage.Delete(ctx, storage.ResourceTypeTemplateInfo, id); err != nil {
		return fmt.Errorf("failed to delete TemplateInfo: %w", err)
	}

	return nil
}
