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
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestIsReady(t *testing.T) {
	cases := map[string]struct {
		kc    *gvkKindCtx
		refs  []corev1.ObjectReference
		o     *fn.KubeObject
		c     *kptv1.Condition
		new   newResource
		ready bool
	}{
		"WatchEmpty": {
			kc:    &gvkKindCtx{gvkKind: watchGVKKind},
			refs:  []corev1.ObjectReference{},
			o:     fn.NewEmptyKubeObject(),
			new:   false,
			ready: true,
		},

		"WatchExistingResourceExists": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
			},
			o:     fn.NewEmptyKubeObject(),
			new:   false,
			ready: true,
		},

		"WatchExistingConditionTrue": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
			},
			c: &kptv1.Condition{
				Type:   "ok",
				Status: kptv1.ConditionTrue,
			},
			new:   false,
			ready: true,
		},
		"WatchExistingResourceDoesNotExists": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
			},
			o:     fn.NewEmptyKubeObject(),
			new:   true,
			ready: false,
		},
		"WatchExistingConditionFalse": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
			},
			c: &kptv1.Condition{
				Type:   "nok",
				Status: kptv1.ConditionFalse,
			},
			new:   true,
			ready: false,
		},
		"WatchExistingConditionFalseWithExistingKubeObject": {
			kc: &gvkKindCtx{gvkKind: watchGVKKind},
			refs: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "a", Name: "a"},
			},
			o: fn.NewEmptyKubeObject(),
			c: &kptv1.Condition{
				Type:   "nok",
				Status: kptv1.ConditionFalse,
			},
			new:   true,
			ready: false,
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

			if len(tc.refs) != 0 {
				if tc.o != nil {
					err = inv.set(tc.kc, tc.refs, tc.o, tc.new)
					if err != nil {
						assert.NoError(t, err)
					}
				}
				if tc.c != nil {
					err = inv.set(tc.kc, tc.refs, tc.c, tc.new)
					if err != nil {
						assert.NoError(t, err)
					}
				}
			}
			x := inv.isReady()
			if x != tc.ready {
				t.Errorf("want %t, got %v", tc.ready, x)
			}
		})
	}
}

/*
func TestGetReadyMap(t *testing.T) {
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
*/
