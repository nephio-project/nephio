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
	"fmt"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
)

// IntegrationExample demonstrates how to use the dependency validation system
// This file serves as documentation and example code for integrating dependency validation
// into the broader NBI system components (handlers, services, storage)

// ExampleResourceService demonstrates how a service would integrate dependency validation
type ExampleResourceService struct {
	dependencyService *DependencyValidationService
	errorBuilder      *DependencyErrorBuilder
}

// NewExampleResourceService creates a new example service with dependency validation
func NewExampleResourceService(resourceLookup ResourceLookup) *ExampleResourceService {
	return &ExampleResourceService{
		dependencyService: NewDependencyValidationService(resourceLookup),
		errorBuilder:      NewDependencyErrorBuilder(),
	}
}

// ApproveResource demonstrates how to validate dependencies before approving a resource
func (ers *ExampleResourceService) ApproveResource(ctx context.Context, resourceType models.ResourceType, resource interface{}) error {
	// Step 1: Validate dependencies
	validationResult := ers.dependencyService.ValidateApprovalDependencies(ctx, resourceType, resource)

	if !validationResult.Success {
		// Create detailed error response
		var resourceID string
		switch resourceType {
		case models.ResourceTypeFocomProvisioningRequest:
			if fpr, ok := resource.(*models.FocomProvisioningRequestData); ok {
				resourceID = fpr.ID
			}
		case models.ResourceTypeOCloud:
			if ocloud, ok := resource.(*models.OCloudData); ok {
				resourceID = ocloud.ID
			}
		case models.ResourceTypeTemplateInfo:
			if templateInfo, ok := resource.(*models.TemplateInfoData); ok {
				resourceID = templateInfo.ID
			}
		}

		// Build user-friendly error message
		summary := ers.errorBuilder.BuildDependencyValidationSummary(resourceType, resourceID, validationResult)
		recommendations := ers.errorBuilder.BuildRecommendationMessage(resourceType, validationResult)

		// Create detailed error report
		report := ers.errorBuilder.BuildDetailedValidationReport(resourceType, resourceID, validationResult)

		// Log the detailed report for debugging
		fmt.Printf("Dependency validation failed: %+v\n", report)

		// Return user-friendly error with recommendations
		errorMsg := fmt.Sprintf("%s. Recommendations: %v", summary, recommendations)
		return errors.New(errorMsg)
	}

	// Step 2: If validation passes, proceed with approval
	// (This would integrate with the actual approval logic)
	fmt.Printf("Dependencies validated successfully for %s %s\n", resourceType, "resource-id")

	return nil
}

// DeleteResource demonstrates how to validate dependencies before deleting a resource
func (ers *ExampleResourceService) DeleteResource(ctx context.Context, resourceType models.ResourceType, resourceID, namespace string) error {
	// Step 1: Validate that deletion is safe (no approved dependents)
	validationResult := ers.dependencyService.ValidateResourceDeletion(ctx, resourceType, resourceID, namespace)

	if !validationResult.Success {
		// Get list of dependents for detailed error message
		dependents, err := ers.dependencyService.GetResourceDependents(ctx, resourceType, resourceID, namespace)
		if err != nil {
			return fmt.Errorf("failed to validate resource deletion: %v", err)
		}

		// Build detailed deletion blocked error
		var dependentIDs []string
		for _, dep := range dependents {
			if dep.State == models.StateApproved {
				dependentIDs = append(dependentIDs, dep.ID)
			}
		}

		if len(dependentIDs) > 0 {
			errorMsg := ers.errorBuilder.BuildDeletionBlockedError(resourceType, resourceID, namespace, len(dependentIDs), dependentIDs)
			return errors.New(errorMsg)
		}

		// Generic validation failure
		summary := ers.errorBuilder.BuildDependencyValidationSummary(resourceType, resourceID, validationResult)
		return fmt.Errorf("deletion validation failed: %s", summary)
	}

	// Step 2: If validation passes, proceed with deletion
	// (This would integrate with the actual deletion logic)
	fmt.Printf("Deletion validation passed for %s %s\n", resourceType, resourceID)

	return nil
}

// ValidateResourceCreation demonstrates how to validate resource creation order
func (ers *ExampleResourceService) ValidateResourceCreation(ctx context.Context, resourceType models.ResourceType, resource interface{}) error {
	// Step 1: Get resource dependencies
	dependencies, err := ers.dependencyService.GetResourceDependencies(ctx, resourceType, resource)
	if err != nil {
		return fmt.Errorf("failed to get resource dependencies: %v", err)
	}

	// Step 2: Validate creation order
	orderResult := ers.dependencyService.ValidateResourceCreationOrder(ctx, resourceType, dependencies)
	if !orderResult.Success {
		errorMsg := ers.errorBuilder.BuildInvalidCreationOrderError(resourceType, dependencies)
		return errors.New(errorMsg)
	}

	// Step 3: Validate dependency chain
	chainResult := ers.dependencyService.ValidateDependencyChain(ctx, resourceType, resource)
	if !chainResult.Success {
		chainErrors := chainResult.Errors
		errorMsg := ers.errorBuilder.BuildDependencyChainError(resourceType, "resource-id", chainErrors)
		return errors.New(errorMsg)
	}

	fmt.Printf("Resource creation validation passed for %s\n", resourceType)
	return nil
}

// ExampleHTTPHandler demonstrates how an HTTP handler would use dependency validation
type ExampleHTTPHandler struct {
	resourceService *ExampleResourceService
	errorBuilder    *DependencyErrorBuilder
}

// NewExampleHTTPHandler creates a new example HTTP handler
func NewExampleHTTPHandler(resourceLookup ResourceLookup) *ExampleHTTPHandler {
	return &ExampleHTTPHandler{
		resourceService: NewExampleResourceService(resourceLookup),
		errorBuilder:    NewDependencyErrorBuilder(),
	}
}

// HandleApproveRequest demonstrates how an HTTP handler would handle approval requests
func (ehh *ExampleHTTPHandler) HandleApproveRequest(ctx context.Context, resourceType models.ResourceType, resourceID string, resource interface{}) (int, map[string]interface{}) {
	// Validate dependencies before approval
	err := ehh.resourceService.ApproveResource(ctx, resourceType, resource)
	if err != nil {
		// Return 400 Bad Request with detailed error information
		errorResponse := map[string]interface{}{
			"error":        "Dependency validation failed",
			"code":         models.ErrorCodeDependency,
			"details":      err.Error(),
			"resourceType": ehh.errorBuilder.GetUserFriendlyResourceTypeName(resourceType),
			"resourceId":   resourceID,
		}
		return 400, errorResponse
	}

	// Return success response
	successResponse := map[string]interface{}{
		"message":      "Resource approved successfully",
		"resourceType": ehh.errorBuilder.GetUserFriendlyResourceTypeName(resourceType),
		"resourceId":   resourceID,
	}
	return 200, successResponse
}

// HandleDeleteRequest demonstrates how an HTTP handler would handle deletion requests
func (ehh *ExampleHTTPHandler) HandleDeleteRequest(ctx context.Context, resourceType models.ResourceType, resourceID, namespace string) (int, map[string]interface{}) {
	// Validate dependencies before deletion
	err := ehh.resourceService.DeleteResource(ctx, resourceType, resourceID, namespace)
	if err != nil {
		// Return 409 Conflict for dependency violations
		errorResponse := map[string]interface{}{
			"error":        "Cannot delete resource due to dependencies",
			"code":         models.ErrorCodeDependency,
			"details":      err.Error(),
			"resourceType": ehh.errorBuilder.GetUserFriendlyResourceTypeName(resourceType),
			"resourceId":   resourceID,
		}
		return 409, errorResponse
	}

	// Return success response
	successResponse := map[string]interface{}{
		"message":      "Resource deletion initiated",
		"resourceType": ehh.errorBuilder.GetUserFriendlyResourceTypeName(resourceType),
		"resourceId":   resourceID,
	}
	return 202, successResponse
}

// ExampleStorageIntegration demonstrates how storage layer would integrate dependency validation
type ExampleStorageIntegration struct {
	dependencyService *DependencyValidationService
	errorBuilder      *DependencyErrorBuilder
}

// NewExampleStorageIntegration creates a new example storage integration
func NewExampleStorageIntegration(resourceLookup ResourceLookup) *ExampleStorageIntegration {
	return &ExampleStorageIntegration{
		dependencyService: NewDependencyValidationService(resourceLookup),
		errorBuilder:      NewDependencyErrorBuilder(),
	}
}

// ValidateDependencies implements the storage interface ValidateDependencies method
func (esi *ExampleStorageIntegration) ValidateDependencies(ctx context.Context, resourceType models.ResourceType, resource interface{}) error {
	// This method would be called by the storage layer before persisting resources
	validationResult := esi.dependencyService.ValidateResourceDependencies(ctx, resourceType, resource)

	if !validationResult.Success {
		// Create storage-specific error
		var resourceID string
		switch resourceType {
		case models.ResourceTypeFocomProvisioningRequest:
			if fpr, ok := resource.(*models.FocomProvisioningRequestData); ok {
				resourceID = fpr.ID
			}
		}

		summary := esi.errorBuilder.BuildDependencyValidationSummary(resourceType, resourceID, validationResult)
		return fmt.Errorf("storage dependency validation failed: %s", summary)
	}

	return nil
}

// Usage Examples and Best Practices:

// Example 1: Validating FPR approval
func ExampleValidateFPRApproval(ctx context.Context, resourceLookup ResourceLookup) {
	service := NewExampleResourceService(resourceLookup)

	fpr := &models.FocomProvisioningRequestData{
		BaseResource: models.BaseResource{
			ID:    "fpr-1",
			State: models.StateValidated,
		},
		OCloudID:        "ocloud-1",
		OCloudNamespace: "focom-system",
		TemplateName:    "template-1",
		TemplateVersion: "v1.0.0",
	}

	err := service.ApproveResource(ctx, models.ResourceTypeFocomProvisioningRequest, fpr)
	if err != nil {
		fmt.Printf("Approval failed: %v\n", err)
		return
	}

	fmt.Println("FPR approved successfully")
}

// Example 2: Validating OCloud deletion
func ExampleValidateOCloudDeletion(ctx context.Context, resourceLookup ResourceLookup) {
	service := NewExampleResourceService(resourceLookup)

	err := service.DeleteResource(ctx, models.ResourceTypeOCloud, "ocloud-1", "focom-system")
	if err != nil {
		fmt.Printf("Deletion blocked: %v\n", err)
		return
	}

	fmt.Println("OCloud deletion allowed")
}

// Example 3: HTTP handler integration
func ExampleHTTPHandlerIntegration(ctx context.Context, resourceLookup ResourceLookup) {
	handler := NewExampleHTTPHandler(resourceLookup)

	fpr := &models.FocomProvisioningRequestData{
		BaseResource: models.BaseResource{ID: "fpr-1"},
		OCloudID:     "ocloud-1",
		TemplateName: "template-1",
	}

	statusCode, response := handler.HandleApproveRequest(ctx, models.ResourceTypeFocomProvisioningRequest, "fpr-1", fpr)
	fmt.Printf("HTTP Response: %d, Body: %+v\n", statusCode, response)
}

// Integration Guidelines:
//
// 1. Service Layer Integration:
//    - Call dependency validation before any state-changing operations
//    - Use error builder to create user-friendly error messages
//    - Include recommendations in error responses
//
// 2. HTTP Handler Integration:
//    - Return appropriate HTTP status codes (400 for validation, 409 for conflicts)
//    - Include detailed error information in response body
//    - Use user-friendly resource type names
//
// 3. Storage Layer Integration:
//    - Validate dependencies before persisting resources
//    - Use storage-specific error codes and messages
//    - Ensure validation is atomic with storage operations
//
// 4. Error Handling Best Practices:
//    - Always include resource type and ID in error messages
//    - Provide actionable recommendations for fixing issues
//    - Log detailed validation reports for debugging
//    - Use structured error responses for programmatic handling
//
// 5. Performance Considerations:
//    - Cache dependency lookup results when possible
//    - Validate dependencies only when necessary (e.g., before approval/deletion)
//    - Use batch validation for multiple resources when applicable
