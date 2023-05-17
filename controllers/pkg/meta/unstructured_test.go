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

package meta

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGetUnstructuredFromGVK(t *testing.T) {
	cases := map[string]struct {
		gvk            *schema.GroupVersionKind
		wantAPIVersion string
		wantKind       string
	}{
		"Normal": {
			gvk: &schema.GroupVersionKind{Group: "a", Version: "b", Kind: "c"},
			wantAPIVersion: "a/b",
			wantKind: "c",
		},
		"EmptyKind": {
			gvk: &schema.GroupVersionKind{Group: "a", Version: "b", Kind: ""},
			wantAPIVersion: "a/b",
			wantKind: "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			u := GetUnstructuredFromGVK(tc.gvk)

			if diff := cmp.Diff(tc.wantAPIVersion, u.GetAPIVersion()); diff != "" {
				t.Errorf("-want error, +got\n%s\n", diff)
			}
			if diff := cmp.Diff(tc.wantKind, u.GetKind()); diff != "" {
				t.Errorf("-want error, +got\n%s\n", diff)
			}
		})
	}
}
