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
	"regexp"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
)

// BusinessRuleValidatorImpl implements BusinessRuleValidator
type BusinessRuleValidatorImpl struct{}

// NewBusinessRuleValidator creates a new BusinessRuleValidatorImpl
func NewBusinessRuleValidator() *BusinessRuleValidatorImpl {
	return &BusinessRuleValidatorImpl{}
}

// ValidateOCloudBusinessRules validates OCloud-specific business rules
func (brv *BusinessRuleValidatorImpl) ValidateOCloudBusinessRules(ctx context.Context, ocloud *models.OCloudData) []string {
	var errors []string

	// Validate name format (should be DNS-1123 compliant)
	if !isValidKubernetesName(ocloud.Name) {
		errors = append(errors, "OCloud name must be a valid Kubernetes resource name (DNS-1123 compliant)")
	}

	// Validate namespace format
	if !isValidKubernetesName(ocloud.Namespace) {
		errors = append(errors, "OCloud namespace must be a valid Kubernetes namespace name")
	}

	// Validate description length
	if len(ocloud.Description) > 1000 {
		errors = append(errors, "OCloud description must not exceed 1000 characters")
	}

	// Validate secret reference
	if !isValidKubernetesName(ocloud.O2IMSSecret.SecretRef.Name) {
		errors = append(errors, "O2IMS secret name must be a valid Kubernetes resource name")
	}
	if !isValidKubernetesName(ocloud.O2IMSSecret.SecretRef.Namespace) {
		errors = append(errors, "O2IMS secret namespace must be a valid Kubernetes namespace name")
	}

	return errors
}

// ValidateTemplateInfoBusinessRules validates TemplateInfo-specific business rules
func (brv *BusinessRuleValidatorImpl) ValidateTemplateInfoBusinessRules(ctx context.Context, templateInfo *models.TemplateInfoData) []string {
	var errors []string

	// Validate name format
	if !isValidKubernetesName(templateInfo.Name) {
		errors = append(errors, "TemplateInfo name must be a valid Kubernetes resource name (DNS-1123 compliant)")
	}

	// Validate namespace format
	if !isValidKubernetesName(templateInfo.Namespace) {
		errors = append(errors, "TemplateInfo namespace must be a valid Kubernetes namespace name")
	}

	// Validate description length
	if len(templateInfo.Description) > 1000 {
		errors = append(errors, "TemplateInfo description must not exceed 1000 characters")
	}

	// Validate template name format
	if !isValidTemplateName(templateInfo.TemplateName) {
		errors = append(errors, "Template name must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Validate template version format (semantic versioning)
	if !isValidSemanticVersion(templateInfo.TemplateVersion) {
		errors = append(errors, "Template version must follow semantic versioning format (e.g., v1.2.3)")
	}

	// Validate template parameter schema length
	if len(templateInfo.TemplateParameterSchema) > 10000 {
		errors = append(errors, "Template parameter schema must not exceed 10000 characters")
	}

	return errors
}

// ValidateFocomProvisioningRequestBusinessRules validates FPR-specific business rules
func (brv *BusinessRuleValidatorImpl) ValidateFocomProvisioningRequestBusinessRules(ctx context.Context, fpr *models.FocomProvisioningRequestData) []string {
	var errors []string

	// Validate name format
	if !isValidKubernetesName(fpr.Name) {
		errors = append(errors, "Provisioning request name must be a valid Kubernetes resource name (DNS-1123 compliant)")
	}

	// Validate namespace format
	if !isValidKubernetesName(fpr.Namespace) {
		errors = append(errors, "Provisioning request namespace must be a valid Kubernetes namespace name")
	}

	// Validate description length
	if len(fpr.Description) > 1000 {
		errors = append(errors, "Provisioning request description must not exceed 1000 characters")
	}

	// Validate OCloud ID format (should be a valid Kubernetes resource name)
	if !isValidKubernetesName(fpr.OCloudID) {
		errors = append(errors, "OCloud ID must be a valid Kubernetes resource name (DNS-1123 compliant)")
	}

	// Validate OCloud namespace format
	if !isValidKubernetesName(fpr.OCloudNamespace) {
		errors = append(errors, "OCloud namespace must be a valid Kubernetes namespace name")
	}

	// Validate template name format
	if !isValidTemplateName(fpr.TemplateName) {
		errors = append(errors, "Template name must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Validate template version format
	if !isValidSemanticVersion(fpr.TemplateVersion) {
		errors = append(errors, "Template version must follow semantic versioning format (e.g., v1.2.3)")
	}

	// Validate template parameters
	if len(fpr.TemplateParameters) == 0 {
		errors = append(errors, "Template parameters cannot be empty")
	}

	// Validate template parameters size (prevent excessively large payloads)
	if len(fpr.TemplateParameters) > 100 {
		errors = append(errors, "Template parameters cannot exceed 100 key-value pairs")
	}

	return errors
}

// Helper functions for validation

// isValidKubernetesName validates that a name follows Kubernetes DNS-1123 naming rules
func isValidKubernetesName(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}

	// DNS-1123 label: lowercase alphanumeric characters or '-', start and end with alphanumeric
	pattern := `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`

	matched, _ := regexp.MatchString(pattern, name)
	return matched
}

// isValidTemplateName validates template name format
func isValidTemplateName(name string) bool {
	if len(name) == 0 || len(name) > 100 {
		return false
	}

	// Allow alphanumeric, hyphens, underscores
	pattern := `^[a-zA-Z0-9_-]+$`

	matched, _ := regexp.MatchString(pattern, name)
	return matched
}

// isValidSemanticVersion validates semantic version format
func isValidSemanticVersion(version string) bool {
	if len(version) == 0 {
		return false
	}

	// Basic semantic version pattern (with optional 'v' prefix)
	pattern := `^v?([0-9]+)\.([0-9]+)\.([0-9]+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`

	matched, _ := regexp.MatchString(pattern, version)
	return matched
}
