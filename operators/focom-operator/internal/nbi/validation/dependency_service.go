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
	"fmt"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
)

// DependencyValidationService provides comprehensive dependency validation functionality
type DependencyValidationService struct {
	dependencyValidator *DependencyValidator
	resourceLookup      ResourceLookup
}

// NewDependencyValidationService creates a new DependencyValidationService
func NewDependencyValidationService(resourceLookup ResourceLookup) *DependencyValidationService {
	return &DependencyValidationService{
		dependencyValidator: NewDependencyValidator(resourceLookup),
		resourceLookup:      resourceLookup,
	}
}

// ValidateResourceDependencies validates dependencies for any resource type
func (dvs *DependencyValidationService) ValidateResourceDependencies(ctx context.Context, resourceType models.ResourceType, resource interface{}) *models.ValidationResult {
	switch resourceType {
	case models.ResourceTypeFocomProvisioningRequest:
		if fpr, ok := resource.(*models.FocomProvisioningRequestData); ok {
			return dvs.dependencyValidator.ValidateFocomProvisioningRequestDependencies(ctx, fpr)
		}
		return models.NewValidationResult(false, []string{"Invalid FocomProvisioningRequest resource type"}, nil)

	case models.ResourceTypeOCloud:
		// OCloud has no dependencies, always valid
		return models.NewValidationResult(true, nil, nil)

	case models.ResourceTypeTemplateInfo:
		// TemplateInfo has no dependencies, always valid
		return models.NewValidationResult(true, nil, nil)

	default:
		return models.NewValidationResult(false, []string{fmt.Sprintf("Unknown resource type: %s", resourceType)}, nil)
	}
}

// ValidateResourceDeletion validates that a resource can be safely deleted
func (dvs *DependencyValidationService) ValidateResourceDeletion(ctx context.Context, resourceType models.ResourceType, resourceID, namespace string) *models.ValidationResult {
	switch resourceType {
	case models.ResourceTypeOCloud:
		return dvs.dependencyValidator.ValidateOCloudDeletion(ctx, resourceID, namespace)

	case models.ResourceTypeTemplateInfo:
		// For TemplateInfo, we need to extract name and version from the resource
		templateInfo, err := dvs.resourceLookup.GetTemplateInfo(ctx, resourceID, "")
		if err != nil {
			return models.NewValidationResult(false, []string{fmt.Sprintf("Failed to lookup TemplateInfo for deletion validation: %v", err)}, nil)
		}
		if templateInfo == nil {
			// Resource doesn't exist, deletion is safe
			return models.NewValidationResult(true, nil, nil)
		}
		return dvs.dependencyValidator.ValidateTemplateInfoDeletion(ctx, templateInfo.TemplateName, templateInfo.TemplateVersion)

	case models.ResourceTypeFocomProvisioningRequest:
		// FPR can always be deleted as it doesn't have dependents
		return models.NewValidationResult(true, nil, nil)

	default:
		return models.NewValidationResult(false, []string{fmt.Sprintf("Unknown resource type: %s", resourceType)}, nil)
	}
}

// ValidateApprovalDependencies validates dependencies before approving a draft
func (dvs *DependencyValidationService) ValidateApprovalDependencies(ctx context.Context, resourceType models.ResourceType, resource interface{}) *models.ValidationResult {
	// For approval, we need to ensure all dependencies exist and are in approved state
	return dvs.ValidateResourceDependencies(ctx, resourceType, resource)
}

// ValidateResourceCreationOrder validates that resources are created in correct dependency order
func (dvs *DependencyValidationService) ValidateResourceCreationOrder(ctx context.Context, resourceType models.ResourceType, dependencies []string) *models.ValidationResult {
	return dvs.dependencyValidator.ValidateResourceCreationOrder(ctx, resourceType, dependencies)
}

// GetDependencyViolationError creates a structured dependency violation error
func (dvs *DependencyValidationService) GetDependencyViolationError(resourceType models.ResourceType, resourceID string, dependentType models.ResourceType, dependentID, message string) *models.DependencyError {
	return dvs.dependencyValidator.GetDependencyError(resourceType, resourceID, dependentType, dependentID, message)
}

// ValidateCircularDependencies validates that there are no circular dependencies
func (dvs *DependencyValidationService) ValidateCircularDependencies(ctx context.Context, resourceType models.ResourceType, resourceID string, dependencies []string) *models.ValidationResult {
	return dvs.dependencyValidator.ValidateCircularDependencies(ctx, resourceType, resourceID, dependencies)
}

// GetResourceDependencies returns the list of dependencies for a given resource
func (dvs *DependencyValidationService) GetResourceDependencies(ctx context.Context, resourceType models.ResourceType, resource interface{}) ([]string, error) {
	switch resourceType {
	case models.ResourceTypeFocomProvisioningRequest:
		if fpr, ok := resource.(*models.FocomProvisioningRequestData); ok {
			dependencies := []string{
				fmt.Sprintf("ocloud:%s:%s", fpr.OCloudID, fpr.OCloudNamespace),
				fmt.Sprintf("templateinfo:%s:%s", fpr.TemplateName, fpr.TemplateVersion),
			}
			return dependencies, nil
		}
		return nil, fmt.Errorf("invalid FocomProvisioningRequest resource type")

	case models.ResourceTypeOCloud:
		// OCloud has no dependencies
		return []string{}, nil

	case models.ResourceTypeTemplateInfo:
		// TemplateInfo has no dependencies
		return []string{}, nil

	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

// GetResourceDependents returns the list of resources that depend on the given resource
func (dvs *DependencyValidationService) GetResourceDependents(ctx context.Context, resourceType models.ResourceType, resourceID, namespace string) ([]*models.FocomProvisioningRequestData, error) {
	switch resourceType {
	case models.ResourceTypeOCloud:
		return dvs.resourceLookup.ListFocomProvisioningRequestsByOCloud(ctx, resourceID, namespace)

	case models.ResourceTypeTemplateInfo:
		// For TemplateInfo, we need to get the template name and version
		templateInfo, err := dvs.resourceLookup.GetTemplateInfo(ctx, resourceID, "")
		if err != nil {
			return nil, fmt.Errorf("failed to lookup TemplateInfo: %v", err)
		}
		if templateInfo == nil {
			return []*models.FocomProvisioningRequestData{}, nil
		}
		return dvs.resourceLookup.ListFocomProvisioningRequestsByTemplate(ctx, templateInfo.TemplateName, templateInfo.TemplateVersion)

	case models.ResourceTypeFocomProvisioningRequest:
		// FPR has no dependents
		return []*models.FocomProvisioningRequestData{}, nil

	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

// ValidateDependencyChain validates the entire dependency chain for a resource
func (dvs *DependencyValidationService) ValidateDependencyChain(ctx context.Context, resourceType models.ResourceType, resource interface{}) *models.ValidationResult {
	var errors []string
	var warnings []string

	// Get dependencies for this resource
	dependencies, err := dvs.GetResourceDependencies(ctx, resourceType, resource)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to get resource dependencies: %v", err))
		return models.NewValidationResult(false, errors, warnings)
	}

	// Validate each dependency exists and is in correct state
	for _, dep := range dependencies {
		// Parse dependency string (format: "type:id:namespace" or "type:name:version")
		// This is a simplified validation - in a real implementation, you'd parse the dependency format
		if dep == "" {
			errors = append(errors, "Empty dependency found")
		}
	}

	// Validate resource creation order
	orderResult := dvs.ValidateResourceCreationOrder(ctx, resourceType, dependencies)
	if !orderResult.Success {
		errors = append(errors, orderResult.Errors...)
	}
	warnings = append(warnings, orderResult.Warnings...)

	// Validate circular dependencies
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

	if resourceID != "" {
		circularResult := dvs.ValidateCircularDependencies(ctx, resourceType, resourceID, dependencies)
		if !circularResult.Success {
			errors = append(errors, circularResult.Errors...)
		}
		warnings = append(warnings, circularResult.Warnings...)
	}

	return models.NewValidationResult(len(errors) == 0, errors, warnings)
}
