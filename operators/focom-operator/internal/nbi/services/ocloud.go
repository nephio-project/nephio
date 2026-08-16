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

// OCloudService provides business logic for OCloud resource management
type OCloudService struct {
	*BaseService
	storage     storage.StorageInterface
	validator   validation.Validator
	integration integration.OperatorIntegration
}

// NewOCloudService creates a new OCloud service
func NewOCloudService(
	storage storage.StorageInterface,
	validator validation.Validator,
	integration integration.OperatorIntegration,
) *OCloudService {
	return &OCloudService{
		BaseService: NewBaseService(),
		storage:     storage,
		validator:   validator,
		integration: integration,
	}
}

// CreateDraft creates a new OCloud draft
func (s *OCloudService) CreateDraft(ctx context.Context, ocloud *models.OCloudData) (*models.OCloudData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	// Set initial state and timestamps (ID is already set by NewOCloudData)
	ocloud.State = models.StateDraft
	ocloud.UpdateTimestamp()

	// Create draft in storage
	if err := s.storage.CreateDraft(ctx, storage.ResourceTypeOCloud, ocloud); err != nil {
		return nil, fmt.Errorf("failed to create OCloud draft: %w", err)
	}

	return ocloud, nil
}

// GetDraft retrieves an OCloud draft
func (s *OCloudService) GetDraft(ctx context.Context, id string) (*models.OCloudData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	draft, err := s.storage.GetDraft(ctx, storage.ResourceTypeOCloud, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get OCloud draft: %w", err)
	}

	ocloud, ok := draft.(*models.OCloudData)
	if !ok {
		return nil, fmt.Errorf("invalid OCloud draft data type")
	}

	return ocloud, nil
}

// UpdateDraft updates an existing OCloud draft
func (s *OCloudService) UpdateDraft(ctx context.Context, id string, updates *models.OCloudData) (*models.OCloudData, error) {
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
	existing.O2IMSSecret = updates.O2IMSSecret
	existing.UpdateTimestamp()

	// Update draft in storage
	if err := s.storage.UpdateDraft(ctx, storage.ResourceTypeOCloud, id, existing); err != nil {
		return nil, fmt.Errorf("failed to update OCloud draft: %w", err)
	}

	return existing, nil
}

// ValidateDraft validates an OCloud draft
func (s *OCloudService) ValidateDraft(ctx context.Context, id string) (*models.ValidationResult, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	// Get draft
	draft, err := s.GetDraft(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate the draft
	result := s.validator.ValidateOCloud(ctx, draft)

	// If validation succeeds, update state to VALIDATED
	if result.Success {
		draft.SetState(models.StateValidated)
		if err := s.storage.UpdateDraft(ctx, storage.ResourceTypeOCloud, id, draft); err != nil {
			return nil, fmt.Errorf("failed to update draft state: %w", err)
		}
	}

	return result, nil
}

// RejectDraft rejects an OCloud draft and resets it to DRAFT state
func (s *OCloudService) RejectDraft(ctx context.Context, id string) error {
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

	if err := s.storage.UpdateDraft(ctx, storage.ResourceTypeOCloud, id, draft); err != nil {
		return fmt.Errorf("failed to reject OCloud draft: %w", err)
	}

	return nil
}

// ApproveDraft approves an OCloud draft and creates the corresponding Kubernetes CR
func (s *OCloudService) ApproveDraft(ctx context.Context, id string) (*models.OCloudData, error) {
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
	if err := s.storage.ApproveDraft(ctx, storage.ResourceTypeOCloud, id); err != nil {
		return nil, fmt.Errorf("failed to approve OCloud draft: %w", err)
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
	// if err := s.integration.CreateOCloudCR(ctx, approvedResource); err != nil {
	// 	return nil, fmt.Errorf("failed to create OCloud CR: %w", err)
	// }

	return approvedResource, nil
}

// DeleteDraft deletes an OCloud draft
func (s *OCloudService) DeleteDraft(ctx context.Context, id string) error {
	if err := s.ValidateContext(ctx); err != nil {
		return err
	}

	if err := s.storage.DeleteDraft(ctx, storage.ResourceTypeOCloud, id); err != nil {
		return fmt.Errorf("failed to delete OCloud draft: %w", err)
	}

	return nil
}

// Get retrieves an approved OCloud resource
func (s *OCloudService) Get(ctx context.Context, id string) (*models.OCloudData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	resource, err := s.storage.Get(ctx, storage.ResourceTypeOCloud, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get OCloud: %w", err)
	}

	ocloud, ok := resource.(*models.OCloudData)
	if !ok {
		return nil, fmt.Errorf("invalid OCloud data type")
	}

	return ocloud, nil
}

// List retrieves all approved OCloud resources
func (s *OCloudService) List(ctx context.Context) ([]*models.OCloudData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	resources, err := s.storage.List(ctx, storage.ResourceTypeOCloud)
	if err != nil {
		return nil, fmt.Errorf("failed to list OClouds: %w", err)
	}

	oclouds := make([]*models.OCloudData, len(resources))
	for i, resource := range resources {
		ocloud, ok := resource.(*models.OCloudData)
		if !ok {
			return nil, fmt.Errorf("invalid OCloud data type at index %d", i)
		}
		oclouds[i] = ocloud
	}

	return oclouds, nil
}

// GetRevisions retrieves all revisions for an OCloud resource
func (s *OCloudService) GetRevisions(ctx context.Context, id string) ([]*models.OCloudData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	revisions, err := s.storage.GetRevisions(ctx, storage.ResourceTypeOCloud, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get OCloud revisions: %w", err)
	}

	oclouds := make([]*models.OCloudData, len(revisions))
	for i, revision := range revisions {
		ocloud, ok := revision.(*models.OCloudData)
		if !ok {
			return nil, fmt.Errorf("invalid OCloud revision data type at index %d", i)
		}
		oclouds[i] = ocloud
	}

	return oclouds, nil
}

// GetRevision retrieves a specific revision of an OCloud resource
func (s *OCloudService) GetRevision(ctx context.Context, id, revisionId string) (*models.OCloudData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	revision, err := s.storage.GetRevision(ctx, storage.ResourceTypeOCloud, id, revisionId)
	if err != nil {
		return nil, fmt.Errorf("failed to get OCloud revision: %w", err)
	}

	ocloud, ok := revision.(*models.OCloudData)
	if !ok {
		return nil, fmt.Errorf("invalid OCloud revision data type")
	}

	return ocloud, nil
}

// CreateDraftFromRevision creates a new draft based on an existing revision
func (s *OCloudService) CreateDraftFromRevision(ctx context.Context, id, revisionId string) (*models.OCloudData, error) {
	if err := s.ValidateContext(ctx); err != nil {
		return nil, err
	}

	// Create draft from revision in storage
	if err := s.storage.CreateDraftFromRevision(ctx, storage.ResourceTypeOCloud, id, revisionId); err != nil {
		return nil, fmt.Errorf("failed to create OCloud draft from revision: %w", err)
	}

	// Get the newly created draft
	return s.GetDraft(ctx, id)
}

// Delete deletes an approved OCloud resource (with dependency validation)
func (s *OCloudService) Delete(ctx context.Context, id string) error {
	if err := s.ValidateContext(ctx); err != nil {
		return err
	}

	// Get the resource to validate dependencies
	ocloud, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Validate dependencies before deletion
	if err := s.storage.ValidateDependencies(ctx, storage.ResourceTypeOCloud, ocloud); err != nil {
		return fmt.Errorf("cannot delete OCloud due to dependencies: %w", err)
	}

	if err := s.storage.Delete(ctx, storage.ResourceTypeOCloud, id); err != nil {
		return fmt.Errorf("failed to delete OCloud: %w", err)
	}

	return nil
}
