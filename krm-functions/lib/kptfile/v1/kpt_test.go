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

	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/google/go-cmp/cmp"
)

func TestParseKubeObject(t *testing.T) {
	f := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: xxx
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: xxx
`

	kf, err := New(f)
	if err != nil {
		t.Errorf("cannot unmarshal file: %s", err.Error())
	}

	cases := map[string]struct {
		wantKind string
		wantName string
	}{
		"ParseObject": {
			wantKind: "Kptfile",
			wantName: "xxx",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			o, err := kf.ParseKubeObject()
			if err != nil {
				t.Errorf("cannot parse object: %s", err.Error())
			}

			if diff := cmp.Diff(tc.wantKind, o.GetKind()); diff != "" {
				t.Errorf("TestParseObjectKind: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantName, o.GetName()); diff != "" {
				t.Errorf("TestParseObjectName: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	f := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: xxx
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: xxx
`

	kf, err := New(f)
	if err != nil {
		t.Errorf("cannot unmarshal file: %s", err.Error())
	}

	cases := map[string]struct {
	}{
		"Marshal": {},
	}

	for name := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := kf.Marshal()
			if err != nil {
				t.Errorf("cannot parse object: %s", err.Error())
			}
		})
	}
}

func TestGetCondition(t *testing.T) {

	f := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: xxx
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: xxx
`

	kf, err := New(f)
	if err != nil {
		t.Errorf("cannot unmarshal file: %s", err.Error())
	}

	cases := map[string]struct {
		cs   []kptv1.Condition
		t    string
		want *kptv1.Condition
	}{
		"ConditionExists": {
			cs: []kptv1.Condition{
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
				{Type: "c", Status: kptv1.ConditionFalse, Reason: "c", Message: "c"},
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
			},
			t:    "a",
			want: &kptv1.Condition{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
		},
		"ConditionDoesNotExist": {
			cs: []kptv1.Condition{
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
				{Type: "c", Status: kptv1.ConditionFalse, Reason: "c", Message: "c"},
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
			},
			t:    "x",
			want: nil,
		},
		"ConditionEmptyList": {
			cs:   nil,
			t:    "x",
			want: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.cs != nil {
				kf.SetConditions(tc.cs...)
			}
			got := kf.GetCondition(tc.t)
			if got == nil || tc.want == nil {
				if got != tc.want {
					t.Errorf("TestGetCondition: -want%s, +got:\n%s", tc.want, got)
				}
			} else {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("TestGetCondition: -want, +got:\n%s", diff)
				}
			}

		})
	}
}

func TestSetConditions(t *testing.T) {

	f := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: xxx
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: xxx
`

	cases := map[string]struct {
		cs   []kptv1.Condition
		t    []kptv1.Condition
		want []kptv1.Condition
	}{
		"SetConditionsEmpty": {
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
		"SetConditionsNonEmpty": {
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
		"SetConditionsOverlap": {
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
		kf, err := New(f)
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}
		t.Run(name, func(t *testing.T) {
			if tc.cs != nil {
				kf.SetConditions(tc.cs...)
			}
			if tc.t != nil {
				kf.SetConditions(tc.t...)
			}
			gots := kf.GetConditions()
			if len(gots) != len(tc.want) {
				t.Errorf("TestSetConditions: got: %v, want: %v", gots, tc.want)
			} else {
				for idx, got := range gots {
					if diff := cmp.Diff(tc.want[idx], got); diff != "" {
						t.Errorf("TestSetCondition: -want, +got:\n%s", diff)
					}
				}
			}
		})
	}
}

func TestDeleteCondition(t *testing.T) {

	f := `apiVersion: kpt.dev/v1
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

	cases := map[string]struct {
		t    []string
		want []kptv1.Condition
	}{
		"DeleteConditionFirst": {
			t: []string{"a"},
			want: []kptv1.Condition{
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
			},
		},
		"DeleteConditionLast": {
			t: []string{"b"},
			want: []kptv1.Condition{
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
			},
		},
		"DeleteConditionAll": {
			t:    []string{"b", "a"},
			want: []kptv1.Condition{},
		},
		"DeleteConditionUnknown": {
			t: []string{"c"},
			want: []kptv1.Condition{
				{Type: "a", Status: kptv1.ConditionFalse, Reason: "a", Message: "a"},
				{Type: "b", Status: kptv1.ConditionFalse, Reason: "b", Message: "b"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			kf, err := New(f)
			if err != nil {
				t.Errorf("cannot unmarshal file: %s", err.Error())
			}
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
