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
	"testing"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// FPRMockValidator is a mock validator that properly delegates ValidateTemplateParameters
type FPRMockValidator struct {
	mock.Mock
}

func (m *FPRMockValidator) ValidateOCloud(ctx context.Context, ocloud *models.OCloudData) *models.ValidationResult {
	args := m.Called(ctx, ocloud)
	return args.Get(0).(*models.ValidationResult)
}

func (m *FPRMockValidator) ValidateTemplateInfo(ctx context.Context, templateInfo *models.TemplateInfoData) *models.ValidationResult {
	args := m.Called(ctx, templateInfo)
	return args.Get(0).(*models.ValidationResult)
}

func (m *FPRMockValidator) ValidateFocomProvisioningRequest(ctx context.Context, fpr *models.FocomProvisioningRequestData) *models.ValidationResult {
	args := m.Called(ctx, fpr)
	return args.Get(0).(*models.ValidationResult)
}

func (m *FPRMockValidator) ValidateTemplateParameters(ctx context.Context, parameters map[string]interface{}, schema string) *models.ValidationResult {
	args := m.Called(ctx, parameters, schema)
	return args.Get(0).(*models.ValidationResult)
}

func (m *FPRMockValidator) ValidateJSON(jsonStr string) error {
	args := m.Called(jsonStr)
	return args.Error(0)
}

func (m *FPRMockValidator) ValidateYAML(yamlStr string) error {
	args := m.Called(yamlStr)
	return args.Error(0)
}

// helper to create a valid FPR draft for testing
func newTestFPR(id, templateName, templateVersion string, params map[string]interface{}) *models.FocomProvisioningRequestData {
	return &models.FocomProvisioningRequestData{
		BaseResource: models.BaseResource{
			ID:          id,
			Namespace:   "default",
			Name:        "test-fpr",
			Description: "Test FPR",
			State:       models.StateDraft,
		},
		OCloudID:           "ocloud-1",
		OCloudNamespace:    "default",
		TemplateName:       templateName,
		TemplateVersion:    templateVersion,
		TemplateParameters: params,
	}
}

// helper to create an approved TemplateInfo with a given schema
func newTestTemplateInfo(name, version, schema string, state models.ResourceState) *models.TemplateInfoData {
	return &models.TemplateInfoData{
		BaseResource: models.BaseResource{
			ID:    models.SanitizeID(name + "-" + version),
			Name:  name,
			State: state,
		},
		TemplateName:            name,
		TemplateVersion:         version,
		TemplateParameterSchema: schema,
	}
}

func TestFPRService_ValidateDraft_ValidParameters(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageInterface)
	mockValidator := new(FPRMockValidator)
	mockIntegration := new(MockOperatorIntegration)

	service := NewFocomProvisioningRequestService(mockStorage, mockValidator, mockIntegration)

	fpr := newTestFPR("fpr-1", "cluster-template", "v1.0", map[string]interface{}{
		"nodeCount":   float64(3),
		"clusterName": "my-cluster",
	})

	schema := `{"type":"object","properties":{"nodeCount":{"type":"number"},"clusterName":{"type":"string"}},"required":["nodeCount","clusterName"]}`
	ti := newTestTemplateInfo("cluster-template", "v1.0", schema, models.StateApproved)

	// GetDraft returns the FPR
	mockStorage.On("GetDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-1").Return(fpr, nil)
	// Basic validation passes
	mockValidator.On("ValidateFocomProvisioningRequest", ctx, fpr).Return(&models.ValidationResult{Success: true, Errors: []string{}})
	// List TemplateInfos returns our template
	mockStorage.On("List", ctx, storage.ResourceTypeTemplateInfo).Return([]interface{}{ti}, nil)
	// Schema validation passes
	mockValidator.On("ValidateTemplateParameters", ctx, fpr.TemplateParameters, schema).Return(&models.ValidationResult{
		Success: true, Errors: []string{}, Warnings: []string{},
	})
	// State update to VALIDATED succeeds
	mockStorage.On("ValidateDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-1").Return(nil)

	result, err := service.ValidateDraft(ctx, "fpr-1")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Empty(t, result.Errors)
	mockStorage.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
}

func TestFPRService_ValidateDraft_InvalidParameters(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageInterface)
	mockValidator := new(FPRMockValidator)
	mockIntegration := new(MockOperatorIntegration)

	service := NewFocomProvisioningRequestService(mockStorage, mockValidator, mockIntegration)

	fpr := newTestFPR("fpr-2", "cluster-template", "v1.0", map[string]interface{}{
		"nodeCount": "not-a-number", // wrong type
	})

	schema := `{"type":"object","properties":{"nodeCount":{"type":"number"},"clusterName":{"type":"string"}},"required":["nodeCount","clusterName"]}`
	ti := newTestTemplateInfo("cluster-template", "v1.0", schema, models.StateApproved)

	mockStorage.On("GetDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-2").Return(fpr, nil)
	mockValidator.On("ValidateFocomProvisioningRequest", ctx, fpr).Return(&models.ValidationResult{Success: true, Errors: []string{}})
	mockStorage.On("List", ctx, storage.ResourceTypeTemplateInfo).Return([]interface{}{ti}, nil)

	schemaErrors := []models.SchemaValidationError{
		{Field: "nodeCount", Description: "Invalid type. Expected: number, given: string", Constraint: "type"},
		{Field: "(root)", Description: "clusterName is required", Constraint: "required"},
	}
	mockValidator.On("ValidateTemplateParameters", ctx, fpr.TemplateParameters, schema).Return(&models.ValidationResult{
		Success: false,
		Errors: []string{
			"Template parameter 'nodeCount' violates constraint 'type': Invalid type. Expected: number, given: string",
			"Template parameter '(root)' violates constraint 'required': clusterName is required",
		},
		SchemaErrors: schemaErrors,
	})

	result, err := service.ValidateDraft(ctx, "fpr-2")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Len(t, result.SchemaErrors, 2)
	assert.Equal(t, "nodeCount", result.SchemaErrors[0].Field)
	assert.Equal(t, "type", result.SchemaErrors[0].Constraint)
	assert.Contains(t, result.Errors[0], "nodeCount")
	mockStorage.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
}

func TestFPRService_ValidateDraft_MissingTemplateInfo(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageInterface)
	mockValidator := new(FPRMockValidator)
	mockIntegration := new(MockOperatorIntegration)

	service := NewFocomProvisioningRequestService(mockStorage, mockValidator, mockIntegration)

	fpr := newTestFPR("fpr-3", "nonexistent-template", "v1.0", map[string]interface{}{"key": "value"})

	mockStorage.On("GetDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-3").Return(fpr, nil)
	mockValidator.On("ValidateFocomProvisioningRequest", ctx, fpr).Return(&models.ValidationResult{Success: true, Errors: []string{}})
	// Return empty list — no matching TemplateInfo
	mockStorage.On("List", ctx, storage.ResourceTypeTemplateInfo).Return([]interface{}{}, nil)

	result, err := service.ValidateDraft(ctx, "fpr-3")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "nonexistent-template")
	assert.Contains(t, result.Errors[0], "v1.0")
	mockStorage.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
}

func TestFPRService_ValidateDraft_TemplateInfoNotApproved(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageInterface)
	mockValidator := new(FPRMockValidator)
	mockIntegration := new(MockOperatorIntegration)

	service := NewFocomProvisioningRequestService(mockStorage, mockValidator, mockIntegration)

	fpr := newTestFPR("fpr-4", "cluster-template", "v1.0", map[string]interface{}{"key": "value"})

	schema := `{"type":"object"}`
	ti := newTestTemplateInfo("cluster-template", "v1.0", schema, models.StateDraft) // NOT approved

	mockStorage.On("GetDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-4").Return(fpr, nil)
	mockValidator.On("ValidateFocomProvisioningRequest", ctx, fpr).Return(&models.ValidationResult{Success: true, Errors: []string{}})
	mockStorage.On("List", ctx, storage.ResourceTypeTemplateInfo).Return([]interface{}{ti}, nil)

	result, err := service.ValidateDraft(ctx, "fpr-4")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "not in APPROVED state")
	assert.Contains(t, result.Errors[0], "cluster-template")
	assert.Contains(t, result.Errors[0], "v1.0")
	mockStorage.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
}
