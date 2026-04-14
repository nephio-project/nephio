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

// ResourceLookup defines the interface for looking up resources during dependency validation
type ResourceLookup interface {
	// GetOCloud retrieves an OCloud by ID and namespace
	GetOCloud(ctx context.Context, id, namespace string) (*models.OCloudData, error)

	// GetTemplateInfo retrieves a TemplateInfo by name and version
	GetTemplateInfo(ctx context.Context, name, version string) (*models.TemplateInfoData, error)

	// ListFocomProvisioningRequestsByOCloud lists all FPRs that reference a specific OCloud
	ListFocomProvisioningRequestsByOCloud(ctx context.Context, oCloudID, oCloudNamespace string) ([]*models.FocomProvisioningRequestData, error)

	// ListFocomProvisioningRequestsByTemplate lists all FPRs that reference a specific template
	ListFocomProvisioningRequestsByTemplate(ctx context.Context, templateName, templateVersion string) ([]*models.FocomProvisioningRequestData, error)
}

// DependencyValidator handles dependency validation between resources
type DependencyValidator struct {
	resourceLookup ResourceLookup
}

// NewDependencyValidator creates a new DependencyValidator
func NewDependencyValidator(resourceLookup ResourceLookup) *DependencyValidator {
	return &DependencyValidator{
		resourceLookup: resourceLookup,
	}
}

// ValidateFocomProvisioningRequestDependencies validates that all dependencies exist for a FPR
func (dv *DependencyValidator) ValidateFocomProvisioningRequestDependencies(ctx context.Context, fpr *models.FocomProvisioningRequestData) *models.ValidationResult {
	var errors []string
	var warnings []string

	// Validate OCloud dependency
	ocloud, err := dv.resourceLookup.GetOCloud(ctx, fpr.OCloudID, fpr.OCloudNamespace)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to lookup OCloud '%s' in namespace '%s': %v", fpr.OCloudID, fpr.OCloudNamespace, err))
	} else if ocloud == nil {
		errors = append(errors, fmt.Sprintf("Referenced OCloud '%s' in namespace '%s' does not exist", fpr.OCloudID, fpr.OCloudNamespace))
	} else if ocloud.State != models.StateApproved {
		errors = append(errors, fmt.Sprintf("Referenced OCloud '%s' in namespace '%s' is not in approved state (current state: %s)", fpr.OCloudID, fpr.OCloudNamespace, ocloud.State))
	}

	// Validate TemplateInfo dependency
	templateInfo, err := dv.resourceLookup.GetTemplateInfo(ctx, fpr.TemplateName, fpr.TemplateVersion)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to lookup TemplateInfo '%s' version '%s': %v", fpr.TemplateName, fpr.TemplateVersion, err))
	} else if templateInfo == nil {
		errors = append(errors, fmt.Sprintf("Referenced TemplateInfo '%s' version '%s' does not exist", fpr.TemplateName, fpr.TemplateVersion))
	} else if templateInfo.State != models.StateApproved {
		errors = append(errors, fmt.Sprintf("Referenced TemplateInfo '%s' version '%s' is not in approved state (current state: %s)", fpr.TemplateName, fpr.TemplateVersion, templateInfo.State))
	}

	return models.NewValidationResult(len(errors) == 0, errors, warnings)
}

// ValidateOCloudDeletion validates that an OCloud can be safely deleted
func (dv *DependencyValidator) ValidateOCloudDeletion(ctx context.Context, oCloudID, oCloudNamespace string) *models.ValidationResult {
	var errors []string
	var warnings []string

	// Check for dependent FocomProvisioningRequests
	dependentFPRs, err := dv.resourceLookup.ListFocomProvisioningRequestsByOCloud(ctx, oCloudID, oCloudNamespace)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to check for dependent provisioning requests: %v", err))
	} else if len(dependentFPRs) > 0 {
		var dependentIDs []string
		for _, fpr := range dependentFPRs {
			if fpr.State == models.StateApproved {
				dependentIDs = append(dependentIDs, fpr.ID)
			}
		}
		if len(dependentIDs) > 0 {
			errors = append(errors, fmt.Sprintf("Cannot delete OCloud '%s' in namespace '%s': it is referenced by %d approved provisioning request(s): %v",
				oCloudID, oCloudNamespace, len(dependentIDs), dependentIDs))
		}
	}

	return models.NewValidationResult(len(errors) == 0, errors, warnings)
}

// ValidateTemplateInfoDeletion validates that a TemplateInfo can be safely deleted
func (dv *DependencyValidator) ValidateTemplateInfoDeletion(ctx context.Context, templateName, templateVersion string) *models.ValidationResult {
	var errors []string
	var warnings []string

	// Check for dependent FocomProvisioningRequests
	dependentFPRs, err := dv.resourceLookup.ListFocomProvisioningRequestsByTemplate(ctx, templateName, templateVersion)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to check for dependent provisioning requests: %v", err))
	} else if len(dependentFPRs) > 0 {
		var dependentIDs []string
		for _, fpr := range dependentFPRs {
			if fpr.State == models.StateApproved {
				dependentIDs = append(dependentIDs, fpr.ID)
			}
		}
		if len(dependentIDs) > 0 {
			errors = append(errors, fmt.Sprintf("Cannot delete TemplateInfo '%s' version '%s': it is referenced by %d approved provisioning request(s): %v",
				templateName, templateVersion, len(dependentIDs), dependentIDs))
		}
	}

	return models.NewValidationResult(len(errors) == 0, errors, warnings)
}

// ValidateResourceCreationOrder validates that resources are created in the correct dependency order
func (dv *DependencyValidator) ValidateResourceCreationOrder(ctx context.Context, resourceType models.ResourceType, dependencies []string) *models.ValidationResult {
	var errors []string
	var warnings []string

	switch resourceType {
	case models.ResourceTypeOCloud:
		// OCloud has no dependencies, can be created first
		if len(dependencies) > 0 {
			warnings = append(warnings, "OCloud resources should not have dependencies")
		}

	case models.ResourceTypeTemplateInfo:
		// TemplateInfo has no dependencies, can be created after OCloud
		if len(dependencies) > 0 {
			warnings = append(warnings, "TemplateInfo resources should not have dependencies")
		}

	case models.ResourceTypeFocomProvisioningRequest:
		// FPR requires both OCloud and TemplateInfo to exist
		if len(dependencies) < 2 {
			errors = append(errors, "FocomProvisioningRequest requires both OCloud and TemplateInfo dependencies")
		}

	default:
		errors = append(errors, fmt.Sprintf("Unknown resource type: %s", resourceType))
	}

	return models.NewValidationResult(len(errors) == 0, errors, warnings)
}

// GetDependencyError creates a structured dependency error
func (dv *DependencyValidator) GetDependencyError(resourceType models.ResourceType, resourceID string, dependentType models.ResourceType, dependentID, message string) *models.DependencyError {
	return models.NewDependencyError(resourceType, resourceID, dependentType, dependentID, message)
}

// ValidateCircularDependencies validates that there are no circular dependencies (future-proofing)
func (dv *DependencyValidator) ValidateCircularDependencies(ctx context.Context, resourceType models.ResourceType, resourceID string, dependencies []string) *models.ValidationResult {
	var errors []string
	var warnings []string

	// For the current FOCOM NBI design, circular dependencies are not possible
	// since we have a clear hierarchy: OCloud -> TemplateInfo -> FocomProvisioningRequest
	// This method is implemented for future extensibility

	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	if dv.hasCycle(resourceID, visited, recursionStack, dependencies) {
		errors = append(errors, fmt.Sprintf("Circular dependency detected for resource %s of type %s", resourceID, resourceType))
	}

	return models.NewValidationResult(len(errors) == 0, errors, warnings)
}

// hasCycle performs depth-first search to detect cycles
func (dv *DependencyValidator) hasCycle(resourceID string, visited, recursionStack map[string]bool, dependencies []string) bool {
	visited[resourceID] = true
	recursionStack[resourceID] = true

	for _, dep := range dependencies {
		if !visited[dep] {
			if dv.hasCycle(dep, visited, recursionStack, []string{}) {
				return true
			}
		} else if recursionStack[dep] {
			return true
		}
	}

	recursionStack[resourceID] = false
	return false
}
