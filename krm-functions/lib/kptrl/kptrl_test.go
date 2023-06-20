/*
 Copyright 2023 The Nephio Authors.

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

package kptrl

import (
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/stretchr/testify/assert"
)

var objA = []byte(`
apiVersion: a.a/v1
kind: A
metadata:
  name: a
  labels:
    a: a
`)

var objB = []byte(`
apiVersion: b.b/v1
kind: B
metadata:
  name: b
  labels:
    b: b
`)

var objC = []byte(`
apiVersion: c.c/v1
kind: C
metadata:
  name: c
  labels:
    c: c
`)

func TestGetResourceList(t *testing.T) {
	cases := map[string]struct {
		t    map[string]string
		want fn.ResourceList
	}{
		"Normal": {
			t: map[string]string{
				"a.yaml": string(objA),
				"b.yaml": string(objB),
				"c.yaml": string(objC)},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			rl, err := GetResourceList(tc.t)
			if err != nil {
				assert.NoError(t, err)
			}
			if len(rl.Items) != len(tc.t) {
				t.Errorf("ResouceList: -want: nil, +got:%v\n", rl.Items)
			}
		})
	}
}
