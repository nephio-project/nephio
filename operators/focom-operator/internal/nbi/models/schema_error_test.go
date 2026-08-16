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

package models

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
)

// genSchemaValidationError generates random SchemaValidationError values
func genSchemaValidationError() gopter.Gen {
	return gen.Struct(reflect.TypeOf(SchemaValidationError{}), map[string]gopter.Gen{
		"Field":       gen.AlphaString(),
		"Description": gen.AlphaString(),
		"Constraint":  gen.AlphaString(),
	})
}

// **Feature: fpr-schema-validation, Property 4: Error serialization round-trip**
// **Validates: Requirements 4.2**
func TestSchemaValidationError_RoundTrip_Property(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("serializing then deserializing SchemaValidationError produces equal value", prop.ForAll(
		func(original SchemaValidationError) bool {
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}

			var restored SchemaValidationError
			err = json.Unmarshal(data, &restored)
			if err != nil {
				return false
			}

			return original == restored
		},
		genSchemaValidationError(),
	))

	properties.TestingRun(t)
}

// Sanity check: a specific example to complement the property test
func TestSchemaValidationError_RoundTrip_Example(t *testing.T) {
	original := SchemaValidationError{
		Field:       "nodeCount",
		Description: "Must be greater than or equal to 1",
		Constraint:  "minimum",
	}

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	var restored SchemaValidationError
	err = json.Unmarshal(data, &restored)
	assert.NoError(t, err)
	assert.Equal(t, original, restored)
}
