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
	"fmt"
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
		"ForErrorRefDepth": {
			kc: &gvkKindCtx{gvkKind: forGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
				{APIVersion: "b", Kind: "b", Name: "b"},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: true,
		},
		"WatchGlobalNormal": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: false,
		},
		"WatchSpecificNormal": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
				{APIVersion: "b", Kind: "b", Name: "b"},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: false,
		},
		"WatchErrorDepth": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
				{APIVersion: "b", Kind: "b", Name: "b"},
				{APIVersion: "c", Kind: "c", Name: "c"},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: true,
		},

		"OwnSpecificNormal": {
			kc: &gvkKindCtx{gvkKind: ownGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
				{APIVersion: "b", Kind: "b", Name: "b"},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: false,
		},
		"OwnErrorDepthTooSmall": {
			kc: &gvkKindCtx{gvkKind: ownGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: true,
		},
		"OwnErrorDepth": {
			kc: &gvkKindCtx{gvkKind: ownGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
				{APIVersion: "b", Kind: "b", Name: "b"},
				{APIVersion: "c", Kind: "c", Name: "c"},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			inv, err := newInventory(&Config{
				For:                corev1.ObjectReference{APIVersion: "a", Kind: "a"},
				GenerateResourceFn: GenerateResourceFnNop,
			})
			if err != nil {
				assert.NoError(t, err)
			}

			err = inv.set(tc.kc, tc.refs, tc.x, tc.new)
			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				x := inv.get(tc.kc.gvkKind, tc.refs)
				fmt.Printf("get %s: %v, len: %d\n", name, x, len(x))
				if len(x) != 1 {
					t.Errorf("expecting %v, got %v", tc.refs[0], x)
				}
			}
		})
	}
}

func TestDelete(t *testing.T) {
	cases := map[string]struct {
		kc          *gvkKindCtx
		refs        []corev1.ObjectReference
		deleteRefs  []corev1.ObjectReference
		x           any
		new         newResource
		errExpected bool
	}{
		"DeleteNormal": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: false,
		},
		"DeleteNotFound": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
				{APIVersion: "b", Kind: "b", Name: "b"},
			},
			deleteRefs: []corev1.ObjectReference{
				{APIVersion: "b", Kind: "b", Name: "b"},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			inv, err := newInventory(&Config{
				For:                corev1.ObjectReference{APIVersion: "a", Kind: "a"},
				GenerateResourceFn: GenerateResourceFnNop,
			})
			if err != nil {
				assert.NoError(t, err)
			}

			err = inv.set(tc.kc, tc.refs, tc.x, tc.new)
			if err != nil {
				assert.NoError(t, err)
			}

			deleteRefs := tc.refs
			if tc.deleteRefs != nil {
				deleteRefs = tc.deleteRefs
			}
			err = inv.delete(tc.kc, deleteRefs)
			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGet(t *testing.T) {
	cases := map[string]struct {
		kc          *gvkKindCtx
		setRefs     [][]corev1.ObjectReference
		getRefs     []corev1.ObjectReference
		x           any
		new         newResource
		errExpected bool
		len         int
	}{
		"GetEmpty": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			getRefs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: false,
			len:         0,
		},
		"WildcardWatch": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			setRefs: [][]corev1.ObjectReference{
				{{APIVersion: "a", Kind: "a", Name: "a"}},
				{{APIVersion: "b", Kind: "b", Name: "b"}},
			},
			getRefs: []corev1.ObjectReference{
				{},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: false,
			len:         2,
		},
		"WildcardFor": {
			kc: &gvkKindCtx{gvkKind: forGVKKind},
			setRefs: [][]corev1.ObjectReference{
				{{APIVersion: "a", Kind: "a", Name: "a"}},
				{{APIVersion: "b", Kind: "b", Name: "b"}},
			},
			getRefs: []corev1.ObjectReference{
				{},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: false,
			len:         2,
		},
		"WildCardOwn": {
			kc: &gvkKindCtx{gvkKind: ownGVKKind},
			setRefs: [][]corev1.ObjectReference{
				{{APIVersion: "a", Kind: "a", Name: "a"}, {APIVersion: "x", Kind: "x", Name: "x"}},
				{{APIVersion: "a", Kind: "a", Name: "a"}, {APIVersion: "y", Kind: "y", Name: "y"}},
				{{APIVersion: "a", Kind: "a", Name: "a"}, {APIVersion: "z", Kind: "z", Name: "z"}},
			},
			getRefs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"}, {},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: false,
			len:         3,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			inv, err := newInventory(&Config{
				For:                corev1.ObjectReference{APIVersion: "a", Kind: "a"},
				GenerateResourceFn: GenerateResourceFnNop,
			})
			if err != nil {
				assert.NoError(t, err)
			}

			for _, refs := range tc.setRefs {
				inv.set(tc.kc, refs, tc.x, tc.new)
			}

			x := inv.get(tc.kc.gvkKind, tc.getRefs)
			if len(x) != tc.len {
				t.Errorf("expecting len %d, got %v", tc.len, x)
			}
		})
	}
}

func TestList(t *testing.T) {
	cases := map[string]struct {
		kc          *gvkKindCtx
		refs        [][]corev1.ObjectReference
		x           any
		new         newResource
		errExpected bool
	}{
		"List": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			refs: [][]corev1.ObjectReference{
				{
					{APIVersion: "a", Kind: "a", Name: "a"},
				},
				{
					{APIVersion: "b1", Kind: "b1", Name: "b1"},
					{APIVersion: "b11", Kind: "b11", Name: "b11"},
				},
				{
					{APIVersion: "b1", Kind: "b1", Name: "b1"},
					{APIVersion: "b12", Kind: "b12", Name: "b12"},
				},
			},
			x:           fn.NewEmptyKubeObject(),
			errExpected: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			inv, err := newInventory(&Config{
				For:                corev1.ObjectReference{APIVersion: "a", Kind: "a"},
				GenerateResourceFn: GenerateResourceFnNop,
			})
			if err != nil {
				assert.NoError(t, err)
			}

			for _, ref := range tc.refs {
				inv.set(tc.kc, ref, tc.x, tc.new)
			}

			x := inv.list()
			if len(x) != 4 {
				t.Errorf("expecting len 4, got len: %d, data %v", len(tc.refs), x)
			}
		})
	}
}
