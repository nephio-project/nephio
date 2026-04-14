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

package validation

import (
	"context"
	"errors"
	"testing"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockResourceLookup is a mock implementation of ResourceLookup for testing
type MockResourceLookup struct {
	mock.Mock
}

func (m *MockResourceLookup) GetOCloud(ctx context.Context, id, namespace string) (*models.OCloudData, error) {
	args := m.Called(ctx, id, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OCloudData), args.Error(1)
}

func (m *MockResourceLookup) GetTemplateInfo(ctx context.Context, name, version string) (*models.TemplateInfoData, error) {
	args := m.Called(ctx, name, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TemplateInfoData), args.Error(1)
}

func (m *MockResourceLookup) ListFocomProvisioningRequestsByOCloud(ctx context.Context, oCloudID, oCloudNamespace string) ([]*models.FocomProvisioningRequestData, error) {
	args := m.Called(ctx, oCloudID, oCloudNamespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FocomProvisioningRequestData), args.Error(1)
}

func (m *MockResourceLookup) ListFocomProvisioningRequestsByTemplate(ctx context.Context, templateName, templateVersion string) ([]*models.FocomProvisioningRequestData, error) {
	args := m.Called(ctx, templateName, templateVersion)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FocomProvisioningRequestData), args.Error(1)
}

func TestDependencyValidator_ValidateFocomProvisioningRequestDependencies(t *testing.T) {
	ctx := context.Background()

	t.Run("Valid dependencies - all exist and approved", func(t *testing.T) {
		mockLookup := new(MockResourceLookup)
		validator := NewDependencyValidator(mockLookup)

		// Create test FPR
		fpr := &models.FocomProvisioningRequestData{
			BaseResource: models.BaseResource{
				ID:          "fpr-1",
				Namespace:   "default",
				Name:        "test-fpr",
				Description: "Test FPR",
				State:       models.StateDraft,
			},
			OCloudID:        "ocloud-1",
			OCloudNamespace: "default",
			TemplateName:    "test-template",
			TemplateVersion: "v1.0.0",
		}

		// Mock approved OCloud
		approvedOCloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "ocloud-1",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateApproved,
			},
		}

		// Mock approved TemplateInfo
		approvedTemplate := &models.TemplateInfoData{
			BaseResource: models.BaseResource{
				ID:          "template-1",
				Namespace:   "default",
				Name:        "test-template",
				Description: "Test Template",
				State:       models.StateApproved,
			},
			TemplateName:    "test-template",
			TemplateVersion: "v1.0.0",
		}

		mockLookup.On("GetOCloud", ctx, "ocloud-1", "default").Return(approvedOCloud, nil)
		mockLookup.On("GetTemplateInfo", ctx, "test-template", "v1.0.0").Return(approvedTemplate, nil)

		result := validator.ValidateFocomProvisioningRequestDependencies(ctx, fpr)

		assert.True(t, result.Success)
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.Warnings)
		mockLookup.AssertExpectations(t)
	})

	t.Run("Missing OCloud dependency", func(t *testing.T) {
		mockLookup := new(MockResourceLookup)
		validator := NewDependencyValidator(mockLookup)

		fpr := &models.FocomProvisioningRequestData{
			BaseResource: models.BaseResource{
				ID:          "fpr-1",
				Namespace:   "default",
				Name:        "test-fpr",
				Description: "Test FPR",
				State:       models.StateDraft,
			},
			OCloudID:        "missing-ocloud",
			OCloudNamespace: "default",
			TemplateName:    "test-template",
			TemplateVersion: "v1.0.0",
		}

		// Mock approved TemplateInfo
		approvedTemplate := &models.TemplateInfoData{
			BaseResource: models.BaseResource{
				ID:          "template-1",
				Namespace:   "default",
				Name:        "test-template",
				Description: "Test Template",
				State:       models.StateApproved,
			},
			TemplateName:    "test-template",
			TemplateVersion: "v1.0.0",
		}

		mockLookup.On("GetOCloud", ctx, "missing-ocloud", "default").Return(nil, nil)
		mockLookup.On("GetTemplateInfo", ctx, "test-template", "v1.0.0").Return(approvedTemplate, nil)

		result := validator.ValidateFocomProvisioningRequestDependencies(ctx, fpr)

		assert.False(t, result.Success)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Referenced OCloud 'missing-ocloud' in namespace 'default' does not exist")
		mockLookup.AssertExpectations(t)
	})

	t.Run("Missing TemplateInfo dependency", func(t *testing.T) {
		mockLookup := new(MockResourceLookup)
		validator := NewDependencyValidator(mockLookup)

		fpr := &models.FocomProvisioningRequestData{
			BaseResource: models.BaseResource{
				ID:          "fpr-1",
				Namespace:   "default",
				Name:        "test-fpr",
				Description: "Test FPR",
				State:       models.StateDraft,
			},
			OCloudID:        "ocloud-1",
			OCloudNamespace: "default",
			TemplateName:    "missing-template",
			TemplateVersion: "v1.0.0",
		}

		// Mock approved OCloud
		approvedOCloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "ocloud-1",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateApproved,
			},
		}

		mockLookup.On("GetOCloud", ctx, "ocloud-1", "default").Return(approvedOCloud, nil)
		mockLookup.On("GetTemplateInfo", ctx, "missing-template", "v1.0.0").Return(nil, nil)

		result := validator.ValidateFocomProvisioningRequestDependencies(ctx, fpr)

		assert.False(t, result.Success)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Referenced TemplateInfo 'missing-template' version 'v1.0.0' does not exist")
		mockLookup.AssertExpectations(t)
	})

	t.Run("OCloud not in approved state", func(t *testing.T) {
		mockLookup := new(MockResourceLookup)
		validator := NewDependencyValidator(mockLookup)

		fpr := &models.FocomProvisioningRequestData{
			BaseResource: models.BaseResource{
				ID:          "fpr-1",
				Namespace:   "default",
				Name:        "test-fpr",
				Description: "Test FPR",
				State:       models.StateDraft,
			},
			OCloudID:        "ocloud-1",
			OCloudNamespace: "default",
			TemplateName:    "test-template",
			TemplateVersion: "v1.0.0",
		}

		// Mock draft OCloud (not approved)
		draftOCloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "ocloud-1",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateDraft,
			},
		}

		// Mock approved TemplateInfo
		approvedTemplate := &models.TemplateInfoData{
			BaseResource: models.BaseResource{
				ID:          "template-1",
				Namespace:   "default",
				Name:        "test-template",
				Description: "Test Template",
				State:       models.StateApproved,
			},
			TemplateName:    "test-template",
			TemplateVersion: "v1.0.0",
		}

		mockLookup.On("GetOCloud", ctx, "ocloud-1", "default").Return(draftOCloud, nil)
		mockLookup.On("GetTemplateInfo", ctx, "test-template", "v1.0.0").Return(approvedTemplate, nil)

		result := validator.ValidateFocomProvisioningRequestDependencies(ctx, fpr)

		assert.False(t, result.Success)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Referenced OCloud 'ocloud-1' in namespace 'default' is not in approved state (current state: DRAFT)")
		mockLookup.AssertExpectations(t)
	})

	t.Run("TemplateInfo not in approved state", func(t *testing.T) {
		mockLookup := new(MockResourceLookup)
		validator := NewDependencyValidator(mockLookup)

		fpr := &models.FocomProvisioningRequestData{
			BaseResource: models.BaseResource{
				ID:          "fpr-1",
				Namespace:   "default",
				Name:        "test-fpr",
				Description: "Test FPR",
				State:       models.StateDraft,
			},
			OCloudID:        "ocloud-1",
			OCloudNamespace: "default",
			TemplateName:    "test-template",
			TemplateVersion: "v1.0.0",
		}

		// Mock approved OCloud
		approvedOCloud := &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          "ocloud-1",
				Namespace:   "default",
				Name:        "test-ocloud",
				Description: "Test OCloud",
				State:       models.StateApproved,
			},
		}

		// Mock validated TemplateInfo (not approved)
		validatedTemplate := &models.TemplateInfoData{
			BaseResource: models.BaseResource{
				ID:          "template-1",
				Namespace:   "default",
				Name:        "test-template",
				Description: "Test Template",
				State:       models.StateValidated,
			},
			TemplateName:    "test-template",
			TemplateVersion: "v1.0.0",
		}

		mockLookup.On("GetOCloud", ctx, "ocloud-1", "default").Return(approvedOCloud, nil)
		mockLookup.On("GetTemplateInfo", ctx, "test-template", "v1.0.0").Return(validatedTemplate, nil)

		result := validator.ValidateFocomProvisioningRequestDependencies(ctx, fpr)

		assert.False(t, result.Success)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Referenced TemplateInfo 'test-template' version 'v1.0.0' is not in approved state (current state: VALIDATED)")
		mockLookup.AssertExpectations(t)
	})

	t.Run("Lookup errors", func(t *testing.T) {
		mockLookup := new(MockResourceLookup)
		validator := NewDependencyValidator(mockLookup)

		fpr := &models.FocomProvisioningRequestData{
			BaseResource: models.BaseResource{
				ID:          "fpr-1",
				Namespace:   "default",
				Name:        "test-fpr",
				Description: "Test FPR",
				State:       models.StateDraft,
			},
			OCloudID:        "ocloud-1",
			OCloudNamespace: "default",
			TemplateName:    "test-template",
			TemplateVersion: "v1.0.0",
		}

		mockLookup.On("GetOCloud", ctx, "ocloud-1", "default").Return(nil, errors.New("database error"))
		mockLookup.On("GetTemplateInfo", ctx, "test-template", "v1.0.0").Return(nil, errors.New("network error"))

		result := validator.ValidateFocomProvisioningRequestDependencies(ctx, fpr)

		assert.False(t, result.Success)
		assert.Len(t, result.Errors, 2)
		assert.Contains(t, result.Errors[0], "Failed to lookup OCloud 'ocloud-1' in namespace 'default': database error")
		assert.Contains(t, result.Errors[1], "Failed to lookup TemplateInfo 'test-template' version 'v1.0.0': network error")
		mockLookup.AssertExpectations(t)
	})
}

func TestDependencyValidator_ValidateOCloudDeletion(t *testing.T) {
	ctx := context.Background()

	t.Run("Can delete OCloud with no dependencies", func(t *testing.T) {
		mockLookup := new(MockResourceLookup)
		validator := NewDependencyValidator(mockLookup)

		mockLookup.On("ListFocomProvisioningRequestsByOCloud", ctx, "ocloud-1", "default").Return([]*models.FocomProvisioningRequestData{}, nil)

		result := validator.ValidateOCloudDeletion(ctx, "ocloud-1", "default")

		assert.True(t, result.Success)
		assert.Empty(t, result.Errors)
		mockLookup.AssertExpectations(t)
	})

	t.Run("Cannot delete OCloud with approved FPR dependencies", func(t *testing.T) {
		mockLookup := new(MockResourceLookup)
		validator := NewDependencyValidator(mockLookup)

		dependentFPRs := []*models.FocomProvisioningRequestData{
			{
				BaseResource: models.BaseResource{
					ID:    "fpr-1",
					State: models.StateApproved,
				},
			},
			{
				BaseResource: models.BaseResource{
					ID:    "fpr-2",
					State: models.StateApproved,
				},
			},
		}

		mockLookup.On("ListFocomProvisioningRequestsByOCloud", ctx, "ocloud-1", "default").Return(dependentFPRs, nil)

		result := validator.ValidateOCloudDeletion(ctx, "ocloud-1", "default")

		assert.False(t, result.Success)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Cannot delete OCloud 'ocloud-1' in namespace 'default': it is referenced by 2 approved provisioning request(s): [fpr-1 fpr-2]")
		mockLookup.AssertExpectations(t)
	})

	t.Run("Can delete OCloud with only draft FPR dependencies", func(t *testing.T) {
		mockLookup := new(MockResourceLookup)
		validator := NewDependencyValidator(mockLookup)

		dependentFPRs := []*models.FocomProvisioningRequestData{
			{
				BaseResource: models.BaseResource{
					ID:    "fpr-1",
					State: models.StateDraft,
				},
			},
			{
				BaseResource: models.BaseResource{
					ID:    "fpr-2",
					State: models.StateValidated,
				},
			},
		}

		mockLookup.On("ListFocomProvisioningRequestsByOCloud", ctx, "ocloud-1", "default").Return(dependentFPRs, nil)

		result := validator.ValidateOCloudDeletion(ctx, "ocloud-1", "default")

		assert.True(t, result.Success)
		assert.Empty(t, result.Errors)
		mockLookup.AssertExpectations(t)
	})

	t.Run("Lookup error", func(t *testing.T) {
		mockLookup := new(MockResourceLookup)
		validator := NewDependencyValidator(mockLookup)

		mockLookup.On("ListFocomProvisioningRequestsByOCloud", ctx, "ocloud-1", "default").Return(nil, errors.New("database error"))

		result := validator.ValidateOCloudDeletion(ctx, "ocloud-1", "default")

		assert.False(t, result.Success)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Failed to check for dependent provisioning requests: database error")
		mockLookup.AssertExpectations(t)
	})
}

func TestDependencyValidator_ValidateTemplateInfoDeletion(t *testing.T) {
	ctx := context.Background()

	t.Run("Can delete TemplateInfo with no dependencies", func(t *testing.T) {
		mockLookup := new(MockResourceLookup)
		validator := NewDependencyValidator(mockLookup)

		mockLookup.On("ListFocomProvisioningRequestsByTemplate", ctx, "test-template", "v1.0.0").Return([]*models.FocomProvisioningRequestData{}, nil)

		result := validator.ValidateTemplateInfoDeletion(ctx, "test-template", "v1.0.0")

		assert.True(t, result.Success)
		assert.Empty(t, result.Errors)
		mockLookup.AssertExpectations(t)
	})

	t.Run("Cannot delete TemplateInfo with approved FPR dependencies", func(t *testing.T) {
		mockLookup := new(MockResourceLookup)
		validator := NewDependencyValidator(mockLookup)

		dependentFPRs := []*models.FocomProvisioningRequestData{
			{
				BaseResource: models.BaseResource{
					ID:    "fpr-1",
					State: models.StateApproved,
				},
			},
		}

		mockLookup.On("ListFocomProvisioningRequestsByTemplate", ctx, "test-template", "v1.0.0").Return(dependentFPRs, nil)

		result := validator.ValidateTemplateInfoDeletion(ctx, "test-template", "v1.0.0")

		assert.False(t, result.Success)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Cannot delete TemplateInfo 'test-template' version 'v1.0.0': it is referenced by 1 approved provisioning request(s): [fpr-1]")
		mockLookup.AssertExpectations(t)
	})
}

func TestDependencyValidator_ValidateResourceCreationOrder(t *testing.T) {
	ctx := context.Background()
	mockLookup := new(MockResourceLookup)
	validator := NewDependencyValidator(mockLookup)

	t.Run("OCloud creation order - valid", func(t *testing.T) {
		result := validator.ValidateResourceCreationOrder(ctx, models.ResourceTypeOCloud, []string{})

		assert.True(t, result.Success)
		assert.Empty(t, result.Errors)
	})

	t.Run("OCloud creation order - with dependencies (warning)", func(t *testing.T) {
		result := validator.ValidateResourceCreationOrder(ctx, models.ResourceTypeOCloud, []string{"dep1"})

		assert.True(t, result.Success)
		assert.Empty(t, result.Errors)
		assert.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0], "OCloud resources should not have dependencies")
	})

	t.Run("TemplateInfo creation order - valid", func(t *testing.T) {
		result := validator.ValidateResourceCreationOrder(ctx, models.ResourceTypeTemplateInfo, []string{})

		assert.True(t, result.Success)
		assert.Empty(t, result.Errors)
	})

	t.Run("FPR creation order - valid", func(t *testing.T) {
		result := validator.ValidateResourceCreationOrder(ctx, models.ResourceTypeFocomProvisioningRequest, []string{"ocloud", "template"})

		assert.True(t, result.Success)
		assert.Empty(t, result.Errors)
	})

	t.Run("FPR creation order - insufficient dependencies", func(t *testing.T) {
		result := validator.ValidateResourceCreationOrder(ctx, models.ResourceTypeFocomProvisioningRequest, []string{"ocloud"})

		assert.False(t, result.Success)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "FocomProvisioningRequest requires both OCloud and TemplateInfo dependencies")
	})

	t.Run("Unknown resource type", func(t *testing.T) {
		result := validator.ValidateResourceCreationOrder(ctx, "UnknownType", []string{})

		assert.False(t, result.Success)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Unknown resource type: UnknownType")
	})
}

func TestDependencyValidator_ValidateCircularDependencies(t *testing.T) {
	ctx := context.Background()
	mockLookup := new(MockResourceLookup)
	validator := NewDependencyValidator(mockLookup)

	t.Run("No circular dependencies", func(t *testing.T) {
		result := validator.ValidateCircularDependencies(ctx, models.ResourceTypeOCloud, "ocloud-1", []string{})

		assert.True(t, result.Success)
		assert.Empty(t, result.Errors)
	})

	t.Run("Simple circular dependency", func(t *testing.T) {
		// This is a contrived example since FOCOM doesn't have circular deps in practice
		result := validator.ValidateCircularDependencies(ctx, models.ResourceTypeOCloud, "ocloud-1", []string{"ocloud-1"})

		assert.False(t, result.Success)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "Circular dependency detected for resource ocloud-1 of type OCloud")
	})
}

func TestDependencyValidator_GetDependencyError(t *testing.T) {
	mockLookup := new(MockResourceLookup)
	validator := NewDependencyValidator(mockLookup)

	depError := validator.GetDependencyError(
		models.ResourceTypeOCloud,
		"ocloud-1",
		models.ResourceTypeFocomProvisioningRequest,
		"fpr-1",
		"Test dependency error",
	)

	assert.Equal(t, models.ResourceTypeOCloud, depError.ResourceType)
	assert.Equal(t, "ocloud-1", depError.ResourceID)
	assert.Equal(t, models.ResourceTypeFocomProvisioningRequest, depError.DependentType)
	assert.Equal(t, "fpr-1", depError.DependentID)
	assert.Equal(t, "Test dependency error", depError.Message)
	assert.Equal(t, "Test dependency error", depError.Error())
}
