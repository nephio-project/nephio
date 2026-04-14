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
	"errors"
	"testing"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/integration"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockStorageInterface is a mock implementation of storage.StorageInterface
type MockStorageInterface struct {
	mock.Mock
}

func (m *MockStorageInterface) Create(ctx context.Context, resourceType storage.ResourceType, resource interface{}) error {
	args := m.Called(ctx, resourceType, resource)
	return args.Error(0)
}

func (m *MockStorageInterface) CreateDraft(ctx context.Context, resourceType storage.ResourceType, draft interface{}) error {
	args := m.Called(ctx, resourceType, draft)
	return args.Error(0)
}

func (m *MockStorageInterface) GetDraft(ctx context.Context, resourceType storage.ResourceType, id string) (interface{}, error) {
	args := m.Called(ctx, resourceType, id)
	return args.Get(0), args.Error(1)
}

func (m *MockStorageInterface) UpdateDraft(ctx context.Context, resourceType storage.ResourceType, id string, draft interface{}) error {
	args := m.Called(ctx, resourceType, id, draft)
	return args.Error(0)
}

func (m *MockStorageInterface) DeleteDraft(ctx context.Context, resourceType storage.ResourceType, id string) error {
	args := m.Called(ctx, resourceType, id)
	return args.Error(0)
}

func (m *MockStorageInterface) ValidateDraft(ctx context.Context, resourceType storage.ResourceType, id string) error {
	args := m.Called(ctx, resourceType, id)
	return args.Error(0)
}

func (m *MockStorageInterface) ApproveDraft(ctx context.Context, resourceType storage.ResourceType, id string) error {
	args := m.Called(ctx, resourceType, id)
	return args.Error(0)
}

func (m *MockStorageInterface) RejectDraft(ctx context.Context, resourceType storage.ResourceType, id string) error {
	args := m.Called(ctx, resourceType, id)
	return args.Error(0)
}

func (m *MockStorageInterface) Get(ctx context.Context, resourceType storage.ResourceType, id string) (interface{}, error) {
	args := m.Called(ctx, resourceType, id)
	return args.Get(0), args.Error(1)
}

func (m *MockStorageInterface) Update(ctx context.Context, resourceType storage.ResourceType, id string, resource interface{}) error {
	args := m.Called(ctx, resourceType, id, resource)
	return args.Error(0)
}

func (m *MockStorageInterface) List(ctx context.Context, resourceType storage.ResourceType) ([]interface{}, error) {
	args := m.Called(ctx, resourceType)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockStorageInterface) Delete(ctx context.Context, resourceType storage.ResourceType, id string) error {
	args := m.Called(ctx, resourceType, id)
	return args.Error(0)
}

func (m *MockStorageInterface) GetRevisions(ctx context.Context, resourceType storage.ResourceType, id string) ([]interface{}, error) {
	args := m.Called(ctx, resourceType, id)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockStorageInterface) GetRevision(ctx context.Context, resourceType storage.ResourceType, id, revisionId string) (interface{}, error) {
	args := m.Called(ctx, resourceType, id, revisionId)
	return args.Get(0), args.Error(1)
}

func (m *MockStorageInterface) CreateRevision(ctx context.Context, resourceType storage.ResourceType, resourceID string, revisionID string, data interface{}) error {
	args := m.Called(ctx, resourceType, resourceID, revisionID, data)
	return args.Error(0)
}

func (m *MockStorageInterface) CreateDraftFromRevision(ctx context.Context, resourceType storage.ResourceType, id, revisionId string) error {
	args := m.Called(ctx, resourceType, id, revisionId)
	return args.Error(0)
}

func (m *MockStorageInterface) UpdateDraftState(ctx context.Context, resourceType storage.ResourceType, id string, state storage.ResourceState) error {
	args := m.Called(ctx, resourceType, id, state)
	return args.Error(0)
}

func (m *MockStorageInterface) ValidateDependencies(ctx context.Context, resourceType storage.ResourceType, resource interface{}) error {
	args := m.Called(ctx, resourceType, resource)
	return args.Error(0)
}

func (m *MockStorageInterface) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockValidator is a mock implementation of validation.Validator
type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) ValidateOCloud(ctx context.Context, ocloud *models.OCloudData) *models.ValidationResult {
	args := m.Called(ctx, ocloud)
	return args.Get(0).(*models.ValidationResult)
}

func (m *MockValidator) ValidateTemplateInfo(ctx context.Context, templateInfo *models.TemplateInfoData) *models.ValidationResult {
	args := m.Called(ctx, templateInfo)
	return args.Get(0).(*models.ValidationResult)
}

func (m *MockValidator) ValidateFocomProvisioningRequest(ctx context.Context, fpr *models.FocomProvisioningRequestData) *models.ValidationResult {
	args := m.Called(ctx, fpr)
	return args.Get(0).(*models.ValidationResult)
}

// TODO: Template parameter validation is temporarily disabled
// func (m *MockValidator) ValidateTemplateParameters(ctx context.Context, parameters map[string]interface{}, schema string) *models.ValidationResult {
// 	args := m.Called(ctx, parameters, schema)
// 	return args.Get(0).(*models.ValidationResult)
// }

func (m *MockValidator) ValidateTemplateParameters(ctx context.Context, parameters map[string]interface{}, schema string) *models.ValidationResult {
	// Return success with warning that validation is disabled
	return &models.ValidationResult{
		Success:  true,
		Errors:   []string{},
		Warnings: []string{"Template parameter schema validation is temporarily disabled"},
	}
}

func (m *MockValidator) ValidateJSON(jsonStr string) error {
	args := m.Called(jsonStr)
	return args.Error(0)
}

func (m *MockValidator) ValidateYAML(yamlStr string) error {
	args := m.Called(yamlStr)
	return args.Error(0)
}

// MockOperatorIntegration is a mock implementation of integration.OperatorIntegration
type MockOperatorIntegration struct {
	mock.Mock
}

func (m *MockOperatorIntegration) CreateOCloudCR(ctx context.Context, ocloud *models.OCloudData) error {
	args := m.Called(ctx, ocloud)
	return args.Error(0)
}

func (m *MockOperatorIntegration) CreateTemplateInfoCR(ctx context.Context, templateInfo *models.TemplateInfoData) error {
	args := m.Called(ctx, templateInfo)
	return args.Error(0)
}

func (m *MockOperatorIntegration) CreateFocomProvisioningRequestCR(ctx context.Context, fpr *models.FocomProvisioningRequestData) error {
	args := m.Called(ctx, fpr)
	return args.Error(0)
}

func (m *MockOperatorIntegration) GetKubernetesClient() client.Client {
	args := m.Called()
	return args.Get(0).(client.Client)
}

func (m *MockOperatorIntegration) CreateO2IMSProvisioningRequest(ctx context.Context, request *integration.O2IMSProvisioningRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

func (m *MockOperatorIntegration) UpdateO2IMSProvisioningRequest(ctx context.Context, id string, request *integration.O2IMSProvisioningRequest) error {
	args := m.Called(ctx, id, request)
	return args.Error(0)
}

func (m *MockOperatorIntegration) DeleteO2IMSProvisioningRequest(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOperatorIntegration) GetO2IMSProvisioningStatus(ctx context.Context, id string) (*integration.O2IMSProvisioningStatus, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*integration.O2IMSProvisioningStatus), args.Error(1)
}

func (m *MockOperatorIntegration) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestOCloudService_CreateDraft(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful draft creation", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		ocloud := models.NewOCloudData("default", "test-ocloud", "Test OCloud", models.O2IMSSecretRef{
			SecretRef: models.SecretReference{
				Name:      "test-secret",
				Namespace: "default",
			},
		})

		mockStorage.On("CreateDraft", ctx, storage.ResourceTypeOCloud, mock.AnythingOfType("*models.OCloudData")).Return(nil)

		result, err := service.CreateDraft(ctx, ocloud)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-ocloud", result.Name)
		assert.Equal(t, models.StateDraft, result.State)
		assert.NotEmpty(t, result.ID)
		assert.NotEmpty(t, result.RevisionID)

		mockStorage.AssertExpectations(t)
	})

	t.Run("Storage error", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		ocloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
			},
		}

		mockStorage.On("CreateDraft", ctx, storage.ResourceTypeOCloud, mock.AnythingOfType("*models.OCloudData")).Return(errors.New("storage error"))

		result, err := service.CreateDraft(ctx, ocloud)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to create OCloud draft")

		mockStorage.AssertExpectations(t)
	})
}

func TestOCloudService_GetDraft(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful draft retrieval", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		expectedOCloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "test-id",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateDraft,
			},
		}

		mockStorage.On("GetDraft", ctx, storage.ResourceTypeOCloud, "test-id").Return(expectedOCloud, nil)

		result, err := service.GetDraft(ctx, "test-id")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-id", result.ID)
		assert.Equal(t, "test-ocloud", result.Name)

		mockStorage.AssertExpectations(t)
	})

	t.Run("Draft not found", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		mockStorage.On("GetDraft", ctx, storage.ResourceTypeOCloud, "non-existent").Return(nil, errors.New("not found"))

		result, err := service.GetDraft(ctx, "non-existent")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get OCloud draft")

		mockStorage.AssertExpectations(t)
	})
}

func TestOCloudService_UpdateDraft(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful draft update", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		existingOCloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "test-id",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Original description",
				State:       models.StateDraft,
			},
		}

		updates := &models.OCloudData{
			BaseResource: models.BaseResource{
				Name:        "updated-ocloud",
				Description: "Updated description",
			},
			O2IMSSecret: models.O2IMSSecretRef{
				SecretRef: models.SecretReference{
					Name:      "updated-secret",
					Namespace: "default",
				},
			},
		}

		mockStorage.On("GetDraft", ctx, storage.ResourceTypeOCloud, "test-id").Return(existingOCloud, nil)
		mockStorage.On("UpdateDraft", ctx, storage.ResourceTypeOCloud, "test-id", mock.AnythingOfType("*models.OCloudData")).Return(nil)

		result, err := service.UpdateDraft(ctx, "test-id", updates)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "updated-ocloud", result.Name)
		assert.Equal(t, "Updated description", result.Description)

		mockStorage.AssertExpectations(t)
	})

	t.Run("Cannot update validated draft", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		existingOCloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "test-id",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Original description",
				State:       models.StateValidated,
			},
		}

		updates := &models.OCloudData{
			BaseResource: models.BaseResource{
				Description: "Updated description",
			},
		}

		mockStorage.On("GetDraft", ctx, storage.ResourceTypeOCloud, "test-id").Return(existingOCloud, nil)

		result, err := service.UpdateDraft(ctx, "test-id", updates)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrInvalidState, err)

		mockStorage.AssertExpectations(t)
	})
}

func TestOCloudService_ValidateDraft(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful validation", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		ocloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "test-id",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateDraft,
			},
		}

		validationResult := &models.ValidationResult{
			Success: true,
			Errors:  []string{},
		}

		mockStorage.On("GetDraft", ctx, storage.ResourceTypeOCloud, "test-id").Return(ocloud, nil)
		mockValidator.On("ValidateOCloud", ctx, ocloud).Return(validationResult)
		mockStorage.On("UpdateDraft", ctx, storage.ResourceTypeOCloud, "test-id", mock.AnythingOfType("*models.OCloudData")).Return(nil)

		result, err := service.ValidateDraft(ctx, "test-id")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)

		mockStorage.AssertExpectations(t)
		mockValidator.AssertExpectations(t)
	})

	t.Run("Validation failure", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		ocloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "test-id",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateDraft,
			},
		}

		validationResult := &models.ValidationResult{
			Success: false,
			Errors:  []string{"Invalid configuration"},
		}

		mockStorage.On("GetDraft", ctx, storage.ResourceTypeOCloud, "test-id").Return(ocloud, nil)
		mockValidator.On("ValidateOCloud", ctx, ocloud).Return(validationResult)

		result, err := service.ValidateDraft(ctx, "test-id")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Contains(t, result.Errors, "Invalid configuration")

		mockStorage.AssertExpectations(t)
		mockValidator.AssertExpectations(t)
	})
}

func TestOCloudService_ApproveDraft(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful approval", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		draftOCloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "test-id",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateValidated,
			},
		}

		approvedOCloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "test-id",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateApproved,
			},
		}

		mockStorage.On("GetDraft", ctx, storage.ResourceTypeOCloud, "test-id").Return(draftOCloud, nil)
		mockStorage.On("ApproveDraft", ctx, storage.ResourceTypeOCloud, "test-id").Return(nil)
		mockStorage.On("Get", ctx, storage.ResourceTypeOCloud, "test-id").Return(approvedOCloud, nil)

		result, err := service.ApproveDraft(ctx, "test-id")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, models.StateApproved, result.State)

		mockStorage.AssertExpectations(t)
	})

	t.Run("Cannot approve non-validated draft", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		draftOCloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "test-id",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateDraft,
			},
		}

		mockStorage.On("GetDraft", ctx, storage.ResourceTypeOCloud, "test-id").Return(draftOCloud, nil)

		result, err := service.ApproveDraft(ctx, "test-id")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrInvalidState, err)

		mockStorage.AssertExpectations(t)
	})
}

func TestOCloudService_RejectDraft(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful rejection", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		ocloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "test-id",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateValidated,
			},
		}

		mockStorage.On("GetDraft", ctx, storage.ResourceTypeOCloud, "test-id").Return(ocloud, nil)
		mockStorage.On("UpdateDraft", ctx, storage.ResourceTypeOCloud, "test-id", mock.AnythingOfType("*models.OCloudData")).Return(nil)

		err := service.RejectDraft(ctx, "test-id")

		assert.NoError(t, err)

		mockStorage.AssertExpectations(t)
	})
}

func TestOCloudService_DeleteDraft(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful deletion", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		mockStorage.On("DeleteDraft", ctx, storage.ResourceTypeOCloud, "test-id").Return(nil)

		err := service.DeleteDraft(ctx, "test-id")

		assert.NoError(t, err)

		mockStorage.AssertExpectations(t)
	})

	t.Run("Storage error", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		mockStorage.On("DeleteDraft", ctx, storage.ResourceTypeOCloud, "test-id").Return(errors.New("storage error"))

		err := service.DeleteDraft(ctx, "test-id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete OCloud draft")

		mockStorage.AssertExpectations(t)
	})
}

func TestOCloudService_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful retrieval", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		expectedOCloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "test-id",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateApproved,
			},
		}

		mockStorage.On("Get", ctx, storage.ResourceTypeOCloud, "test-id").Return(expectedOCloud, nil)

		result, err := service.Get(ctx, "test-id")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-id", result.ID)
		assert.Equal(t, models.StateApproved, result.State)

		mockStorage.AssertExpectations(t)
	})
}

func TestOCloudService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful listing", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		expectedOClouds := []interface{}{
			&models.OCloudData{
				BaseResource: models.BaseResource{
					ID:   "ocloud-1",
					Name: "OCloud 1",
				},
			},
			&models.OCloudData{
				BaseResource: models.BaseResource{
					ID:   "ocloud-2",
					Name: "OCloud 2",
				},
			},
		}

		mockStorage.On("List", ctx, storage.ResourceTypeOCloud).Return(expectedOClouds, nil)

		result, err := service.List(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 2)
		assert.Equal(t, "ocloud-1", result[0].ID)
		assert.Equal(t, "ocloud-2", result[1].ID)

		mockStorage.AssertExpectations(t)
	})
}

func TestOCloudService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful deletion", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		ocloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "test-id",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateApproved,
			},
		}

		mockStorage.On("Get", ctx, storage.ResourceTypeOCloud, "test-id").Return(ocloud, nil)
		mockStorage.On("ValidateDependencies", ctx, storage.ResourceTypeOCloud, ocloud).Return(nil)
		mockStorage.On("Delete", ctx, storage.ResourceTypeOCloud, "test-id").Return(nil)

		err := service.Delete(ctx, "test-id")

		assert.NoError(t, err)

		mockStorage.AssertExpectations(t)
	})

	t.Run("Cannot delete due to dependencies", func(t *testing.T) {
		mockStorage := new(MockStorageInterface)
		mockValidator := new(MockValidator)
		mockIntegration := new(MockOperatorIntegration)

		service := NewOCloudService(mockStorage, mockValidator, mockIntegration)

		ocloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "test-id",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateApproved,
			},
		}

		mockStorage.On("Get", ctx, storage.ResourceTypeOCloud, "test-id").Return(ocloud, nil)
		mockStorage.On("ValidateDependencies", ctx, storage.ResourceTypeOCloud, ocloud).Return(errors.New("has dependencies"))

		err := service.Delete(ctx, "test-id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete OCloud due to dependencies")

		mockStorage.AssertExpectations(t)
	})
}
