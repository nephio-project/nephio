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
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"gopkg.in/yaml.v2"
)

// ValidationService implements the Validator interface
type ValidationService struct {
	structValidator   *validator.Validate
	schemaValidator   SchemaValidator
	businessValidator BusinessRuleValidator
}

// NewValidationService creates a new ValidationService
func NewValidationService(schemaValidator SchemaValidator, businessValidator BusinessRuleValidator) *ValidationService {
	return &ValidationService{
		structValidator:   validator.New(),
		schemaValidator:   schemaValidator,
		businessValidator: businessValidator,
	}
}

// ValidateOCloud validates an OCloud configuration
func (v *ValidationService) ValidateOCloud(ctx context.Context, ocloud *models.OCloudData) *models.ValidationResult {
	var errors []string
	var warnings []string

	// Struct validation
	if err := v.structValidator.Struct(ocloud); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, fmt.Sprintf("Field '%s' validation failed: %s", err.Field(), err.Tag()))
		}
	}

	// Business rule validation
	if v.businessValidator != nil {
		businessErrors := v.businessValidator.ValidateOCloudBusinessRules(ctx, ocloud)
		errors = append(errors, businessErrors...)
	}

	// Additional OCloud-specific validations
	if ocloud.O2IMSSecret.SecretRef.Name == "" {
		errors = append(errors, "O2IMS secret name cannot be empty")
	}
	if ocloud.O2IMSSecret.SecretRef.Namespace == "" {
		errors = append(errors, "O2IMS secret namespace cannot be empty")
	}

	return models.NewValidationResult(len(errors) == 0, errors, warnings)
}

// ValidateTemplateInfo validates a TemplateInfo configuration
func (v *ValidationService) ValidateTemplateInfo(ctx context.Context, templateInfo *models.TemplateInfoData) *models.ValidationResult {
	var errors []string
	var warnings []string

	// Struct validation
	if err := v.structValidator.Struct(templateInfo); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, fmt.Sprintf("Field '%s' validation failed: %s", err.Field(), err.Tag()))
		}
	}

	// Template parameter schema validation
	if templateInfo.TemplateParameterSchema != "" {
		if err := v.ValidateJSON(templateInfo.TemplateParameterSchema); err != nil {
			// Try YAML if JSON fails
			if yamlErr := v.ValidateYAML(templateInfo.TemplateParameterSchema); yamlErr != nil {
				errors = append(errors, fmt.Sprintf("Template parameter schema is neither valid JSON nor YAML: JSON error: %v, YAML error: %v", err, yamlErr))
			}
		}

		// Schema metavalidation: verify the templateParameterSchema is a valid JSON Schema document
		if v.schemaValidator != nil {
			if err := v.schemaValidator.ValidateSchema(templateInfo.TemplateParameterSchema); err != nil {
				errors = append(errors, fmt.Sprintf("Template parameter schema is not a valid JSON Schema: %v", err))
			}
		}
	}

	// Business rule validation
	if v.businessValidator != nil {
		businessErrors := v.businessValidator.ValidateTemplateInfoBusinessRules(ctx, templateInfo)
		errors = append(errors, businessErrors...)
	}

	// Additional TemplateInfo-specific validations
	if templateInfo.TemplateName == "" {
		errors = append(errors, "Template name cannot be empty")
	}
	if templateInfo.TemplateVersion == "" {
		errors = append(errors, "Template version cannot be empty")
	}

	return models.NewValidationResult(len(errors) == 0, errors, warnings)
}

// ValidateFocomProvisioningRequest validates a FOCOM provisioning request
func (v *ValidationService) ValidateFocomProvisioningRequest(ctx context.Context, fpr *models.FocomProvisioningRequestData) *models.ValidationResult {
	var errors []string
	var warnings []string

	// Struct validation
	if err := v.structValidator.Struct(fpr); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, fmt.Sprintf("Field '%s' validation failed: %s", err.Field(), err.Tag()))
		}
	}

	// Business rule validation
	if v.businessValidator != nil {
		businessErrors := v.businessValidator.ValidateFocomProvisioningRequestBusinessRules(ctx, fpr)
		errors = append(errors, businessErrors...)
	}

	// Additional FPR-specific validations
	if fpr.OCloudID == "" {
		errors = append(errors, "OCloud ID cannot be empty")
	}
	if fpr.OCloudNamespace == "" {
		errors = append(errors, "OCloud namespace cannot be empty")
	}
	if fpr.TemplateName == "" {
		errors = append(errors, "Template name cannot be empty")
	}
	if fpr.TemplateVersion == "" {
		errors = append(errors, "Template version cannot be empty")
	}
	if fpr.TemplateParameters == nil {
		errors = append(errors, "Template parameters cannot be nil")
	}

	return models.NewValidationResult(len(errors) == 0, errors, warnings)
}

// ValidateTemplateParameters validates template parameters against a schema
func (v *ValidationService) ValidateTemplateParameters(ctx context.Context, parameters map[string]interface{}, schema string) *models.ValidationResult {
	var errors []string
	var warnings []string
	var schemaErrors []models.SchemaValidationError

	if v.schemaValidator != nil {
		validationErrors, err := v.schemaValidator.ValidateAgainstSchema(parameters, schema)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Template parameters validation failed: %v", err))
		} else if len(validationErrors) > 0 {
			for _, schemaErr := range validationErrors {
				errors = append(errors, fmt.Sprintf("Template parameter '%s' violates constraint '%s': %s", schemaErr.Field, schemaErr.Constraint, schemaErr.Description))
				schemaErrors = append(schemaErrors, schemaErr)
			}
		}
	} else {
		warnings = append(warnings, "Schema validator not available, skipping schema validation")
	}

	result := models.NewValidationResult(len(errors) == 0, errors, warnings)
	result.SchemaErrors = schemaErrors
	return result
}

// ValidateJSON validates that a string contains valid JSON
func (v *ValidationService) ValidateJSON(jsonStr string) error {
	var js interface{}
	return json.Unmarshal([]byte(jsonStr), &js)
}

// ValidateYAML validates that a string contains valid YAML
func (v *ValidationService) ValidateYAML(yamlStr string) error {
	var ys interface{}
	return yaml.Unmarshal([]byte(yamlStr), &ys)
}
