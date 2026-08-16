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

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
)

// Validator defines the interface for validation operations
type Validator interface {
	// ValidateOCloud validates an OCloud configuration
	ValidateOCloud(ctx context.Context, ocloud *models.OCloudData) *models.ValidationResult

	// ValidateTemplateInfo validates a TemplateInfo configuration
	ValidateTemplateInfo(ctx context.Context, templateInfo *models.TemplateInfoData) *models.ValidationResult

	// ValidateFocomProvisioningRequest validates a FOCOM provisioning request
	ValidateFocomProvisioningRequest(ctx context.Context, fpr *models.FocomProvisioningRequestData) *models.ValidationResult

	// ValidateTemplateParameters validates template parameters against a schema
	ValidateTemplateParameters(ctx context.Context, parameters map[string]interface{}, schema string) *models.ValidationResult

	// ValidateJSON validates that a string contains valid JSON
	ValidateJSON(jsonStr string) error

	// ValidateYAML validates that a string contains valid YAML
	ValidateYAML(yamlStr string) error
}

// SchemaValidator defines the interface for schema validation
type SchemaValidator interface {
	// ValidateAgainstSchema validates data against a JSON schema and returns structured errors
	ValidateAgainstSchema(data interface{}, schema string) ([]models.SchemaValidationError, error)

	// ValidateSchema validates that a schema string is a valid JSON Schema document
	ValidateSchema(schema string) error
}

// BusinessRuleValidator defines the interface for business rule validation
type BusinessRuleValidator interface {
	// ValidateOCloudBusinessRules validates OCloud-specific business rules
	ValidateOCloudBusinessRules(ctx context.Context, ocloud *models.OCloudData) []string

	// ValidateTemplateInfoBusinessRules validates TemplateInfo-specific business rules
	ValidateTemplateInfoBusinessRules(ctx context.Context, templateInfo *models.TemplateInfoData) []string

	// ValidateFocomProvisioningRequestBusinessRules validates FPR-specific business rules
	ValidateFocomProvisioningRequestBusinessRules(ctx context.Context, fpr *models.FocomProvisioningRequestData) []string
}
