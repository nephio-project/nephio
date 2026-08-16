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
	"encoding/json"
	"fmt"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/xeipuuv/gojsonschema"
)

// JSONSchemaValidator implements SchemaValidator using JSON Schema
type JSONSchemaValidator struct{}

// NewJSONSchemaValidator creates a new JSONSchemaValidator
func NewJSONSchemaValidator() *JSONSchemaValidator {
	return &JSONSchemaValidator{}
}

// ValidateAgainstSchema validates data against a JSON schema and returns structured errors.
// It returns a slice of SchemaValidationError for each violation found, or a non-nil error
// if the validation process itself fails (e.g. bad schema, unmarshalable data).
func (jsv *JSONSchemaValidator) ValidateAgainstSchema(data interface{}, schema string) ([]models.SchemaValidationError, error) {
	// Convert data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	// Create schema loader
	schemaLoader := gojsonschema.NewStringLoader(schema)

	// Create document loader
	documentLoader := gojsonschema.NewBytesLoader(dataJSON)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	if !result.Valid() {
		var schemaErrors []models.SchemaValidationError
		for _, desc := range result.Errors() {
			schemaErrors = append(schemaErrors, models.SchemaValidationError{
				Field:       desc.Field(),
				Description: desc.Description(),
				Constraint:  desc.Type(),
			})
		}
		return schemaErrors, nil
	}

	return nil, nil
}

// ValidateSchema validates that a schema string is a valid JSON Schema document
// by attempting to compile it with the gojsonschema library.
func (jsv *JSONSchemaValidator) ValidateSchema(schema string) error {
	schemaLoader := gojsonschema.NewStringLoader(schema)
	_, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("invalid JSON Schema: %w", err)
	}
	return nil
}
