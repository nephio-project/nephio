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

package v1

import (
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/nephio-project/porch/pkg/kpt/api/kptfile/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

var f1 = `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: xxx
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: xxx
`

var f2 = `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: xxx
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: xxx
status:
  conditions:
  - type: a
    status: "False"
    reason: a
    message: a
  - type: b
    status: "False"
    reason: b
    message: b
`

func TestGetReadinessGates(t *testing.T) {
	cases := map[string]struct {
		rg   []string
		want []kptv1.ReadinessGate
	}{
		"Exists": {
			rg: []string{"a"},
			want: []kptv1.ReadinessGate{
				{ConditionType: "a"},
			},
		},
		"Empty": {
			rg:   nil,
			want: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ko, err := fn.ParseKubeObject([]byte(f1))
			if err != nil {
				assert.Error(t, err)
			}
			kf := KptFile{Kptfile: ko}

			if tc.rg != nil {
				if err := kf.SetReadinessGates(tc.rg...); err != nil {
					assert.Error(t, err)
				}
			}
			got := kf.GetReadinessGates()

			if got == nil || tc.want == nil {
				if len(got) != len(tc.want) {
					t.Errorf("-want%s, +got:\n%s", tc.want, got)
				}
			} else {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("-want, +got:\n%s", diff)
				}
			}
		})
	}
}

func TestHasReadinessGates(t *testing.T) {
	cases := map[string]struct {
		rg   []string
		ct   string
		want bool
	}{
		"Has": {
			rg:   []string{"a"},
			ct:   "a",
			want: true,
		},
		"HasNot": {
			rg:   []string{"a"},
			ct:   "b",
			want: false,
		},
		"Empty": {
			rg:   nil,
			ct:   "a",
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ko, err := fn.ParseKubeObject([]byte(f1))
			if err != nil {
				assert.Error(t, err)
			}
			kf := KptFile{Kptfile: ko}

			if tc.rg != nil {
				if err := kf.SetReadinessGates(tc.rg...); err != nil {
					assert.Error(t, err)
				}
			}
			got := kf.HasReadinessGate(tc.ct)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}
		})
	}
}

func TestSetReadinessGates(t *testing.T) {
	cases := map[string]struct {
		rg   []string
		want []kptv1.ReadinessGate
	}{
		"Exists": {
			rg: []string{"a", "b", "c"},
			want: []kptv1.ReadinessGate{
				{ConditionType: "a"},
				{ConditionType: "b"},
				{ConditionType: "c"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ko, err := fn.ParseKubeObject([]byte(f1))
			if err != nil {
				assert.Error(t, err)
			}
			kf := KptFile{Kptfile: ko}

			for _, rg := range tc.rg {
				if err := kf.SetReadinessGates(rg); err != nil {
					assert.Error(t, err)
				}
			}

			got := kf.GetReadinessGates()

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetCondition(t *testing.T) {
	cases := map[string]struct {
		cs   []kptv1.Condition
		t    string
		want *kptv1.Condition
	}{
		"Exists": {
			cs: []kptv1.Condition{
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
				{Type: "c", Status: kptv1.ConditionFalse, Reason: "c", Message: "c"},
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
			},
			t:    "a",
			want: &kptv1.Condition{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
		},
		"NotExists": {
			cs: []kptv1.Condition{
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
				{Type: "c", Status: kptv1.ConditionFalse, Reason: "c", Message: "c"},
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
			},
			t:    "x",
			want: nil,
		},
		"EmptyList": {
			cs:   nil,
			t:    "x",
			want: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ko, err := fn.ParseKubeObject([]byte(f1))
			if err != nil {
				assert.Error(t, err)
			}
			kf := KptFile{Kptfile: ko}
			if tc.cs != nil {
				if err := kf.SetConditions(tc.cs...); err != nil {
					assert.Error(t, err)
				}
			}
			got := kf.GetCondition(tc.t)
			if err != nil {
				assert.Error(t, err)
			}
			if got == nil || tc.want == nil {
				if got != tc.want {
					t.Errorf("-want%s, +got:\n%s", tc.want, got)
				}
			} else {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("-want, +got:\n%s", diff)
				}
			}
		})
	}
}

func TestSetConditions(t *testing.T) {
	cases := map[string]struct {
		cs   []kptv1.Condition
		t    []kptv1.Condition
		want []kptv1.Condition
	}{
		"StartEmpty": {
			cs: nil,
			t: []kptv1.Condition{
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
				{Type: "c", Status: kptv1.ConditionFalse, Reason: "c", Message: "c"},
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
			},
			want: []kptv1.Condition{
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
				{Type: "c", Status: kptv1.ConditionFalse, Reason: "c", Message: "c"},
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
			},
		},
		"StartNonEmpty": {
			cs: []kptv1.Condition{
				{Type: "x", Status: kptv1.ConditionFalse, Reason: "x", Message: "x"},
				{Type: "y", Status: kptv1.ConditionFalse, Reason: "y", Message: "y"},
			},
			t: []kptv1.Condition{
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
				{Type: "c", Status: kptv1.ConditionFalse, Reason: "c", Message: "c"},
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
			},
			want: []kptv1.Condition{
				{Type: "x", Status: kptv1.ConditionFalse, Reason: "x", Message: "x"},
				{Type: "y", Status: kptv1.ConditionFalse, Reason: "y", Message: "y"},
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
				{Type: "c", Status: kptv1.ConditionFalse, Reason: "c", Message: "c"},
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
			},
		},
		"Overlap": {
			cs: []kptv1.Condition{
				{Type: "x", Status: kptv1.ConditionFalse, Reason: "x", Message: "x"},
				{Type: "y", Status: kptv1.ConditionFalse, Reason: "y", Message: "y"},
			},
			t: []kptv1.Condition{
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
				{Type: "y", Status: kptv1.ConditionFalse, Reason: "ynew", Message: "ynew"},
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
			},
			want: []kptv1.Condition{
				{Type: "x", Status: kptv1.ConditionFalse, Reason: "x", Message: "x"},
				{Type: "y", Status: kptv1.ConditionFalse, Reason: "ynew", Message: "ynew"},
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ko, err := fn.ParseKubeObject([]byte(f1))
			if err != nil {
				assert.Error(t, err)
			}
			kf := KptFile{Kptfile: ko}
			if tc.cs != nil {
				if err := kf.SetConditions(tc.cs...); err != nil {
					assert.Error(t, err)
				}
			}
			if tc.cs != nil {
				kf.SetConditions(tc.cs...)
			}
			if tc.t != nil {
				kf.SetConditions(tc.t...)
			}

			got := kf.GetConditions()
			if len(got) != len(tc.want) {
				t.Errorf("want: %v, got: %v", tc.want, got)
			} else {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("-want, +got:\n%s", diff)
				}
			}
		})
	}
}

func TestDeleteCondition(t *testing.T) {
	cases := map[string]struct {
		t    []string
		want []kptv1.Condition
	}{
		"First": {
			t: []string{"a"},
			want: []kptv1.Condition{
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
			},
		},
		"Last": {
			t: []string{"b"},
			want: []kptv1.Condition{
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
			},
		},
		"All": {
			t:    []string{"b", "a"},
			want: []kptv1.Condition{},
		},
		"Unknown": {
			t: []string{"c"},
			want: []kptv1.Condition{
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ko, err := fn.ParseKubeObject([]byte(f2))
			if err != nil {
				assert.Error(t, err)
			}
			kf := KptFile{Kptfile: ko}

			for _, t := range tc.t {
				kf.DeleteCondition(t)
			}
			gots := kf.GetConditions()
			if len(gots) != len(tc.want) {
				t.Errorf("TestDeleteCondition: got: %v, want: %v", gots, tc.want)
			} else {
				for idx, got := range gots {
					if diff := cmp.Diff(tc.want[idx], got); diff != "" {
						t.Errorf("TestDeleteCondition: -want, +got:\n%s", diff)
					}
				}
			}
		})
	}
}
