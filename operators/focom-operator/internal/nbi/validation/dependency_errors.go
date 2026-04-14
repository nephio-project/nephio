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
	"fmt"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
)

// DependencyErrorBuilder provides methods to create standardized dependency error messages
type DependencyErrorBuilder struct{}

// NewDependencyErrorBuilder creates a new DependencyErrorBuilder
func NewDependencyErrorBuilder() *DependencyErrorBuilder {
	return &DependencyErrorBuilder{}
}

// BuildMissingDependencyError creates an error message for missing dependencies
func (deb *DependencyErrorBuilder) BuildMissingDependencyError(resourceType models.ResourceType, resourceID string, dependencyType models.ResourceType, dependencyID, dependencyNamespace string) string {
	switch dependencyType {
	case models.ResourceTypeOCloud:
		return fmt.Sprintf("Cannot approve %s '%s': referenced OCloud '%s' in namespace '%s' does not exist or is not in approved state",
			resourceType, resourceID, dependencyID, dependencyNamespace)
	case models.ResourceTypeTemplateInfo:
		return fmt.Sprintf("Cannot approve %s '%s': referenced TemplateInfo '%s' does not exist or is not in approved state",
			resourceType, resourceID, dependencyID)
	default:
		return fmt.Sprintf("Cannot approve %s '%s': referenced %s '%s' does not exist or is not in approved state",
			resourceType, resourceID, dependencyType, dependencyID)
	}
}

// BuildDeletionBlockedError creates an error message for deletion blocked by dependencies
func (deb *DependencyErrorBuilder) BuildDeletionBlockedError(resourceType models.ResourceType, resourceID, namespace string, dependentCount int, dependentIDs []string) string {
	var resourceIdentifier string
	if namespace != "" {
		resourceIdentifier = fmt.Sprintf("%s '%s' in namespace '%s'", resourceType, resourceID, namespace)
	} else {
		resourceIdentifier = fmt.Sprintf("%s '%s'", resourceType, resourceID)
	}

	if dependentCount == 1 {
		return fmt.Sprintf("Cannot delete %s: it is referenced by 1 approved provisioning request: %s",
			resourceIdentifier, dependentIDs[0])
	}

	return fmt.Sprintf("Cannot delete %s: it is referenced by %d approved provisioning requests: %v",
		resourceIdentifier, dependentCount, dependentIDs)
}

// BuildInvalidStateError creates an error message for invalid resource state
func (deb *DependencyErrorBuilder) BuildInvalidStateError(resourceType models.ResourceType, resourceID string, currentState models.ResourceState, requiredState models.ResourceState) string {
	return fmt.Sprintf("Referenced %s '%s' is in '%s' state, but '%s' state is required",
		resourceType, resourceID, currentState, requiredState)
}

// BuildCircularDependencyError creates an error message for circular dependencies
func (deb *DependencyErrorBuilder) BuildCircularDependencyError(resourceType models.ResourceType, resourceID string, dependencyChain []string) string {
	return fmt.Sprintf("Circular dependency detected for %s '%s': dependency chain %v creates a cycle",
		resourceType, resourceID, dependencyChain)
}

// BuildInvalidCreationOrderError creates an error message for invalid resource creation order
func (deb *DependencyErrorBuilder) BuildInvalidCreationOrderError(resourceType models.ResourceType, missingDependencies []string) string {
	switch resourceType {
	case models.ResourceTypeFocomProvisioningRequest:
		return fmt.Sprintf("FocomProvisioningRequest requires both OCloud and TemplateInfo dependencies to be created first. Missing: %v", missingDependencies)
	case models.ResourceTypeTemplateInfo:
		return fmt.Sprintf("TemplateInfo should be created after OCloud resources. Unexpected dependencies: %v", missingDependencies)
	case models.ResourceTypeOCloud:
		return fmt.Sprintf("OCloud should be created first and should not have dependencies. Unexpected dependencies: %v", missingDependencies)
	default:
		return fmt.Sprintf("Invalid creation order for %s. Missing dependencies: %v", resourceType, missingDependencies)
	}
}

// BuildLookupFailureError creates an error message for dependency lookup failures
func (deb *DependencyErrorBuilder) BuildLookupFailureError(resourceType models.ResourceType, resourceID string, dependencyType models.ResourceType, dependencyID string, err error) string {
	return fmt.Sprintf("Failed to lookup %s '%s' while validating dependencies for %s '%s': %v",
		dependencyType, dependencyID, resourceType, resourceID, err)
}

// BuildDependencyValidationSummary creates a summary message for dependency validation results
func (deb *DependencyErrorBuilder) BuildDependencyValidationSummary(resourceType models.ResourceType, resourceID string, validationResult *models.ValidationResult) string {
	if validationResult.Success {
		return fmt.Sprintf("All dependencies for %s '%s' are valid", resourceType, resourceID)
	}

	errorCount := len(validationResult.Errors)
	warningCount := len(validationResult.Warnings)

	summary := fmt.Sprintf("Dependency validation failed for %s '%s'", resourceType, resourceID)

	if errorCount > 0 {
		summary += fmt.Sprintf(" with %d error(s)", errorCount)
	}

	if warningCount > 0 {
		summary += fmt.Sprintf(" and %d warning(s)", warningCount)
	}

	return summary
}

// BuildResourceNotFoundError creates an error message for resource not found during dependency validation
func (deb *DependencyErrorBuilder) BuildResourceNotFoundError(resourceType models.ResourceType, resourceID, namespace string) string {
	if namespace != "" {
		return fmt.Sprintf("%s '%s' not found in namespace '%s'", resourceType, resourceID, namespace)
	}
	return fmt.Sprintf("%s '%s' not found", resourceType, resourceID)
}

// BuildApprovalPrerequisiteError creates an error message for approval prerequisite failures
func (deb *DependencyErrorBuilder) BuildApprovalPrerequisiteError(resourceType models.ResourceType, resourceID string, prerequisiteErrors []string) string {
	if len(prerequisiteErrors) == 1 {
		return fmt.Sprintf("Cannot approve %s '%s': %s", resourceType, resourceID, prerequisiteErrors[0])
	}

	return fmt.Sprintf("Cannot approve %s '%s': multiple prerequisite failures: %v",
		resourceType, resourceID, prerequisiteErrors)
}

// BuildDependencyChainError creates an error message for dependency chain validation failures
func (deb *DependencyErrorBuilder) BuildDependencyChainError(resourceType models.ResourceType, resourceID string, chainErrors []string) string {
	return fmt.Sprintf("Dependency chain validation failed for %s '%s': %v",
		resourceType, resourceID, chainErrors)
}

// GetUserFriendlyResourceTypeName returns a user-friendly name for resource types
func (deb *DependencyErrorBuilder) GetUserFriendlyResourceTypeName(resourceType models.ResourceType) string {
	switch resourceType {
	case models.ResourceTypeOCloud:
		return "O-Cloud Configuration"
	case models.ResourceTypeTemplateInfo:
		return "Template Information"
	case models.ResourceTypeFocomProvisioningRequest:
		return "FOCOM Provisioning Request"
	default:
		return string(resourceType)
	}
}

// BuildDetailedValidationReport creates a detailed validation report with all errors and warnings
func (deb *DependencyErrorBuilder) BuildDetailedValidationReport(resourceType models.ResourceType, resourceID string, validationResult *models.ValidationResult) map[string]interface{} {
	report := map[string]interface{}{
		"resourceType":   deb.GetUserFriendlyResourceTypeName(resourceType),
		"resourceId":     resourceID,
		"validationTime": validationResult.ValidationTime,
		"success":        validationResult.Success,
		"errorCount":     len(validationResult.Errors),
		"warningCount":   len(validationResult.Warnings),
	}

	if len(validationResult.Errors) > 0 {
		report["errors"] = validationResult.Errors
	}

	if len(validationResult.Warnings) > 0 {
		report["warnings"] = validationResult.Warnings
	}

	report["summary"] = deb.BuildDependencyValidationSummary(resourceType, resourceID, validationResult)

	return report
}

// BuildRecommendationMessage creates a recommendation message for fixing dependency issues
func (deb *DependencyErrorBuilder) BuildRecommendationMessage(resourceType models.ResourceType, validationResult *models.ValidationResult) []string {
	var recommendations []string

	for _, err := range validationResult.Errors {
		switch resourceType {
		case models.ResourceTypeFocomProvisioningRequest:
			if contains(err, "OCloud") && contains(err, "does not exist") {
				recommendations = append(recommendations, "Create and approve the required OCloud configuration before approving this provisioning request")
			}
			if contains(err, "TemplateInfo") && contains(err, "does not exist") {
				recommendations = append(recommendations, "Create and approve the required TemplateInfo configuration before approving this provisioning request")
			}
			if contains(err, "not in approved state") {
				recommendations = append(recommendations, "Ensure all referenced resources are validated and approved before approving this provisioning request")
			}
		case models.ResourceTypeOCloud:
			if contains(err, "referenced by") {
				recommendations = append(recommendations, "Delete or modify all dependent provisioning requests before deleting this OCloud configuration")
			}
		case models.ResourceTypeTemplateInfo:
			if contains(err, "referenced by") {
				recommendations = append(recommendations, "Delete or modify all dependent provisioning requests before deleting this TemplateInfo configuration")
			}
		}
	}

	if len(recommendations) == 0 && !validationResult.Success {
		recommendations = append(recommendations, "Review the error messages above and ensure all dependencies are properly configured")
	}

	return recommendations
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
