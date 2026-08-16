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
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ResourceState represents the state of a resource revision
type ResourceState string

const (
	StateDraft     ResourceState = "DRAFT"
	StateValidated ResourceState = "VALIDATED"
	StateApproved  ResourceState = "APPROVED"
)

// BaseResource contains common fields for all resource types
type BaseResource struct {
	ID          string                 `json:"-" validate:"required"`                // Excluded from JSON, handled by custom marshaling
	RevisionID  string                 `json:"revisionId" validate:"omitempty,uuid"` // Optional at draft stage, assigned at approval
	Namespace   string                 `json:"namespace" validate:"required"`
	Name        string                 `json:"name" validate:"required"`
	Description string                 `json:"description" validate:"required"`
	State       ResourceState          `json:"-" validate:"required,oneof=DRAFT VALIDATED APPROVED"` // Excluded from JSON, handled by custom marshaling
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SanitizeID converts a string to a valid Kubernetes resource name
// Rules: lowercase, alphanumeric, hyphens only, max 63 chars, no leading/trailing hyphens
func SanitizeID(id string) string {
	// Convert to lowercase
	id = strings.ToLower(id)

	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	id = reg.ReplaceAllString(id, "-")

	// Remove consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	id = reg.ReplaceAllString(id, "-")

	// Remove leading/trailing hyphens
	id = strings.Trim(id, "-")

	// Limit to 63 characters (Kubernetes resource name limit)
	if len(id) > 63 {
		id = id[:63]
		// Ensure we don't end with a hyphen after truncation
		id = strings.TrimRight(id, "-")
	}

	return id
}

// NewBaseResource creates a new BaseResource with generated IDs and timestamps
func NewBaseResource(namespace, name, description string) BaseResource {
	now := time.Now()
	return BaseResource{
		ID:          uuid.New().String(),
		RevisionID:  uuid.New().String(),
		Namespace:   namespace,
		Name:        name,
		Description: description,
		State:       StateDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata:    make(map[string]interface{}),
	}
}

// UpdateRevision creates a new revision ID and updates the timestamp
func (br *BaseResource) UpdateRevision() {
	br.RevisionID = uuid.New().String()
	br.UpdatedAt = time.Now()
}

// UpdateTimestamp updates only the UpdatedAt timestamp
func (br *BaseResource) UpdateTimestamp() {
	br.UpdatedAt = time.Now()
}

// IsValidStateTransition checks if a state transition is valid
func (br *BaseResource) IsValidStateTransition(newState ResourceState) bool {
	switch br.State {
	case StateDraft:
		return newState == StateValidated || newState == StateDraft
	case StateValidated:
		return newState == StateApproved || newState == StateDraft
	case StateApproved:
		return false // Approved resources cannot change state
	default:
		return false
	}
}

// SetState sets the resource state if the transition is valid
func (br *BaseResource) SetState(newState ResourceState) bool {
	if br.IsValidStateTransition(newState) {
		br.State = newState
		br.UpdateTimestamp()
		return true
	}
	return false
}
