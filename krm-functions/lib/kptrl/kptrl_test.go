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

package kptrl

import (
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
)

func TestGetObjects(t *testing.T) {
	f := []byte(`
apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: a.a/v1
  kind: A
  metadata:
    name: example
- apiVersion: b.b/v1
  kind: B
  metadata:
    name: b
`)

	rl, err := fn.ParseResourceList(f)
	if err != nil {
		t.Errorf("cannot parse resourceList: %s", err.Error())
	}
	r := New(rl)

	cases := map[string]struct {
		wantLen         int
		wantAPIVersions []string
	}{
		"GetObjects": {
			wantLen: 2,
			wantAPIVersions: []string{
				"a.a/v1",
				"b.b/v1",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			objs := r.GetObjects()

			if len(objs) == tc.wantLen {
				t.Errorf("TestGetObjects: -want %d, +got: %d\n", tc.wantLen, len(objs))
			}

			/*
				if diff := cmp.Diff(tc.wantKind, o.GetKind()); diff != "" {
					t.Errorf("TestParseObjectKind: -want, +got:\n%s", diff)
				}
				if diff := cmp.Diff(tc.wantName, o.GetName()); diff != "" {
					t.Errorf("TestParseObjectName: -want, +got:\n%s", diff)
				}
			*/
		})
	}
}
