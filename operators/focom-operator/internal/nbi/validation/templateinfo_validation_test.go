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
	"strings"
	"testing"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/stretchr/testify/assert"
)

// newTestTemplateInfo creates a TemplateInfoData with the given schema and valid defaults for other fields.
func newTestTemplateInfo(schema string) *models.TemplateInfoData {
	return &models.TemplateInfoData{
		BaseResource: models.BaseResource{
			ID:          "ti-1",
			Namespace:   "default",
			Name:        "test-template",
			Description: "Test template",
			State:       models.StateDraft,
		},
		TemplateName:            "test-template",
		TemplateVersion:         "v1.0.0",
		TemplateParameterSchema: schema,
	}
}

func TestValidateTemplateInfo_SchemaMetavalidation(t *testing.T) {
	ctx := context.Background()
	schemaValidator := NewJSONSchemaValidator()
	svc := NewValidationService(schemaValidator, nil)

	t.Run("Valid JSON Schema passes metavalidation", func(t *testing.T) {
		ti := newTestTemplateInfo(`{
			"type": "object",
			"properties": {
				"clusterName": {"type": "string"},
				"nodeCount": {"type": "integer", "minimum": 1}
			},
			"required": ["clusterName"]
		}`)

		result := svc.ValidateTemplateInfo(ctx, ti)

		assert.True(t, result.Success, "expected validation to pass, got errors: %v", result.Errors)
		assert.Empty(t, result.Errors)
	})

	t.Run("Valid JSON but invalid schema fails metavalidation", func(t *testing.T) {
		ti := newTestTemplateInfo(`{"type": "bogus"}`)

		result := svc.ValidateTemplateInfo(ctx, ti)

		assert.False(t, result.Success, "expected validation to fail for invalid schema type")
		foundSchemaError := false
		for _, e := range result.Errors {
			if strings.Contains(e, "not a valid JSON Schema") {
				foundSchemaError = true
				break
			}
		}
		assert.True(t, foundSchemaError, "expected error about invalid JSON Schema, got: %v", result.Errors)
	})

	t.Run("Empty schema string fails validation", func(t *testing.T) {
		ti := newTestTemplateInfo("")

		result := svc.ValidateTemplateInfo(ctx, ti)

		// Empty string fails the struct validator's "required" tag on TemplateParameterSchema
		assert.False(t, result.Success, "expected validation to fail for empty schema")
		assert.NotEmpty(t, result.Errors)
	})
}
