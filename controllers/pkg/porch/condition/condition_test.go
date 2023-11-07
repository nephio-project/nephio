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

package porchcondition

import (
	"testing"

	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/google/go-cmp/cmp"
)

func TestGetPorchConditions(t *testing.T) {
	cases := map[string]struct {
		t    []kptv1.Condition
		want []porchv1alpha1.Condition
	}{
		"Normal": {
			t: []kptv1.Condition{
				{
					Type:   "a",
					Status: "True",
					Reason: "b",
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			pc := GetPorchConditions(tc.t)

			if len(pc) != len(tc.t) {
				t.Errorf("unexpected conditions: -want: %d, -got: %d", len(tc.t), len(pc))
			}
			for i, c := range pc {
				if diff := cmp.Diff(string(c.Status), string(tc.t[i].Status)); diff != "" {
					t.Errorf("-want, +got:\n%s", diff)
				}
				if diff := cmp.Diff(c.Type, tc.t[i].Type); diff != "" {
					t.Errorf("-want, +got:\n%s", diff)
				}
				if diff := cmp.Diff(c.Reason, tc.t[i].Reason); diff != "" {
					t.Errorf("-want, +got:\n%s", diff)
				}
				if diff := cmp.Diff(c.Message, tc.t[i].Message); diff != "" {
					t.Errorf("-want, +got:\n%s", diff)
				}
			}
		})
	}
}

func TestHasSpecificTypeConditions(t *testing.T) {
	cases := map[string]struct {
		t    []porchv1alpha1.Condition
		s    string
		want bool
	}{
		"Found": {
			t: []porchv1alpha1.Condition{
				{
					Type: "a.b.b",
				},
				{
					Type: "a.b.b",
				},
			},
			s:    "a.b",
			want: true,
		},
		"NotFound": {
			t: []porchv1alpha1.Condition{
				{
					Type: "a.b.b",
				},
				{
					Type: "a.b.b",
				},
			},
			s:    "c.b",
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := HasSpecificTypeConditions(tc.t, tc.s)
			if diff := cmp.Diff(b, tc.want); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}
		})
	}
}

func TestPackageRevisionIsReady(t *testing.T) {
	cases := map[string]struct {
		conds    []porchv1alpha1.Condition
		readyGates []porchv1alpha1.ReadinessGate
		want bool
	}{
		"Ready": {
			conds: []porchv1alpha1.Condition{
				{
					Type: "foo",
					Status: porchv1alpha1.ConditionStatus(porchv1alpha1.ConditionTrue),
				},
				{
					Type: "foobar",
					Status: porchv1alpha1.ConditionStatus(porchv1alpha1.ConditionFalse),
				},
				{
					Type: "myterriblecondition",
					Status: porchv1alpha1.ConditionStatus(porchv1alpha1.ConditionFalse),
				},
			},
			readyGates:   []porchv1alpha1.ReadinessGate{
				{
					ConditionType: "foo",
				},
			},
			want: true,
		},
		"Not ready": {
			conds: []porchv1alpha1.Condition{
				{
					Type: "bar",
					Status: porchv1alpha1.ConditionStatus(porchv1alpha1.ConditionFalse),
				},
			},
			readyGates:   []porchv1alpha1.ReadinessGate{
				{
					ConditionType: "bar",
				},
			},
			want: false,
		},
		"Empty readinessGates": {
			conds: []porchv1alpha1.Condition{
				{
					Type: "bar",
					Status: porchv1alpha1.ConditionStatus(porchv1alpha1.ConditionTrue),
				},
			},
			readyGates:   []porchv1alpha1.ReadinessGate{},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := PackageRevisionIsReady(tc.readyGates, tc.conds)
			if diff := cmp.Diff(b, tc.want); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}
		})
	}
}