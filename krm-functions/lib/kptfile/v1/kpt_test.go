package v1

import (
	"testing"

	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/google/go-cmp/cmp"
)

func TestGetCondition(t *testing.T) {

	f := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pkg-upf
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: upf package example
`

	kf := NewMutator(f)
	if _, err := kf.UnMarshal(); err != nil {
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
  name: pkg-upf
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: upf package example
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
		kf := NewMutator(f)
		if _, err := kf.UnMarshal(); err != nil {
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
					// no need to validate length as this was already done
					/*
						if idx > len(tc.want)-1 {
							t.Errorf("TestSetConditions: got: %v, want: %v", gots, tc.want)
						}
					*/
					if diff := cmp.Diff(tc.want[idx], got); diff != "" {
						t.Errorf("TestGetCondition: -want, +got:\n%s", diff)
					}
				}
			}
		})
	}
}
