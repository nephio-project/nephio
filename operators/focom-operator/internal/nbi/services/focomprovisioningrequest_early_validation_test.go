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

var testSchema = `{"type":"object","properties":{"nodeCount":{"type":"number","minimum":1},"clusterName":{"type":"string"}},"required":["nodeCount","clusterName"]}`

func TestFPRService_CreateDraft_EarlyValidation_Enabled_ValidParams(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageInterface)
	mockValidator := new(FPRMockValidator)
	mockIntegration := new(MockOperatorIntegration)

	service := NewFocomProvisioningRequestService(mockStorage, mockValidator, mockIntegration)
	service.SetEarlySchemaValidation(true)

	fpr := newTestFPR("fpr-ev-1", "cluster-template", "v1.0", map[string]interface{}{
		"nodeCount":   float64(3),
		"clusterName": "my-cluster",
	})

	ti := newTestTemplateInfo("cluster-template", "v1.0", testSchema, models.StateApproved)

	// TemplateInfo lookup
	mockStorage.On("List", ctx, storage.ResourceTypeTemplateInfo).Return([]interface{}{ti}, nil)
	// Schema validation passes
	mockValidator.On("ValidateTemplateParameters", ctx, fpr.TemplateParameters, testSchema).Return(&models.ValidationResult{
		Success: true, Errors: []string{},
	})
	// Storage create succeeds
	mockStorage.On("CreateDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, mock.AnythingOfType("*models.FocomProvisioningRequestData")).Return(nil)

	result, err := service.CreateDraft(ctx, fpr)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockStorage.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
}

func TestFPRService_CreateDraft_EarlyValidation_Enabled_InvalidParams(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageInterface)
	mockValidator := new(FPRMockValidator)
	mockIntegration := new(MockOperatorIntegration)

	service := NewFocomProvisioningRequestService(mockStorage, mockValidator, mockIntegration)
	service.SetEarlySchemaValidation(true)

	fpr := newTestFPR("fpr-ev-2", "cluster-template", "v1.0", map[string]interface{}{
		"nodeCount": "not-a-number",
	})

	ti := newTestTemplateInfo("cluster-template", "v1.0", testSchema, models.StateApproved)

	mockStorage.On("List", ctx, storage.ResourceTypeTemplateInfo).Return([]interface{}{ti}, nil)
	mockValidator.On("ValidateTemplateParameters", ctx, fpr.TemplateParameters, testSchema).Return(&models.ValidationResult{
		Success: false,
		Errors:  []string{"Template parameter 'nodeCount' violates constraint 'type': Invalid type"},
		SchemaErrors: []models.SchemaValidationError{
			{Field: "nodeCount", Description: "Invalid type", Constraint: "type"},
		},
	})

	result, err := service.CreateDraft(ctx, fpr)

	assert.Error(t, err)
	assert.Nil(t, result)
	// Verify it's an EarlyValidationError with schema details
	var earlyErr *EarlyValidationError
	assert.ErrorAs(t, err, &earlyErr)
	assert.Len(t, earlyErr.SchemaErrors, 1)
	assert.Equal(t, "nodeCount", earlyErr.SchemaErrors[0].Field)
	// Storage CreateDraft should NOT have been called
	mockStorage.AssertNotCalled(t, "CreateDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, mock.Anything)
}

func TestFPRService_CreateDraft_EarlyValidation_Enabled_MissingTemplateInfo(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageInterface)
	mockValidator := new(FPRMockValidator)
	mockIntegration := new(MockOperatorIntegration)

	service := NewFocomProvisioningRequestService(mockStorage, mockValidator, mockIntegration)
	service.SetEarlySchemaValidation(true)

	fpr := newTestFPR("fpr-ev-3", "nonexistent-template", "v1.0", map[string]interface{}{
		"nodeCount":   float64(1),
		"clusterName": "test",
	})

	// Return empty list — no matching TemplateInfo
	mockStorage.On("List", ctx, storage.ResourceTypeTemplateInfo).Return([]interface{}{}, nil)

	result, err := service.CreateDraft(ctx, fpr)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "nonexistent-template")
	assert.Contains(t, err.Error(), "v1.0")
	mockStorage.AssertNotCalled(t, "CreateDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, mock.Anything)
}

func TestFPRService_CreateDraft_EarlyValidation_Disabled_InvalidParams(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageInterface)
	mockValidator := new(FPRMockValidator)
	mockIntegration := new(MockOperatorIntegration)

	service := NewFocomProvisioningRequestService(mockStorage, mockValidator, mockIntegration)
	// Early validation disabled (default)

	fpr := newTestFPR("fpr-ev-4", "cluster-template", "v1.0", map[string]interface{}{
		"nodeCount": "not-a-number",
	})

	// Storage create succeeds — no validation happens
	mockStorage.On("CreateDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, mock.AnythingOfType("*models.FocomProvisioningRequestData")).Return(nil)

	result, err := service.CreateDraft(ctx, fpr)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// ValidateTemplateParameters should NOT have been called
	mockValidator.AssertNotCalled(t, "ValidateTemplateParameters", mock.Anything, mock.Anything, mock.Anything)
	mockStorage.AssertExpectations(t)
}

func TestFPRService_UpdateDraft_EarlyValidation_Enabled_ValidParams(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageInterface)
	mockValidator := new(FPRMockValidator)
	mockIntegration := new(MockOperatorIntegration)

	service := NewFocomProvisioningRequestService(mockStorage, mockValidator, mockIntegration)
	service.SetEarlySchemaValidation(true)

	existing := newTestFPR("fpr-ev-5", "cluster-template", "v1.0", map[string]interface{}{
		"nodeCount":   float64(1),
		"clusterName": "old-cluster",
	})

	updates := &models.FocomProvisioningRequestData{
		TemplateParameters: map[string]interface{}{
			"nodeCount":   float64(5),
			"clusterName": "new-cluster",
		},
	}

	ti := newTestTemplateInfo("cluster-template", "v1.0", testSchema, models.StateApproved)

	mockStorage.On("GetDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-ev-5").Return(existing, nil)
	mockStorage.On("List", ctx, storage.ResourceTypeTemplateInfo).Return([]interface{}{ti}, nil)
	mockValidator.On("ValidateTemplateParameters", ctx, mock.AnythingOfType("map[string]interface {}"), testSchema).Return(&models.ValidationResult{
		Success: true, Errors: []string{},
	})
	mockStorage.On("UpdateDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-ev-5", mock.AnythingOfType("*models.FocomProvisioningRequestData")).Return(nil)

	result, err := service.UpdateDraft(ctx, "fpr-ev-5", updates)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(5), result.TemplateParameters["nodeCount"])
	mockStorage.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
}

func TestFPRService_UpdateDraft_EarlyValidation_Enabled_InvalidParams(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageInterface)
	mockValidator := new(FPRMockValidator)
	mockIntegration := new(MockOperatorIntegration)

	service := NewFocomProvisioningRequestService(mockStorage, mockValidator, mockIntegration)
	service.SetEarlySchemaValidation(true)

	existing := newTestFPR("fpr-ev-6", "cluster-template", "v1.0", map[string]interface{}{
		"nodeCount":   float64(1),
		"clusterName": "old-cluster",
	})

	updates := &models.FocomProvisioningRequestData{
		TemplateParameters: map[string]interface{}{
			"nodeCount": "bad-value",
		},
	}

	ti := newTestTemplateInfo("cluster-template", "v1.0", testSchema, models.StateApproved)

	mockStorage.On("GetDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-ev-6").Return(existing, nil)
	mockStorage.On("List", ctx, storage.ResourceTypeTemplateInfo).Return([]interface{}{ti}, nil)
	mockValidator.On("ValidateTemplateParameters", ctx, mock.AnythingOfType("map[string]interface {}"), testSchema).Return(&models.ValidationResult{
		Success: false,
		Errors:  []string{"Template parameter 'nodeCount' violates constraint 'type': Invalid type"},
		SchemaErrors: []models.SchemaValidationError{
			{Field: "nodeCount", Description: "Invalid type", Constraint: "type"},
		},
	})

	result, err := service.UpdateDraft(ctx, "fpr-ev-6", updates)

	assert.Error(t, err)
	assert.Nil(t, result)
	var earlyErr *EarlyValidationError
	assert.ErrorAs(t, err, &earlyErr)
	assert.Len(t, earlyErr.SchemaErrors, 1)
	// Storage UpdateDraft should NOT have been called
	mockStorage.AssertNotCalled(t, "UpdateDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-ev-6", mock.Anything)
}

func TestFPRService_UpdateDraft_EarlyValidation_Enabled_MissingTemplateInfo(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageInterface)
	mockValidator := new(FPRMockValidator)
	mockIntegration := new(MockOperatorIntegration)

	service := NewFocomProvisioningRequestService(mockStorage, mockValidator, mockIntegration)
	service.SetEarlySchemaValidation(true)

	existing := newTestFPR("fpr-ev-7", "missing-template", "v2.0", map[string]interface{}{
		"nodeCount":   float64(1),
		"clusterName": "test",
	})

	updates := &models.FocomProvisioningRequestData{
		TemplateParameters: map[string]interface{}{
			"nodeCount": float64(2),
		},
	}

	mockStorage.On("GetDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-ev-7").Return(existing, nil)
	mockStorage.On("List", ctx, storage.ResourceTypeTemplateInfo).Return([]interface{}{}, nil)

	result, err := service.UpdateDraft(ctx, "fpr-ev-7", updates)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "missing-template")
	assert.Contains(t, err.Error(), "v2.0")
	mockStorage.AssertNotCalled(t, "UpdateDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-ev-7", mock.Anything)
}

func TestFPRService_UpdateDraft_EarlyValidation_Disabled_InvalidParams(t *testing.T) {
	ctx := context.Background()
	mockStorage := new(MockStorageInterface)
	mockValidator := new(FPRMockValidator)
	mockIntegration := new(MockOperatorIntegration)

	service := NewFocomProvisioningRequestService(mockStorage, mockValidator, mockIntegration)
	// Early validation disabled (default)

	existing := newTestFPR("fpr-ev-8", "cluster-template", "v1.0", map[string]interface{}{
		"nodeCount":   float64(1),
		"clusterName": "old-cluster",
	})

	updates := &models.FocomProvisioningRequestData{
		TemplateParameters: map[string]interface{}{
			"nodeCount": "bad-value",
		},
	}

	mockStorage.On("GetDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-ev-8").Return(existing, nil)
	mockStorage.On("UpdateDraft", ctx, storage.ResourceTypeFocomProvisioningRequest, "fpr-ev-8", mock.AnythingOfType("*models.FocomProvisioningRequestData")).Return(nil)

	result, err := service.UpdateDraft(ctx, "fpr-ev-8", updates)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockValidator.AssertNotCalled(t, "ValidateTemplateParameters", mock.Anything, mock.Anything, mock.Anything)
	mockStorage.AssertExpectations(t)
}
