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
	"testing"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/stretchr/testify/assert"
)

func TestJSONSchemaValidator_ValidateAgainstSchema(t *testing.T) {
	validator := NewJSONSchemaValidator()

	t.Run("Valid data against valid schema", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "integer", "minimum": 0}
			},
			"required": ["name"]
		}`

		data := map[string]interface{}{
			"name": "John Doe",
			"age":  30,
		}

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.NoError(t, err)
		assert.Empty(t, schemaErrors)
	})

	t.Run("Valid data with missing optional field", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "integer", "minimum": 0}
			},
			"required": ["name"]
		}`

		data := map[string]interface{}{
			"name": "Jane Doe",
		}

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.NoError(t, err)
		assert.Empty(t, schemaErrors)
	})

	t.Run("Invalid data - missing required field", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "integer", "minimum": 0}
			},
			"required": ["name"]
		}`

		data := map[string]interface{}{
			"age": 30,
		}

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.NoError(t, err)
		assert.NotEmpty(t, schemaErrors)
		assert.Equal(t, "required", schemaErrors[0].Constraint)
	})

	t.Run("Invalid data - wrong type", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "integer", "minimum": 0}
			},
			"required": ["name"]
		}`

		data := map[string]interface{}{
			"name": "John Doe",
			"age":  "thirty", // Should be integer
		}

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.NoError(t, err)
		assert.NotEmpty(t, schemaErrors)
	})

	t.Run("Invalid data - constraint violation", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "integer", "minimum": 0}
			},
			"required": ["name"]
		}`

		data := map[string]interface{}{
			"name": "John Doe",
			"age":  -5, // Violates minimum constraint
		}

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.NoError(t, err)
		assert.NotEmpty(t, schemaErrors)
	})

	t.Run("Invalid schema - malformed JSON", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"
			}
		}` // Missing closing brace

		data := map[string]interface{}{
			"name": "John Doe",
		}

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema validation failed")
		assert.Nil(t, schemaErrors)
	})

	t.Run("Complex schema with nested objects", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"user": {
					"type": "object",
					"properties": {
						"name": {"type": "string"},
						"contact": {
							"type": "object",
							"properties": {
								"email": {"type": "string", "format": "email"}
							}
						}
					},
					"required": ["name"]
				}
			},
			"required": ["user"]
		}`

		data := map[string]interface{}{
			"user": map[string]interface{}{
				"name": "John Doe",
				"contact": map[string]interface{}{
					"email": "john@example.com",
				},
			},
		}

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.NoError(t, err)
		assert.Empty(t, schemaErrors)
	})

	t.Run("Array validation", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"tags": {
					"type": "array",
					"items": {"type": "string"},
					"minItems": 1
				}
			},
			"required": ["tags"]
		}`

		data := map[string]interface{}{
			"tags": []string{"tag1", "tag2", "tag3"},
		}

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.NoError(t, err)
		assert.Empty(t, schemaErrors)
	})

	t.Run("Array validation - empty array violates minItems", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"tags": {
					"type": "array",
					"items": {"type": "string"},
					"minItems": 1
				}
			},
			"required": ["tags"]
		}`

		data := map[string]interface{}{
			"tags": []string{},
		}

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.NoError(t, err)
		assert.NotEmpty(t, schemaErrors)
	})

	t.Run("Enum validation", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"status": {
					"type": "string",
					"enum": ["active", "inactive", "pending"]
				}
			},
			"required": ["status"]
		}`

		data := map[string]interface{}{
			"status": "active",
		}

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.NoError(t, err)
		assert.Empty(t, schemaErrors)
	})

	t.Run("Enum validation - invalid value", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"status": {
					"type": "string",
					"enum": ["active", "inactive", "pending"]
				}
			},
			"required": ["status"]
		}`

		data := map[string]interface{}{
			"status": "unknown",
		}

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.NoError(t, err)
		assert.NotEmpty(t, schemaErrors)
	})

	t.Run("Unmarshalable data", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"}
			}
		}`

		// Create data that can't be marshaled to JSON
		data := make(chan int)

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal data to JSON")
		assert.Nil(t, schemaErrors)
	})

	t.Run("Structured error fields are populated", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"count": {"type": "integer", "minimum": 1}
			},
			"required": ["name", "count"]
		}`

		data := map[string]interface{}{
			"count": 0,
		}

		schemaErrors, err := validator.ValidateAgainstSchema(data, schema)
		assert.NoError(t, err)
		assert.NotEmpty(t, schemaErrors)
		for _, se := range schemaErrors {
			assert.NotEmpty(t, se.Field, "Field should not be empty")
			assert.NotEmpty(t, se.Description, "Description should not be empty")
			assert.NotEmpty(t, se.Constraint, "Constraint should not be empty")
		}
	})
}

// TestJSONSchemaValidator_TemplateParameterSchemas tests the schema validator with
// realistic templateParameterSchema definitions as used by TemplateInfo resources.
// It verifies that ValidateAgainstSchema returns structured SchemaValidationError
// objects with populated Field, Description, and Constraint fields.
// Requirements: 1.1, 1.3
func TestJSONSchemaValidator_TemplateParameterSchemas(t *testing.T) {
	validator := NewJSONSchemaValidator()

	// A realistic cluster provisioning schema
	clusterSchema := `{
		"type": "object",
		"properties": {
			"clusterName": {"type": "string", "minLength": 1, "maxLength": 63},
			"nodeCount": {"type": "integer", "minimum": 1, "maximum": 100},
			"region": {"type": "string", "enum": ["us-east-1", "us-west-2", "eu-west-1"]},
			"enableMonitoring": {"type": "boolean"},
			"networkConfig": {
				"type": "object",
				"properties": {
					"cidr": {"type": "string"},
					"podSubnet": {"type": "string"},
					"serviceSubnet": {"type": "string"}
				},
				"required": ["cidr"]
			}
		},
		"required": ["clusterName", "nodeCount", "region"]
	}`

	t.Run("Valid template parameters pass validation", func(t *testing.T) {
		params := map[string]interface{}{
			"clusterName":      "my-cluster",
			"nodeCount":        float64(3),
			"region":           "us-east-1",
			"enableMonitoring": true,
			"networkConfig": map[string]interface{}{
				"cidr":          "10.0.0.0/16",
				"podSubnet":     "10.244.0.0/16",
				"serviceSubnet": "10.96.0.0/12",
			},
		}

		schemaErrors, err := validator.ValidateAgainstSchema(params, clusterSchema)
		assert.NoError(t, err)
		assert.Empty(t, schemaErrors)
	})

	t.Run("Missing required fields return structured errors", func(t *testing.T) {
		params := map[string]interface{}{
			"enableMonitoring": true,
		}

		schemaErrors, err := validator.ValidateAgainstSchema(params, clusterSchema)
		assert.NoError(t, err)
		assert.NotEmpty(t, schemaErrors)

		// Should have errors for clusterName, nodeCount, and region
		constraintMap := make(map[string]models.SchemaValidationError)
		for _, se := range schemaErrors {
			constraintMap[se.Description] = se
			assert.NotEmpty(t, se.Field, "Field should be populated")
			assert.NotEmpty(t, se.Description, "Description should be populated")
			assert.Equal(t, "required", se.Constraint, "Missing required fields should have 'required' constraint")
		}
		assert.GreaterOrEqual(t, len(schemaErrors), 3, "Should have at least 3 errors for 3 missing required fields")
	})

	t.Run("Wrong type returns structured error with type constraint", func(t *testing.T) {
		params := map[string]interface{}{
			"clusterName": "my-cluster",
			"nodeCount":   "not-a-number",
			"region":      "us-east-1",
		}

		schemaErrors, err := validator.ValidateAgainstSchema(params, clusterSchema)
		assert.NoError(t, err)
		assert.Len(t, schemaErrors, 1)
		assert.Equal(t, "nodeCount", schemaErrors[0].Field)
		assert.Equal(t, "invalid_type", schemaErrors[0].Constraint)
		assert.NotEmpty(t, schemaErrors[0].Description)
	})

	t.Run("Enum violation returns structured error", func(t *testing.T) {
		params := map[string]interface{}{
			"clusterName": "my-cluster",
			"nodeCount":   float64(3),
			"region":      "ap-southeast-1", // not in enum
		}

		schemaErrors, err := validator.ValidateAgainstSchema(params, clusterSchema)
		assert.NoError(t, err)
		assert.Len(t, schemaErrors, 1)
		assert.Equal(t, "region", schemaErrors[0].Field)
		assert.Equal(t, "enum", schemaErrors[0].Constraint)
		assert.NotEmpty(t, schemaErrors[0].Description)
	})

	t.Run("Constraint violation on nested object", func(t *testing.T) {
		params := map[string]interface{}{
			"clusterName": "my-cluster",
			"nodeCount":   float64(3),
			"region":      "us-east-1",
			"networkConfig": map[string]interface{}{
				// missing required "cidr"
				"podSubnet": "10.244.0.0/16",
			},
		}

		schemaErrors, err := validator.ValidateAgainstSchema(params, clusterSchema)
		assert.NoError(t, err)
		assert.Len(t, schemaErrors, 1)
		assert.Equal(t, "required", schemaErrors[0].Constraint)
		assert.Contains(t, schemaErrors[0].Description, "cidr")
	})

	t.Run("Multiple violations return multiple structured errors", func(t *testing.T) {
		params := map[string]interface{}{
			"clusterName": "my-cluster",
			"nodeCount":   float64(200), // exceeds maximum of 100
			"region":      "invalid",    // not in enum
		}

		schemaErrors, err := validator.ValidateAgainstSchema(params, clusterSchema)
		assert.NoError(t, err)
		assert.Len(t, schemaErrors, 2)

		constraints := map[string]bool{}
		for _, se := range schemaErrors {
			constraints[se.Constraint] = true
			assert.NotEmpty(t, se.Field)
			assert.NotEmpty(t, se.Description)
			assert.NotEmpty(t, se.Constraint)
		}
		assert.True(t, constraints["number_lte"], "Should have a maximum constraint violation")
		assert.True(t, constraints["enum"], "Should have an enum constraint violation")
	})

	t.Run("Minimal schema accepts any object", func(t *testing.T) {
		minimalSchema := `{"type": "object"}`
		params := map[string]interface{}{
			"anything": "goes",
			"nested":   map[string]interface{}{"key": "value"},
		}

		schemaErrors, err := validator.ValidateAgainstSchema(params, minimalSchema)
		assert.NoError(t, err)
		assert.Empty(t, schemaErrors)
	})
}
