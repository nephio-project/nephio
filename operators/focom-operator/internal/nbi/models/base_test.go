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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBaseResource(t *testing.T) {
	namespace := "test-namespace"
	name := "test-resource"
	description := "Test resource description"

	resource := NewBaseResource(namespace, name, description)

	assert.NotEmpty(t, resource.ID)
	assert.NotEmpty(t, resource.RevisionID)
	assert.Equal(t, namespace, resource.Namespace)
	assert.Equal(t, name, resource.Name)
	assert.Equal(t, description, resource.Description)
	assert.Equal(t, StateDraft, resource.State)
	assert.NotZero(t, resource.CreatedAt)
	assert.NotZero(t, resource.UpdatedAt)
	assert.NotNil(t, resource.Metadata)
	assert.Equal(t, resource.CreatedAt, resource.UpdatedAt)
}

func TestBaseResource_UpdateRevision(t *testing.T) {
	resource := NewBaseResource("test", "test", "test")
	originalRevisionID := resource.RevisionID
	originalUpdatedAt := resource.UpdatedAt

	// Wait a bit to ensure timestamp difference
	time.Sleep(1 * time.Millisecond)

	resource.UpdateRevision()

	assert.NotEqual(t, originalRevisionID, resource.RevisionID)
	assert.True(t, resource.UpdatedAt.After(originalUpdatedAt))
}

func TestBaseResource_UpdateTimestamp(t *testing.T) {
	resource := NewBaseResource("test", "test", "test")
	originalRevisionID := resource.RevisionID
	originalUpdatedAt := resource.UpdatedAt

	// Wait a bit to ensure timestamp difference
	time.Sleep(1 * time.Millisecond)

	resource.UpdateTimestamp()

	assert.Equal(t, originalRevisionID, resource.RevisionID) // Should not change
	assert.True(t, resource.UpdatedAt.After(originalUpdatedAt))
}

func TestBaseResource_IsValidStateTransition(t *testing.T) {
	tests := []struct {
		name         string
		currentState ResourceState
		newState     ResourceState
		expected     bool
	}{
		// From DRAFT
		{"DRAFT to DRAFT", StateDraft, StateDraft, true},
		{"DRAFT to VALIDATED", StateDraft, StateValidated, true},
		{"DRAFT to APPROVED", StateDraft, StateApproved, false},

		// From VALIDATED
		{"VALIDATED to DRAFT", StateValidated, StateDraft, true},
		{"VALIDATED to VALIDATED", StateValidated, StateValidated, false},
		{"VALIDATED to APPROVED", StateValidated, StateApproved, true},

		// From APPROVED
		{"APPROVED to DRAFT", StateApproved, StateDraft, false},
		{"APPROVED to VALIDATED", StateApproved, StateValidated, false},
		{"APPROVED to APPROVED", StateApproved, StateApproved, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := NewBaseResource("test", "test", "test")
			resource.State = tt.currentState

			result := resource.IsValidStateTransition(tt.newState)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBaseResource_SetState(t *testing.T) {
	t.Run("Valid state transition", func(t *testing.T) {
		resource := NewBaseResource("test", "test", "test")
		originalUpdatedAt := resource.UpdatedAt

		// Wait a bit to ensure timestamp difference
		time.Sleep(1 * time.Millisecond)

		success := resource.SetState(StateValidated)

		assert.True(t, success)
		assert.Equal(t, StateValidated, resource.State)
		assert.True(t, resource.UpdatedAt.After(originalUpdatedAt))
	})

	t.Run("Invalid state transition", func(t *testing.T) {
		resource := NewBaseResource("test", "test", "test")
		resource.State = StateApproved
		originalUpdatedAt := resource.UpdatedAt

		success := resource.SetState(StateDraft)

		assert.False(t, success)
		assert.Equal(t, StateApproved, resource.State)         // Should not change
		assert.Equal(t, originalUpdatedAt, resource.UpdatedAt) // Should not change
	})

	t.Run("Complete workflow", func(t *testing.T) {
		resource := NewBaseResource("test", "test", "test")

		// DRAFT -> VALIDATED
		assert.True(t, resource.SetState(StateValidated))
		assert.Equal(t, StateValidated, resource.State)

		// VALIDATED -> APPROVED
		assert.True(t, resource.SetState(StateApproved))
		assert.Equal(t, StateApproved, resource.State)

		// APPROVED -> DRAFT (should fail)
		assert.False(t, resource.SetState(StateDraft))
		assert.Equal(t, StateApproved, resource.State)
	})

	t.Run("Rejection workflow", func(t *testing.T) {
		resource := NewBaseResource("test", "test", "test")

		// DRAFT -> VALIDATED
		assert.True(t, resource.SetState(StateValidated))
		assert.Equal(t, StateValidated, resource.State)

		// VALIDATED -> DRAFT (rejection)
		assert.True(t, resource.SetState(StateDraft))
		assert.Equal(t, StateDraft, resource.State)

		// DRAFT -> VALIDATED (re-validation)
		assert.True(t, resource.SetState(StateValidated))
		assert.Equal(t, StateValidated, resource.State)
	})
}

func TestResourceState_Constants(t *testing.T) {
	assert.Equal(t, ResourceState("DRAFT"), StateDraft)
	assert.Equal(t, ResourceState("VALIDATED"), StateValidated)
	assert.Equal(t, ResourceState("APPROVED"), StateApproved)
}

func TestBaseResource_Metadata(t *testing.T) {
	resource := NewBaseResource("test", "test", "test")

	// Test initial metadata
	assert.NotNil(t, resource.Metadata)
	assert.Empty(t, resource.Metadata)

	// Test adding metadata
	resource.Metadata["key1"] = "value1"
	resource.Metadata["key2"] = 42
	resource.Metadata["key3"] = map[string]string{"nested": "value"}

	assert.Equal(t, "value1", resource.Metadata["key1"])
	assert.Equal(t, 42, resource.Metadata["key2"])
	assert.Equal(t, map[string]string{"nested": "value"}, resource.Metadata["key3"])
}

func TestBaseResource_ImmutableFields(t *testing.T) {
	resource := NewBaseResource("test-namespace", "test-name", "test-description")

	originalID := resource.ID
	originalCreatedAt := resource.CreatedAt

	// Simulate some operations that should not change immutable fields
	resource.UpdateTimestamp()
	resource.UpdateRevision()
	resource.SetState(StateValidated)

	// ID and CreatedAt should remain unchanged
	assert.Equal(t, originalID, resource.ID)
	assert.Equal(t, originalCreatedAt, resource.CreatedAt)

	// But other fields should be updatable
	resource.Name = "updated-name"
	resource.Description = "updated-description"

	assert.Equal(t, "updated-name", resource.Name)
	assert.Equal(t, "updated-description", resource.Description)
}
