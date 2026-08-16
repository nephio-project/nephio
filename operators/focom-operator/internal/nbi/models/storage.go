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

package models

import (
	"time"
)

// ResourceType represents the type of resource being stored
type ResourceType string

const (
	ResourceTypeOCloud                   ResourceType = "OCloud"
	ResourceTypeTemplateInfo             ResourceType = "TemplateInfo"
	ResourceTypeFocomProvisioningRequest ResourceType = "FocomProvisioningRequest"
)

// DraftStorage represents the storage model for draft revisions
type DraftStorage struct {
	ResourceID   string        `json:"resourceId" validate:"required"`
	ResourceType ResourceType  `json:"resourceType" validate:"required,oneof=OCloud TemplateInfo FocomProvisioningRequest"`
	DraftData    interface{}   `json:"draftData" validate:"required"`
	State        ResourceState `json:"state" validate:"required,oneof=DRAFT VALIDATED APPROVED"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
}

// NewDraftStorage creates a new DraftStorage instance
func NewDraftStorage(resourceID string, resourceType ResourceType, draftData interface{}, state ResourceState) *DraftStorage {
	now := time.Now()
	return &DraftStorage{
		ResourceID:   resourceID,
		ResourceType: resourceType,
		DraftData:    draftData,
		State:        state,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// UpdateTimestamp updates the UpdatedAt timestamp
func (ds *DraftStorage) UpdateTimestamp() {
	ds.UpdatedAt = time.Now()
}

// RevisionStorage represents the storage model for approved revisions
type RevisionStorage struct {
	ResourceID   string       `json:"resourceId" validate:"required"`
	RevisionID   string       `json:"revisionId" validate:"required,uuid"`
	ResourceType ResourceType `json:"resourceType" validate:"required,oneof=OCloud TemplateInfo FocomProvisioningRequest"`
	RevisionData interface{}  `json:"revisionData" validate:"required"`
	ApprovedAt   time.Time    `json:"approvedAt"`
}

// NewRevisionStorage creates a new RevisionStorage instance
func NewRevisionStorage(resourceID, revisionID string, resourceType ResourceType, revisionData interface{}) *RevisionStorage {
	return &RevisionStorage{
		ResourceID:   resourceID,
		RevisionID:   revisionID,
		ResourceType: resourceType,
		RevisionData: revisionData,
		ApprovedAt:   time.Now(),
	}
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Success        bool                    `json:"success"`
	ValidationTime time.Time               `json:"validationTime"`
	Errors         []string                `json:"errors,omitempty"`
	Warnings       []string                `json:"warnings,omitempty"`
	SchemaErrors   []SchemaValidationError `json:"schemaErrors,omitempty"`
}

// NewValidationResult creates a new ValidationResult
func NewValidationResult(success bool, errors, warnings []string) *ValidationResult {
	return &ValidationResult{
		Success:        success,
		ValidationTime: time.Now(),
		Errors:         errors,
		Warnings:       warnings,
	}
}
