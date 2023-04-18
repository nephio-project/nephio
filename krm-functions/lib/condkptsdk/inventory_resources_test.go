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

package condkptsdk

import (
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestSet(t *testing.T) {
	cases := map[string]struct {
		kc          *gvkKindCtx
		refs        []corev1.ObjectReference
		x           any
		new         newResource
		errExpected bool
	}{
		"ForNormal": {
			kc: &gvkKindCtx{gvkKind: forGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: false,
		},
		"ForErrorRefs": {
			kc: &gvkKindCtx{gvkKind: forGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
				{APIVersion: "a", Kind: "a", Name: "a"},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			i, err := newInventory(&Config{
				For:                corev1.ObjectReference{APIVersion: "a", Kind: "a"},
				GenerateResourceFn: GenerateResourceFnNop,
			})
			if err != nil {
				assert.NoError(t, err)
			}

			err = i.set(tc.kc, tc.refs, tc.x, tc.new)
			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
