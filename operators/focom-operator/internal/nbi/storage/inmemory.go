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

package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
)

// InMemoryStorage implements the StorageInterface using in-memory data structures
type InMemoryStorage struct {
	// Mutex for concurrent access protection
	mu sync.RWMutex

	// Storage for approved resources by resource type and ID
	approvedResources map[ResourceType]map[string]interface{}

	// Storage for draft resources by resource type and ID
	draftResources map[ResourceType]map[string]*models.DraftStorage

	// Storage for revision history by resource type, resource ID, and revision ID
	revisionHistory map[ResourceType]map[string]map[string]*models.RevisionStorage

	// Track resource creation order for dependency validation
	resourceOrder map[string]time.Time
}

// NewInMemoryStorage creates a new in-memory storage instance
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		approvedResources: make(map[ResourceType]map[string]interface{}),
		draftResources:    make(map[ResourceType]map[string]*models.DraftStorage),
		revisionHistory:   make(map[ResourceType]map[string]map[string]*models.RevisionStorage),
		resourceOrder:     make(map[string]time.Time),
	}
}

// Clear removes all data from storage
func (s *InMemoryStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.approvedResources = make(map[ResourceType]map[string]interface{})
	s.draftResources = make(map[ResourceType]map[string]*models.DraftStorage)
	s.revisionHistory = make(map[ResourceType]map[string]map[string]*models.RevisionStorage)
	s.resourceOrder = make(map[string]time.Time)
}

// initializeResourceType ensures the maps for a resource type are initialized
func (s *InMemoryStorage) initializeResourceType(resourceType ResourceType) {
	if s.approvedResources[resourceType] == nil {
		s.approvedResources[resourceType] = make(map[string]interface{})
	}
	if s.draftResources[resourceType] == nil {
		s.draftResources[resourceType] = make(map[string]*models.DraftStorage)
	}
	if s.revisionHistory[resourceType] == nil {
		s.revisionHistory[resourceType] = make(map[string]map[string]*models.RevisionStorage)
	}
}

// Create stores a new approved resource
func (s *InMemoryStorage) Create(ctx context.Context, resourceType ResourceType, resource interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeResourceType(resourceType)

	// Extract ID from resource
	id, err := s.extractResourceID(resource)
	if err != nil {
		return NewStorageError(ErrorCodeInvalidID, "failed to extract resource ID", err)
	}

	// Check if resource already exists
	if _, exists := s.approvedResources[resourceType][id]; exists {
		return NewStorageError(ErrorCodeAlreadyExists, fmt.Sprintf("resource %s already exists", id), ErrResourceExists)
	}

	// Store the resource
	s.approvedResources[resourceType][id] = resource
	s.resourceOrder[id] = time.Now()

	return nil
}

// Get retrieves an approved resource by ID
func (s *InMemoryStorage) Get(ctx context.Context, resourceType ResourceType, id string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.initializeResourceType(resourceType)

	resource, exists := s.approvedResources[resourceType][id]
	if !exists {
		return nil, NewStorageError(ErrorCodeNotFound, fmt.Sprintf("resource %s not found", id), ErrResourceNotFound)
	}

	return resource, nil
}

// Update modifies an existing approved resource
func (s *InMemoryStorage) Update(ctx context.Context, resourceType ResourceType, id string, resource interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeResourceType(resourceType)

	// Check if resource exists
	if _, exists := s.approvedResources[resourceType][id]; !exists {
		return NewStorageError(ErrorCodeNotFound, fmt.Sprintf("resource %s not found", id), ErrResourceNotFound)
	}

	// Update the resource
	s.approvedResources[resourceType][id] = resource

	return nil
}

// Delete removes an approved resource
func (s *InMemoryStorage) Delete(ctx context.Context, resourceType ResourceType, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeResourceType(resourceType)

	// Check if resource exists
	if _, exists := s.approvedResources[resourceType][id]; !exists {
		return NewStorageError(ErrorCodeNotFound, fmt.Sprintf("resource %s not found", id), ErrResourceNotFound)
	}

	// Delete the resource
	delete(s.approvedResources[resourceType], id)
	delete(s.resourceOrder, id)

	// Also clean up any draft and revision history
	delete(s.draftResources[resourceType], id)
	delete(s.revisionHistory[resourceType], id)

	return nil
}

// List returns all approved resources of a given type
func (s *InMemoryStorage) List(ctx context.Context, resourceType ResourceType) ([]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.initializeResourceType(resourceType)

	resources := make([]interface{}, 0, len(s.approvedResources[resourceType]))
	for _, resource := range s.approvedResources[resourceType] {
		resources = append(resources, resource)
	}

	return resources, nil
}

// extractResourceID extracts the ID from a resource interface
func (s *InMemoryStorage) extractResourceID(resource interface{}) (string, error) {
	switch r := resource.(type) {
	case *models.OCloudData:
		return r.ID, nil
	case *models.TemplateInfoData:
		return r.ID, nil
	case *models.FocomProvisioningRequestData:
		return r.ID, nil
	case models.OCloudData:
		return r.ID, nil
	case models.TemplateInfoData:
		return r.ID, nil
	case models.FocomProvisioningRequestData:
		return r.ID, nil
	case *OCloudData:
		return r.ID, nil
	case *TemplateInfoData:
		return r.ID, nil
	case *FocomProvisioningRequestData:
		return r.ID, nil
	case OCloudData:
		return r.ID, nil
	case TemplateInfoData:
		return r.ID, nil
	case FocomProvisioningRequestData:
		return r.ID, nil
	default:
		// Try to extract ID using reflection or type assertion
		if hasID, ok := resource.(interface{ GetID() string }); ok {
			return hasID.GetID(), nil
		}
		return "", fmt.Errorf("unsupported resource type: %T", resource)
	}
}

// HealthCheck verifies the storage is operational
func (s *InMemoryStorage) HealthCheck(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// For in-memory storage, we just check if the maps are initialized
	if s.approvedResources == nil || s.draftResources == nil || s.revisionHistory == nil {
		return NewStorageError(ErrorCodeStorageFailure, "storage not properly initialized", ErrStorageUnavailable)
	}

	return nil
}

// CreateDraft creates a new draft resource
func (s *InMemoryStorage) CreateDraft(ctx context.Context, resourceType ResourceType, draft interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeResourceType(resourceType)

	// Extract ID from draft
	id, err := s.extractResourceID(draft)
	if err != nil {
		return NewStorageError(ErrorCodeInvalidID, "failed to extract resource ID from draft", err)
	}

	// Check if draft already exists
	if _, exists := s.draftResources[resourceType][id]; exists {
		return NewStorageError(ErrorCodeAlreadyExists, fmt.Sprintf("draft for resource %s already exists", id), ErrResourceExists)
	}

	// Convert resource type to models.ResourceType
	modelResourceType := s.convertResourceType(resourceType)

	// Create draft storage
	draftStorage := models.NewDraftStorage(id, modelResourceType, draft, models.StateDraft)
	s.draftResources[resourceType][id] = draftStorage

	return nil
}

// GetDraft retrieves a draft resource by ID
func (s *InMemoryStorage) GetDraft(ctx context.Context, resourceType ResourceType, id string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.initializeResourceType(resourceType)

	draft, exists := s.draftResources[resourceType][id]
	if !exists {
		return nil, NewStorageError(ErrorCodeNotFound, fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	return draft.DraftData, nil
}

// UpdateDraft modifies an existing draft resource
func (s *InMemoryStorage) UpdateDraft(ctx context.Context, resourceType ResourceType, id string, draft interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeResourceType(resourceType)

	// Check if draft exists
	draftStorage, exists := s.draftResources[resourceType][id]
	if !exists {
		return NewStorageError(ErrorCodeNotFound, fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	// Check if draft is in a state that allows updates
	if draftStorage.State == models.StateValidated {
		return NewStorageError(ErrorCodeInvalidState, "cannot update draft in VALIDATED state", fmt.Errorf("draft is validated"))
	}

	// Update the draft
	draftStorage.DraftData = draft
	draftStorage.UpdateTimestamp()

	return nil
}

// DeleteDraft removes a draft resource
func (s *InMemoryStorage) DeleteDraft(ctx context.Context, resourceType ResourceType, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeResourceType(resourceType)

	// Check if draft exists
	if _, exists := s.draftResources[resourceType][id]; !exists {
		return NewStorageError(ErrorCodeNotFound, fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	// Delete the draft
	delete(s.draftResources[resourceType], id)

	// If there's no approved resource and no revisions, clean up completely
	if _, hasApproved := s.approvedResources[resourceType][id]; !hasApproved {
		if revisions, hasRevisions := s.revisionHistory[resourceType][id]; !hasRevisions || len(revisions) == 0 {
			delete(s.resourceOrder, id)
		}
	}

	return nil
}

// ValidateDraft validates a draft and changes its state to VALIDATED
func (s *InMemoryStorage) ValidateDraft(ctx context.Context, resourceType ResourceType, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeResourceType(resourceType)

	// Check if draft exists
	draftStorage, exists := s.draftResources[resourceType][id]
	if !exists {
		return NewStorageError(ErrorCodeNotFound, fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	// Check if draft is in DRAFT state
	if draftStorage.State != models.StateDraft {
		return NewStorageError(ErrorCodeInvalidState, "can only validate drafts in DRAFT state", fmt.Errorf("current state: %s", draftStorage.State))
	}

	// Update both the draft storage state and the resource data state to VALIDATED
	draftStorage.State = models.StateValidated
	draftStorage.UpdateTimestamp()

	// Also update the resource data's state
	if err := s.updateResourceState(draftStorage.DraftData, models.StateValidated); err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update resource state to VALIDATED", err)
	}

	return nil
}

// ApproveDraft approves a validated draft and promotes it to an approved revision
func (s *InMemoryStorage) ApproveDraft(ctx context.Context, resourceType ResourceType, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeResourceType(resourceType)

	// Check if draft exists
	draftStorage, exists := s.draftResources[resourceType][id]
	if !exists {
		return NewStorageError(ErrorCodeNotFound, fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	// Check if draft is in VALIDATED state
	if draftStorage.State != models.StateValidated {
		return NewStorageError(ErrorCodeInvalidState, "can only approve drafts in VALIDATED state", fmt.Errorf("current state: %s", draftStorage.State))
	}

	// Update the draft data state to APPROVED before storing
	if err := s.updateResourceState(draftStorage.DraftData, models.StateApproved); err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update resource state to APPROVED", err)
	}

	// Extract revision ID from the draft data
	revisionID, err := s.extractRevisionID(draftStorage.DraftData)
	if err != nil {
		return NewStorageError(ErrorCodeInvalidRevision, "failed to extract revision ID from draft", err)
	}

	// Convert resource type to models.ResourceType
	modelResourceType := s.convertResourceType(resourceType)

	// Create revision storage
	if s.revisionHistory[resourceType][id] == nil {
		s.revisionHistory[resourceType][id] = make(map[string]*models.RevisionStorage)
	}

	revisionStorage := models.NewRevisionStorage(id, revisionID, modelResourceType, draftStorage.DraftData)
	s.revisionHistory[resourceType][id][revisionID] = revisionStorage

	// Update approved resource
	s.approvedResources[resourceType][id] = draftStorage.DraftData
	s.resourceOrder[id] = time.Now()

	// Remove the draft
	delete(s.draftResources[resourceType], id)

	return nil
}

// RejectDraft rejects a draft and changes its state back to DRAFT
func (s *InMemoryStorage) RejectDraft(ctx context.Context, resourceType ResourceType, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeResourceType(resourceType)

	// Check if draft exists
	draftStorage, exists := s.draftResources[resourceType][id]
	if !exists {
		return NewStorageError(ErrorCodeNotFound, fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	// Check if draft is in VALIDATED state
	if draftStorage.State != models.StateValidated {
		return NewStorageError(ErrorCodeInvalidState, "can only reject drafts in VALIDATED state", fmt.Errorf("current state: %s", draftStorage.State))
	}

	// Update both the draft storage state and the resource data state back to DRAFT
	draftStorage.State = models.StateDraft
	draftStorage.UpdateTimestamp()

	// Also update the resource data's state
	if err := s.updateResourceState(draftStorage.DraftData, models.StateDraft); err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update resource state to DRAFT", err)
	}

	return nil
}

// convertResourceType converts storage.ResourceType to models.ResourceType
func (s *InMemoryStorage) convertResourceType(resourceType ResourceType) models.ResourceType {
	switch resourceType {
	case ResourceTypeOCloud:
		return models.ResourceTypeOCloud
	case ResourceTypeTemplateInfo:
		return models.ResourceTypeTemplateInfo
	case ResourceTypeFocomProvisioningRequest:
		return models.ResourceTypeFocomProvisioningRequest
	default:
		return models.ResourceType(string(resourceType))
	}
}

// extractRevisionID extracts the revision ID from a resource interface
func (s *InMemoryStorage) extractRevisionID(resource interface{}) (string, error) {
	switch r := resource.(type) {
	case *models.OCloudData:
		return r.RevisionID, nil
	case *models.TemplateInfoData:
		return r.RevisionID, nil
	case *models.FocomProvisioningRequestData:
		return r.RevisionID, nil
	case models.OCloudData:
		return r.RevisionID, nil
	case models.TemplateInfoData:
		return r.RevisionID, nil
	case models.FocomProvisioningRequestData:
		return r.RevisionID, nil
	case *OCloudData:
		return r.RevisionID, nil
	case *TemplateInfoData:
		return r.RevisionID, nil
	case *FocomProvisioningRequestData:
		return r.RevisionID, nil
	case OCloudData:
		return r.RevisionID, nil
	case TemplateInfoData:
		return r.RevisionID, nil
	case FocomProvisioningRequestData:
		return r.RevisionID, nil
	default:
		// Try to extract revision ID using reflection or type assertion
		if hasRevisionID, ok := resource.(interface{ GetRevisionID() string }); ok {
			return hasRevisionID.GetRevisionID(), nil
		}
		return "", fmt.Errorf("unsupported resource type for revision ID extraction: %T", resource)
	}
}

// GetRevisions returns all approved revisions for a resource
func (s *InMemoryStorage) GetRevisions(ctx context.Context, resourceType ResourceType, id string) ([]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.initializeResourceType(resourceType)

	// Check if resource has any revisions
	revisions, exists := s.revisionHistory[resourceType][id]
	if !exists || len(revisions) == 0 {
		return []interface{}{}, nil
	}

	// Convert revision storage to interface slice
	// Return the actual revision data, not the RevisionStorage wrapper
	result := make([]interface{}, 0, len(revisions))
	for _, revision := range revisions {
		result = append(result, revision.RevisionData)
	}

	return result, nil
}

// GetRevision retrieves a specific approved revision
func (s *InMemoryStorage) GetRevision(ctx context.Context, resourceType ResourceType, id string, revisionId string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.initializeResourceType(resourceType)

	// Check if resource has revisions
	revisions, exists := s.revisionHistory[resourceType][id]
	if !exists {
		return nil, NewStorageError(ErrorCodeNotFound, fmt.Sprintf("no revisions found for resource %s", id), ErrResourceNotFound)
	}

	// Check if specific revision exists
	revision, exists := revisions[revisionId]
	if !exists {
		return nil, NewStorageError(ErrorCodeInvalidRevision, fmt.Sprintf("revision %s not found for resource %s", revisionId, id), ErrInvalidRevisionID)
	}

	return revision, nil
}

// CreateDraftFromRevision creates a new draft based on an existing approved revision
func (s *InMemoryStorage) CreateDraftFromRevision(ctx context.Context, resourceType ResourceType, id string, revisionId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeResourceType(resourceType)

	// Check if a draft already exists for this resource
	if _, exists := s.draftResources[resourceType][id]; exists {
		return NewStorageError(ErrorCodeAlreadyExists, fmt.Sprintf("draft already exists for resource %s", id), ErrResourceExists)
	}

	// Check if resource has revisions
	revisions, exists := s.revisionHistory[resourceType][id]
	if !exists {
		return NewStorageError(ErrorCodeNotFound, fmt.Sprintf("no revisions found for resource %s", id), ErrResourceNotFound)
	}

	// Check if specific revision exists
	revision, exists := revisions[revisionId]
	if !exists {
		return NewStorageError(ErrorCodeInvalidRevision, fmt.Sprintf("revision %s not found for resource %s", revisionId, id), ErrInvalidRevisionID)
	}

	// Create a copy of the revision data for the new draft
	draftData := s.copyResourceData(revision.RevisionData)

	// Update the draft data with new revision ID and reset state
	if err := s.updateDraftDataForNewRevision(draftData); err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to prepare draft data from revision", err)
	}

	// Convert resource type to models.ResourceType
	modelResourceType := s.convertResourceType(resourceType)

	// Create draft storage
	draftStorage := models.NewDraftStorage(id, modelResourceType, draftData, models.StateDraft)
	s.draftResources[resourceType][id] = draftStorage

	return nil
}

// copyResourceData creates a deep copy of resource data
func (s *InMemoryStorage) copyResourceData(original interface{}) interface{} {
	switch r := original.(type) {
	case *models.OCloudData:
		return r.Clone()
	case *models.TemplateInfoData:
		return r.Clone()
	case *models.FocomProvisioningRequestData:
		return r.Clone()
	case models.OCloudData:
		return r.Clone()
	case models.TemplateInfoData:
		return r.Clone()
	case models.FocomProvisioningRequestData:
		return r.Clone()
	default:
		// For unknown types, return as-is (this might need enhancement for complex types)
		return original
	}
}

// updateDraftDataForNewRevision updates the draft data with new revision ID and draft state
func (s *InMemoryStorage) updateDraftDataForNewRevision(draftData interface{}) error {
	switch r := draftData.(type) {
	case *models.OCloudData:
		r.UpdateRevision()
		r.State = models.StateDraft
		return nil
	case *models.TemplateInfoData:
		r.UpdateRevision()
		r.State = models.StateDraft
		return nil
	case *models.FocomProvisioningRequestData:
		r.UpdateRevision()
		r.State = models.StateDraft
		return nil
	default:
		// Try to use interface methods if available
		if hasUpdateRevision, ok := draftData.(interface{ UpdateRevision() }); ok {
			hasUpdateRevision.UpdateRevision()
		}
		if hasSetState, ok := draftData.(interface {
			SetState(models.ResourceState) bool
		}); ok {
			hasSetState.SetState(models.StateDraft)
		}
		return nil
	}
}

// ValidateDependencies validates dependencies based on the operation:
// - For FocomProvisioningRequest: validates that referenced resources exist
// - For OCloud/TemplateInfo: validates that no FPRs reference this resource (for deletion prevention)
func (s *InMemoryStorage) ValidateDependencies(ctx context.Context, resourceType ResourceType, resource interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch resourceType {
	case ResourceTypeFocomProvisioningRequest:
		return s.validateFPRDependencies(resource)
	case ResourceTypeOCloud:
		return s.validateOCloudReferences(resource)
	case ResourceTypeTemplateInfo:
		return s.validateTemplateInfoReferences(resource)
	default:
		return nil
	}
}

// validateFPRDependencies validates that an FPR's dependencies exist
func (s *InMemoryStorage) validateFPRDependencies(resource interface{}) error {
	// Extract dependency information from the resource
	fpr, ok := resource.(*models.FocomProvisioningRequestData)
	if !ok {
		if fprValue, ok := resource.(models.FocomProvisioningRequestData); ok {
			fpr = &fprValue
		} else {
			return NewStorageError(ErrorCodeDependencyFailed, "invalid resource type for dependency validation", fmt.Errorf("expected FocomProvisioningRequestData, got %T", resource))
		}
	}

	// Validate OCloud dependency
	if fpr.OCloudID != "" {
		if _, exists := s.approvedResources[ResourceTypeOCloud][fpr.OCloudID]; !exists {
			return NewStorageError(ErrorCodeDependencyFailed, fmt.Sprintf("referenced OCloud %s does not exist", fpr.OCloudID), fmt.Errorf("missing OCloud dependency"))
		}
	}

	// Validate TemplateInfo dependency
	if fpr.TemplateName != "" && fpr.TemplateVersion != "" {
		// Find TemplateInfo by name and version
		templateFound := false
		for _, templateResource := range s.approvedResources[ResourceTypeTemplateInfo] {
			if template, ok := templateResource.(*models.TemplateInfoData); ok {
				if template.TemplateName == fpr.TemplateName && template.TemplateVersion == fpr.TemplateVersion {
					templateFound = true
					break
				}
			} else if template, ok := templateResource.(models.TemplateInfoData); ok {
				if template.TemplateName == fpr.TemplateName && template.TemplateVersion == fpr.TemplateVersion {
					templateFound = true
					break
				}
			}
		}

		if !templateFound {
			return NewStorageError(ErrorCodeDependencyFailed, fmt.Sprintf("referenced TemplateInfo %s:%s does not exist", fpr.TemplateName, fpr.TemplateVersion), fmt.Errorf("missing TemplateInfo dependency"))
		}
	}

	return nil
}

// validateOCloudReferences validates that no FPRs reference this OCloud (for deletion prevention)
func (s *InMemoryStorage) validateOCloudReferences(resource interface{}) error {
	// Extract OCloud ID
	var ocloudID string
	switch r := resource.(type) {
	case *models.OCloudData:
		ocloudID = r.ID
	case models.OCloudData:
		ocloudID = r.ID
	default:
		return NewStorageError(ErrorCodeDependencyFailed, "invalid resource type for OCloud reference validation", fmt.Errorf("expected OCloudData, got %T", resource))
	}

	// Check if any FPRs reference this OCloud
	var referencingFPRs []string
	for _, fprResource := range s.approvedResources[ResourceTypeFocomProvisioningRequest] {
		if fpr, ok := fprResource.(*models.FocomProvisioningRequestData); ok {
			if fpr.OCloudID == ocloudID {
				referencingFPRs = append(referencingFPRs, fpr.ID)
			}
		} else if fpr, ok := fprResource.(models.FocomProvisioningRequestData); ok {
			if fpr.OCloudID == ocloudID {
				referencingFPRs = append(referencingFPRs, fpr.ID)
			}
		}
	}

	if len(referencingFPRs) > 0 {
		return NewStorageError(ErrorCodeDependencyFailed, fmt.Sprintf("OCloud %s cannot be deleted because it is referenced by FocomProvisioningRequest(s): %v", ocloudID, referencingFPRs), fmt.Errorf("resource is referenced"))
	}

	return nil
}

// validateTemplateInfoReferences validates that no FPRs reference this TemplateInfo (for deletion prevention)
func (s *InMemoryStorage) validateTemplateInfoReferences(resource interface{}) error {
	// Extract TemplateInfo details
	var templateName, templateVersion string
	switch r := resource.(type) {
	case *models.TemplateInfoData:
		templateName = r.TemplateName
		templateVersion = r.TemplateVersion
	case models.TemplateInfoData:
		templateName = r.TemplateName
		templateVersion = r.TemplateVersion
	default:
		return NewStorageError(ErrorCodeDependencyFailed, "invalid resource type for TemplateInfo reference validation", fmt.Errorf("expected TemplateInfoData, got %T", resource))
	}

	// Check if any FPRs reference this TemplateInfo
	var referencingFPRs []string
	for _, fprResource := range s.approvedResources[ResourceTypeFocomProvisioningRequest] {
		if fpr, ok := fprResource.(*models.FocomProvisioningRequestData); ok {
			if fpr.TemplateName == templateName && fpr.TemplateVersion == templateVersion {
				referencingFPRs = append(referencingFPRs, fpr.ID)
			}
		} else if fpr, ok := fprResource.(models.FocomProvisioningRequestData); ok {
			if fpr.TemplateName == templateName && fpr.TemplateVersion == templateVersion {
				referencingFPRs = append(referencingFPRs, fpr.ID)
			}
		}
	}

	if len(referencingFPRs) > 0 {
		return NewStorageError(ErrorCodeDependencyFailed, fmt.Sprintf("TemplateInfo %s:%s cannot be deleted because it is referenced by FocomProvisioningRequest(s): %v", templateName, templateVersion, referencingFPRs), fmt.Errorf("resource is referenced"))
	}

	return nil
}

// CreateRevision creates a new revision directly (used for loading test data)
func (s *InMemoryStorage) CreateRevision(ctx context.Context, resourceType ResourceType, resourceID string, revisionID string, data interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeResourceType(resourceType)

	// Initialize revision history for this resource if needed
	if s.revisionHistory[resourceType][resourceID] == nil {
		s.revisionHistory[resourceType][resourceID] = make(map[string]*models.RevisionStorage)
	}

	// Convert resource type to models.ResourceType
	modelResourceType := s.convertResourceType(resourceType)

	// Create revision storage
	revisionStorage := models.NewRevisionStorage(resourceID, revisionID, modelResourceType, data)
	s.revisionHistory[resourceType][resourceID][revisionID] = revisionStorage

	// Update approved resource
	s.approvedResources[resourceType][resourceID] = data
	s.resourceOrder[resourceID] = time.Now()

	return nil
}

// updateResourceState updates the state of a resource data object
func (s *InMemoryStorage) updateResourceState(resource interface{}, state models.ResourceState) error {
	switch r := resource.(type) {
	case *models.OCloudData:
		r.State = state
		r.UpdateTimestamp()
		return nil
	case *models.TemplateInfoData:
		r.State = state
		r.UpdateTimestamp()
		return nil
	case *models.FocomProvisioningRequestData:
		r.State = state
		r.UpdateTimestamp()
		return nil
	default:
		// Try to use interface methods if available
		if hasSetState, ok := resource.(interface {
			SetState(models.ResourceState) bool
		}); ok {
			if hasSetState.SetState(state) {
				return nil
			}
		}
		return fmt.Errorf("unsupported resource type for state update: %T", resource)
	}
}

// UpdateDraftState updates the state of a draft resource
func (s *InMemoryStorage) UpdateDraftState(ctx context.Context, resourceType ResourceType, id string, state ResourceState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeResourceType(resourceType)

	// Check if draft exists
	draftStorage, exists := s.draftResources[resourceType][id]
	if !exists {
		return NewStorageError(ErrorCodeNotFound, fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	// Convert storage.ResourceState to models.ResourceState
	var modelState models.ResourceState
	switch state {
	case StateDraft:
		modelState = models.StateDraft
	case StateValidated:
		modelState = models.StateValidated
	case StateApproved:
		modelState = models.StateApproved
	default:
		return NewStorageError(ErrorCodeInvalidState, fmt.Sprintf("invalid state: %s", state), fmt.Errorf("unsupported state"))
	}

	// Update the draft state
	draftStorage.State = modelState
	draftStorage.UpdateTimestamp()

	return nil
}
