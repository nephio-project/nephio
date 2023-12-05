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

package bootstrappackages

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetResourcesPRR(t *testing.T) {

	kptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: xxx
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: xxx
`
	yamlFile1 := `apiVersion: f1/v1
kind: F1
metadata:
  name: f1
spec:
  description: xxx
`

	yamlFile2 := `apiVersion: f2/v1
kind: F2
metadata:
  name: f2
spec:
  description: xxx
`

	mdFile := `test`

	cases := map[string]struct {
		resources   map[string]string
		wanted      map[string]struct{}
		expectedErr bool
	}{
		"Normal": {
			resources: map[string]string{
				"a.md":    mdFile,
				"f1.yaml": yamlFile1,
				"f2.yaml": yamlFile2,
				"Kptfile": kptfile,
			},
			wanted: map[string]struct{}{
				"f2/v1.F2.f2": {},
				"f1/v1.F1.f1": {},
			},
			expectedErr: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := reconciler{}
			us, err := r.filterNonLocalResources(context.Background(), tc.resources)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if len(us) != len(tc.wanted) {
				t.Errorf("want %d, got: %d, data: %v", len(tc.wanted), len(us), us)
			}
			for _, u := range us {
				gvkn := fmt.Sprintf("%s.%s.%s", u.GetAPIVersion(), u.GetKind(), u.GetName())
				_, ok := tc.wanted[gvkn]
				if !ok {
					t.Errorf("got unexpected gvkn: %s us: %v", gvkn, us)
				}
				delete(tc.wanted, gvkn)
			}

		})
	}
}
