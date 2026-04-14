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
	"testing"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStorage_CreateDraft(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	t.Run("Create OCloud draft", func(t *testing.T) {
		ocloud := &models.OCloudData{
			BaseResource: models.NewBaseResource("default", "test-ocloud", "Test OCloud"),
			O2IMSSecret: models.O2IMSSecretRef{
				SecretRef: models.SecretReference{
					Name:      "test-secret",
					Namespace: "default",
				},
			},
		}

		err := storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
		assert.NoError(t, err)

		// Verify draft was created
		draft, err := storage.GetDraft(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.NoError(t, err)
		assert.NotNil(t, draft)

		ocloudDraft := draft.(*models.OCloudData)
		assert.Equal(t, ocloud.ID, ocloudDraft.ID)
		assert.Equal(t, ocloud.Name, ocloudDraft.Name)
		assert.Equal(t, models.StateDraft, ocloudDraft.State)
	})

	t.Run("Create TemplateInfo draft", func(t *testing.T) {
		template := &models.TemplateInfoData{
			BaseResource:            models.NewBaseResource("default", "test-template", "Test Template"),
			TemplateName:            "cluster-template",
			TemplateVersion:         "v1.0.0",
			TemplateParameterSchema: `{"type": "object"}`,
		}

		err := storage.CreateDraft(ctx, ResourceTypeTemplateInfo, template)
		assert.NoError(t, err)

		// Verify draft was created
		draft, err := storage.GetDraft(ctx, ResourceTypeTemplateInfo, template.ID)
		assert.NoError(t, err)
		assert.NotNil(t, draft)

		templateDraft := draft.(*models.TemplateInfoData)
		assert.Equal(t, template.ID, templateDraft.ID)
		assert.Equal(t, template.TemplateName, templateDraft.TemplateName)
	})

	t.Run("Create FPR draft", func(t *testing.T) {
		fpr := &models.FocomProvisioningRequestData{
			BaseResource:       models.NewBaseResource("default", "test-fpr", "Test FPR"),
			OCloudID:           "ocloud-1",
			OCloudNamespace:    "default",
			TemplateName:       "cluster-template",
			TemplateVersion:    "v1.0.0",
			TemplateParameters: map[string]interface{}{"param1": "value1"},
		}

		err := storage.CreateDraft(ctx, ResourceTypeFocomProvisioningRequest, fpr)
		assert.NoError(t, err)

		// Verify draft was created
		draft, err := storage.GetDraft(ctx, ResourceTypeFocomProvisioningRequest, fpr.ID)
		assert.NoError(t, err)
		assert.NotNil(t, draft)

		fprDraft := draft.(*models.FocomProvisioningRequestData)
		assert.Equal(t, fpr.ID, fprDraft.ID)
		assert.Equal(t, fpr.OCloudID, fprDraft.OCloudID)
	})

	t.Run("Create duplicate draft should fail", func(t *testing.T) {
		ocloud := &models.OCloudData{
			BaseResource: models.NewBaseResource("default", "duplicate-test", "Duplicate Test"),
		}

		// Create first draft
		err := storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
		assert.NoError(t, err)

		// Try to create duplicate
		err = storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestInMemoryStorage_GetDraft(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	t.Run("Get existing draft", func(t *testing.T) {
		ocloud := &models.OCloudData{
			BaseResource: models.NewBaseResource("default", "test-ocloud", "Test OCloud"),
		}

		err := storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
		require.NoError(t, err)

		draft, err := storage.GetDraft(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.NoError(t, err)
		assert.NotNil(t, draft)

		ocloudDraft := draft.(*models.OCloudData)
		assert.Equal(t, ocloud.ID, ocloudDraft.ID)
	})

	t.Run("Get non-existent draft", func(t *testing.T) {
		draft, err := storage.GetDraft(ctx, ResourceTypeOCloud, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, draft)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestInMemoryStorage_UpdateDraft(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	t.Run("Update existing draft", func(t *testing.T) {
		ocloud := &models.OCloudData{
			BaseResource: models.NewBaseResource("default", "test-ocloud", "Test OCloud"),
		}

		err := storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
		require.NoError(t, err)

		// Update the draft
		ocloud.Description = "Updated description"
		ocloud.State = models.StateValidated

		err = storage.UpdateDraft(ctx, ResourceTypeOCloud, ocloud.ID, ocloud)
		assert.NoError(t, err)

		// Verify update
		draft, err := storage.GetDraft(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.NoError(t, err)

		ocloudDraft := draft.(*models.OCloudData)
		assert.Equal(t, "Updated description", ocloudDraft.Description)
		assert.Equal(t, models.StateValidated, ocloudDraft.State)
	})

	t.Run("Update non-existent draft", func(t *testing.T) {
		ocloud := &models.OCloudData{
			BaseResource: models.NewBaseResource("default", "non-existent", "Non-existent"),
		}

		err := storage.UpdateDraft(ctx, ResourceTypeOCloud, "non-existent", ocloud)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestInMemoryStorage_DeleteDraft(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	t.Run("Delete existing draft", func(t *testing.T) {
		ocloud := &models.OCloudData{
			BaseResource: models.NewBaseResource("default", "test-ocloud", "Test OCloud"),
		}

		err := storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
		require.NoError(t, err)

		err = storage.DeleteDraft(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.NoError(t, err)

		// Verify deletion
		draft, err := storage.GetDraft(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.Error(t, err)
		assert.Nil(t, draft)
	})

	t.Run("Delete non-existent draft", func(t *testing.T) {
		err := storage.DeleteDraft(ctx, ResourceTypeOCloud, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestInMemoryStorage_ApproveDraft(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	t.Run("Approve validated draft", func(t *testing.T) {
		ocloud := &models.OCloudData{
			BaseResource: models.NewBaseResource("default", "test-ocloud", "Test OCloud"),
		}

		err := storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
		require.NoError(t, err)

		// Update state to VALIDATED
		err = storage.UpdateDraftState(ctx, ResourceTypeOCloud, ocloud.ID, StateValidated)
		require.NoError(t, err)

		err = storage.ApproveDraft(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.NoError(t, err)

		// Verify approved resource exists
		resource, err := storage.Get(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.NoError(t, err)
		assert.NotNil(t, resource)

		ocloudResource := resource.(*models.OCloudData)
		assert.Equal(t, models.StateApproved, ocloudResource.State)

		// Verify revision was created
		revisions, err := storage.GetRevisions(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.NoError(t, err)
		assert.Len(t, revisions, 1)
	})

	t.Run("Approve non-validated draft should fail", func(t *testing.T) {
		ocloud := &models.OCloudData{
			BaseResource: models.NewBaseResource("default", "draft-ocloud", "Draft OCloud"),
		}
		// Keep state as DRAFT

		err := storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
		require.NoError(t, err)

		err = storage.ApproveDraft(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "VALIDATED state")
	})

	t.Run("Approve non-existent draft", func(t *testing.T) {
		err := storage.ApproveDraft(ctx, ResourceTypeOCloud, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestInMemoryStorage_GetAndList(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Create and approve multiple resources
	resources := []*models.OCloudData{
		{BaseResource: models.NewBaseResource("default", "ocloud-1", "OCloud 1")},
		{BaseResource: models.NewBaseResource("default", "ocloud-2", "OCloud 2")},
		{BaseResource: models.NewBaseResource("default", "ocloud-3", "OCloud 3")},
	}

	for _, ocloud := range resources {
		err := storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
		require.NoError(t, err)

		err = storage.UpdateDraftState(ctx, ResourceTypeOCloud, ocloud.ID, StateValidated)
		require.NoError(t, err)

		err = storage.ApproveDraft(ctx, ResourceTypeOCloud, ocloud.ID)
		require.NoError(t, err)
	}

	t.Run("Get specific resource", func(t *testing.T) {
		resource, err := storage.Get(ctx, ResourceTypeOCloud, resources[0].ID)
		assert.NoError(t, err)
		assert.NotNil(t, resource)

		ocloud := resource.(*models.OCloudData)
		assert.Equal(t, resources[0].ID, ocloud.ID)
		assert.Equal(t, "ocloud-1", ocloud.Name)
	})

	t.Run("Get non-existent resource", func(t *testing.T) {
		resource, err := storage.Get(ctx, ResourceTypeOCloud, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, resource)
	})

	t.Run("List all resources", func(t *testing.T) {
		resourceList, err := storage.List(ctx, ResourceTypeOCloud)
		assert.NoError(t, err)
		assert.Len(t, resourceList, 3)

		// Verify all resources are present
		names := make(map[string]bool)
		for _, resource := range resourceList {
			ocloud := resource.(*models.OCloudData)
			names[ocloud.Name] = true
		}

		assert.True(t, names["ocloud-1"])
		assert.True(t, names["ocloud-2"])
		assert.True(t, names["ocloud-3"])
	})
}

func TestInMemoryStorage_Revisions(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	ocloud := &models.OCloudData{
		BaseResource: models.NewBaseResource("default", "test-ocloud", "Test OCloud"),
	}

	// Create and approve first revision
	err := storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
	require.NoError(t, err)

	err = storage.UpdateDraftState(ctx, ResourceTypeOCloud, ocloud.ID, StateValidated)
	require.NoError(t, err)

	err = storage.ApproveDraft(ctx, ResourceTypeOCloud, ocloud.ID)
	require.NoError(t, err)

	firstRevisionID := ocloud.RevisionID

	t.Run("Get revisions", func(t *testing.T) {
		revisions, err := storage.GetRevisions(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.NoError(t, err)
		assert.Len(t, revisions, 1)

		// GetRevisions now returns the actual revision data, not the RevisionStorage wrapper
		revisionData := revisions[0].(*models.OCloudData)
		assert.Equal(t, ocloud.ID, revisionData.ID)
		assert.Equal(t, firstRevisionID, revisionData.RevisionID)
	})

	t.Run("Get specific revision", func(t *testing.T) {
		revision, err := storage.GetRevision(ctx, ResourceTypeOCloud, ocloud.ID, firstRevisionID)
		assert.NoError(t, err)
		assert.NotNil(t, revision)

		revisionStorage := revision.(*models.RevisionStorage)
		assert.Equal(t, ocloud.ID, revisionStorage.ResourceID)
		assert.Equal(t, firstRevisionID, revisionStorage.RevisionID)

		// Check the actual revision data
		revisionData := revisionStorage.RevisionData.(*models.OCloudData)
		assert.Equal(t, ocloud.ID, revisionData.ID)
		assert.Equal(t, firstRevisionID, revisionData.RevisionID)
	})

	t.Run("Create draft from revision", func(t *testing.T) {
		err := storage.CreateDraftFromRevision(ctx, ResourceTypeOCloud, ocloud.ID, firstRevisionID)
		assert.NoError(t, err)

		// Verify draft was created
		draft, err := storage.GetDraft(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.NoError(t, err)
		assert.NotNil(t, draft)

		ocloudDraft := draft.(*models.OCloudData)
		assert.Equal(t, ocloud.ID, ocloudDraft.ID)
		assert.Equal(t, models.StateDraft, ocloudDraft.State)
	})

	t.Run("Get non-existent revision", func(t *testing.T) {
		revision, err := storage.GetRevision(ctx, ResourceTypeOCloud, ocloud.ID, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, revision)
	})
}

func TestInMemoryStorage_Delete(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	ocloud := &models.OCloudData{
		BaseResource: models.NewBaseResource("default", "test-ocloud", "Test OCloud"),
	}

	// Create and approve resource
	err := storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
	require.NoError(t, err)

	err = storage.UpdateDraftState(ctx, ResourceTypeOCloud, ocloud.ID, StateValidated)
	require.NoError(t, err)

	err = storage.ApproveDraft(ctx, ResourceTypeOCloud, ocloud.ID)
	require.NoError(t, err)

	t.Run("Delete existing resource", func(t *testing.T) {
		err := storage.Delete(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.NoError(t, err)

		// Verify resource is deleted
		resource, err := storage.Get(ctx, ResourceTypeOCloud, ocloud.ID)
		assert.Error(t, err)
		assert.Nil(t, resource)
	})

	t.Run("Delete non-existent resource", func(t *testing.T) {
		err := storage.Delete(ctx, ResourceTypeOCloud, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestInMemoryStorage_UpdateDraftState(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	ocloud := &models.OCloudData{
		BaseResource: models.NewBaseResource("default", "test-ocloud", "Test OCloud"),
	}

	err := storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
	require.NoError(t, err)

	t.Run("Update draft state", func(t *testing.T) {
		err := storage.UpdateDraftState(ctx, ResourceTypeOCloud, ocloud.ID, StateValidated)
		assert.NoError(t, err)

		// The UpdateDraftState method only updates the DraftStorage state, not the resource data state
		// This is by design - the resource data state should be updated through other methods
		// So we don't check the resource data state here, just that the method succeeds
	})

	t.Run("Update non-existent draft state", func(t *testing.T) {
		err := storage.UpdateDraftState(ctx, ResourceTypeOCloud, "non-existent", StateValidated)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestInMemoryStorage_Clear(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Create some resources
	ocloud := &models.OCloudData{
		BaseResource: models.NewBaseResource("default", "test-ocloud", "Test OCloud"),
	}

	err := storage.CreateDraft(ctx, ResourceTypeOCloud, ocloud)
	require.NoError(t, err)

	err = storage.UpdateDraftState(ctx, ResourceTypeOCloud, ocloud.ID, StateValidated)
	require.NoError(t, err)

	err = storage.ApproveDraft(ctx, ResourceTypeOCloud, ocloud.ID)
	require.NoError(t, err)

	// Verify resource exists
	resource, err := storage.Get(ctx, ResourceTypeOCloud, ocloud.ID)
	assert.NoError(t, err)
	assert.NotNil(t, resource)

	// Clear storage
	storage.Clear()

	// Verify resource is gone
	resource, err = storage.Get(ctx, ResourceTypeOCloud, ocloud.ID)
	assert.Error(t, err)
	assert.Nil(t, resource)

	// Verify list is empty
	resources, err := storage.List(ctx, ResourceTypeOCloud)
	assert.NoError(t, err)
	assert.Empty(t, resources)
}
