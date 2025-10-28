/*
Copyright 2023 Nephio.

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

package fn

import (
	"testing"

	"github.com/kptdev/krm-functions-sdk/go/fn"
	"github.com/stretchr/testify/assert"
)

func TestErrorCases(t *testing.T) {
	cases := map[string]struct {
		input       []byte
		errExpected string
	}{
		"MissingKey": {
			input: []byte(`
apiVersion: fn.kpt.dev/v1alpha1
kind: GenConfigMap
metadata:
  name: empty-test
data:
- type: literal
  value: foo`),
			errExpected: "data entry 0, key must not be empty",
		},
		"GoTemplateParseError": {
			input: []byte(`
apiVersion: fn.kpt.dev/v1alpha1
kind: GenConfigMap
metadata:
  name: empty-test
data:
- type: gotmpl
  key: bad
  value: foo{{`),
			errExpected: "data entry 0 generate error: template: bad:1: unclosed action",
		},
		"GoTemplateExecuteError": {
			input: []byte(`
apiVersion: fn.kpt.dev/v1alpha1
kind: GenConfigMap
metadata:
  name: empty-test
data:
- type: gotmpl
  key: bad
  value: foo{{template "nope"}}`),
			errExpected: "data entry 0 generate error: template: bad:1:14: executing \"bad\" at <{{template \"nope\"}}>: template \"nope\" not defined",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ko, err := fn.ParseKubeObject(tc.input)
			assert.NoError(t, err)

			_, err = Process(&fn.ResourceList{FunctionConfig: ko})
			if tc.errExpected != "" {
				assert.EqualError(t, err, tc.errExpected)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
