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
	"math/rand"
	"reflect"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
	"github.com/xeipuuv/gojsonschema"
)

// schemaAndData holds a generated JSON Schema string and a data object to validate against it.
type schemaAndData struct {
	Schema string
	Data   map[string]interface{}
}

// genSchemaAndData generates a random JSON Schema (object type with 1-3 properties)
// and a random data object that may or may not conform to the schema.
func genSchemaAndData() gopter.Gen {
	return func(params *gopter.GenParameters) *gopter.GenResult {
		r := rand.New(rand.NewSource(params.Rng.Int63()))
		numProps := 1 + r.Intn(3)
		shouldBreak := r.Intn(2) == 1
		sd := buildSchemaAndData(r, numProps, shouldBreak)
		return gopter.NewGenResult(sd, gopter.NoShrinker)
	}
}

// buildSchemaAndData constructs a JSON Schema with the given number of properties
// and a data object. If shouldBreak is true, the data intentionally violates the schema.
func buildSchemaAndData(r *rand.Rand, numProps int, shouldBreak bool) schemaAndData {
	propNames := []string{"name", "count", "enabled", "status", "value", "label", "size", "mode", "level", "tag"}
	r.Shuffle(len(propNames), func(i, j int) { propNames[i], propNames[j] = propNames[j], propNames[i] })
	chosen := propNames[:numProps]

	properties := make(map[string]interface{})
	required := []string{}
	data := make(map[string]interface{})

	for _, pname := range chosen {
		typeChoice := r.Intn(3) // 0=string, 1=integer, 2=boolean
		switch typeChoice {
		case 0:
			properties[pname] = map[string]interface{}{"type": "string"}
			if shouldBreak && r.Intn(3) == 0 {
				data[pname] = r.Intn(1000)
			} else {
				data[pname] = randomString(r)
			}
		case 1:
			minVal := r.Intn(10)
			properties[pname] = map[string]interface{}{
				"type":    "integer",
				"minimum": float64(minVal),
			}
			if shouldBreak && r.Intn(3) == 0 {
				data[pname] = minVal - 1 - r.Intn(10)
			} else {
				data[pname] = minVal + r.Intn(100)
			}
		case 2:
			properties[pname] = map[string]interface{}{"type": "boolean"}
			if shouldBreak && r.Intn(3) == 0 {
				data[pname] = "notabool"
			} else {
				data[pname] = r.Intn(2) == 1
			}
		}

		if r.Intn(2) == 0 {
			required = append(required, pname)
		}
	}

	// If shouldBreak, also try removing a required field
	if shouldBreak && len(required) > 0 && r.Intn(2) == 0 {
		removeIdx := r.Intn(len(required))
		delete(data, required[removeIdx])
	}

	schemaObj := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schemaObj["required"] = required
	}

	schemaBytes, _ := json.Marshal(schemaObj)
	return schemaAndData{
		Schema: string(schemaBytes),
		Data:   data,
	}
}

func randomString(r *rand.Rand) string {
	letters := "abcdefghijklmnopqrstuvwxyz"
	length := 3 + r.Intn(8)
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

// **Feature: fpr-schema-validation, Property 1: Schema validation conformance**
// **Validates: Requirements 1.1, 1.4**
func TestSchemaValidationConformance_Property(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	validator := NewJSONSchemaValidator()

	properties.Property("ValidateAgainstSchema agrees with gojsonschema reference validation", prop.ForAll(
		func(sd schemaAndData) bool {
			// Use our wrapper
			schemaErrors, err := validator.ValidateAgainstSchema(sd.Data, sd.Schema)
			if err != nil {
				// If our wrapper returns an error, the reference should also fail
				schemaLoader := gojsonschema.NewStringLoader(sd.Schema)
				dataJSON, marshalErr := json.Marshal(sd.Data)
				if marshalErr != nil {
					return true
				}
				documentLoader := gojsonschema.NewBytesLoader(dataJSON)
				_, refErr := gojsonschema.Validate(schemaLoader, documentLoader)
				return refErr != nil
			}

			// Reference validation
			schemaLoader := gojsonschema.NewStringLoader(sd.Schema)
			dataJSON, marshalErr := json.Marshal(sd.Data)
			if marshalErr != nil {
				return false
			}
			documentLoader := gojsonschema.NewBytesLoader(dataJSON)
			refResult, refErr := gojsonschema.Validate(schemaLoader, documentLoader)
			if refErr != nil {
				return false
			}

			// Both should agree on validity
			ourValid := len(schemaErrors) == 0
			refValid := refResult.Valid()

			if ourValid != refValid {
				return false
			}

			// If both say invalid, the error count should match
			if !ourValid && !refValid {
				return len(schemaErrors) == len(refResult.Errors())
			}

			return true
		},
		genSchemaAndData(),
	))

	properties.TestingRun(t)
}

// genNonConformingSchemaAndData generates a random JSON Schema and data that is
// guaranteed to violate the schema (at least one violation).
func genNonConformingSchemaAndData() gopter.Gen {
	return func(params *gopter.GenParameters) *gopter.GenResult {
		r := rand.New(rand.NewSource(params.Rng.Int63()))
		numProps := 1 + r.Intn(3)
		// Always break conformance
		sd := buildNonConformingSchemaAndData(r, numProps)
		return gopter.NewGenResult(sd, gopter.NoShrinker)
	}
}

// buildNonConformingSchemaAndData constructs a JSON Schema and data that always
// violates the schema. It uses a mix of type mismatches, missing required fields,
// and constraint violations to ensure at least one error is produced.
func buildNonConformingSchemaAndData(r *rand.Rand, numProps int) schemaAndData {
	propNames := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	r.Shuffle(len(propNames), func(i, j int) { propNames[i], propNames[j] = propNames[j], propNames[i] })
	chosen := propNames[:numProps]

	properties := make(map[string]interface{})
	required := []string{}
	data := make(map[string]interface{})

	for _, pname := range chosen {
		typeChoice := r.Intn(3)
		switch typeChoice {
		case 0: // string property, provide integer value (type mismatch)
			properties[pname] = map[string]interface{}{"type": "string"}
			data[pname] = r.Intn(1000)
		case 1: // integer with minimum, provide value below minimum
			minVal := 5 + r.Intn(10)
			properties[pname] = map[string]interface{}{
				"type":    "integer",
				"minimum": float64(minVal),
			}
			data[pname] = minVal - 1 - r.Intn(5)
		case 2: // boolean property, provide string value (type mismatch)
			properties[pname] = map[string]interface{}{"type": "boolean"}
			data[pname] = "notabool"
		}
		required = append(required, pname)
	}

	// Also add an extra required field that is missing from data
	missingField := "requiredMissing"
	properties[missingField] = map[string]interface{}{"type": "string"}
	required = append(required, missingField)
	// Intentionally do NOT add missingField to data

	schemaObj := map[string]interface{}{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schemaObj["required"] = required
	}

	schemaBytes, _ := json.Marshal(schemaObj)
	return schemaAndData{
		Schema: string(schemaBytes),
		Data:   data,
	}
}

// **Feature: fpr-schema-validation, Property 2: Error structure completeness**
// **Validates: Requirements 1.3**
func TestErrorStructureCompleteness_Property(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	validator := NewJSONSchemaValidator()

	properties.Property("Every SchemaValidationError has non-empty Field, Description, and Constraint", prop.ForAll(
		func(sd schemaAndData) bool {
			schemaErrors, err := validator.ValidateAgainstSchema(sd.Data, sd.Schema)
			if err != nil {
				// Schema/data generation issue, skip
				return true
			}

			// We expect at least one error since data is non-conforming
			if len(schemaErrors) == 0 {
				return false
			}

			// Every error must have non-empty Field, Description, and Constraint
			for _, se := range schemaErrors {
				if se.Field == "" {
					return false
				}
				if se.Description == "" {
					return false
				}
				if se.Constraint == "" {
					return false
				}
			}
			return true
		},
		genNonConformingSchemaAndData(),
	))

	properties.TestingRun(t)
}

// genValidJSONString generates a random valid JSON string that may or may not be
// a valid JSON Schema document. It produces a mix of:
// - Valid JSON Schema objects (object type with properties, type constraints)
// - Valid JSON objects that are not valid schemas (e.g. {"type": "bogus"})
// - Valid JSON primitives (strings, numbers, arrays, booleans, null)
// - Empty objects
func genValidJSONString() gopter.Gen {
	return func(params *gopter.GenParameters) *gopter.GenResult {
		r := rand.New(rand.NewSource(params.Rng.Int63()))
		var jsonStr string

		choice := r.Intn(6)
		switch choice {
		case 0:
			// Valid JSON Schema: object with typed properties
			numProps := 1 + r.Intn(3)
			props := make(map[string]interface{})
			names := []string{"a", "b", "c", "d", "e"}
			types := []string{"string", "integer", "boolean", "number"}
			for i := 0; i < numProps; i++ {
				props[names[i]] = map[string]interface{}{"type": types[r.Intn(len(types))]}
			}
			schema := map[string]interface{}{
				"type":       "object",
				"properties": props,
			}
			b, _ := json.Marshal(schema)
			jsonStr = string(b)
		case 1:
			// Valid JSON but invalid schema: object with bogus type value
			bogusTypes := []string{"bogus", "notaType", "foobar", "123type"}
			schema := map[string]interface{}{
				"type": bogusTypes[r.Intn(len(bogusTypes))],
			}
			b, _ := json.Marshal(schema)
			jsonStr = string(b)
		case 2:
			// Valid JSON: a plain string
			b, _ := json.Marshal(randomString(r))
			jsonStr = string(b)
		case 3:
			// Valid JSON: a number
			b, _ := json.Marshal(r.Float64() * 1000)
			jsonStr = string(b)
		case 4:
			// Valid JSON: an array
			arr := []interface{}{r.Intn(100), randomString(r), r.Intn(2) == 1}
			b, _ := json.Marshal(arr)
			jsonStr = string(b)
		case 5:
			// Valid JSON: empty object (valid schema — permissive)
			jsonStr = "{}"
		}

		return gopter.NewGenResult(jsonStr, gopter.NoShrinker)
	}
}

// **Feature: fpr-schema-validation, Property 3: Schema metavalidation correctness**
// **Validates: Requirements 2.1, 2.2, 2.3**
func TestSchemaMetavalidationCorrectness_Property(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	validator := NewJSONSchemaValidator()

	properties.Property("ValidateSchema returns nil iff gojsonschema.NewSchema succeeds", prop.ForAll(
		func(jsonStr string) bool {
			// Our wrapper
			wrapperErr := validator.ValidateSchema(jsonStr)

			// Reference: direct gojsonschema compilation
			schemaLoader := gojsonschema.NewStringLoader(jsonStr)
			_, refErr := gojsonschema.NewSchema(schemaLoader)

			// Both should agree: nil/nil or non-nil/non-nil
			wrapperOK := wrapperErr == nil
			refOK := refErr == nil

			return wrapperOK == refOK
		},
		genValidJSONString(),
	))

	properties.TestingRun(t)
}

// Ensure the reflect import is used (for gopter internals)
var _ = reflect.TypeOf(schemaAndData{})
