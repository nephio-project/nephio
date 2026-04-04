/*
Copyright 2026 The Nephio Authors.

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

package cel

import (
	"testing"

	"github.com/kptdev/krm-functions-sdk/go/fn"
)

var kptfileWithCondition = `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pkg-test
  annotations:
    config.kubernetes.io/local-config: "true"
pipeline:
  mutators:
  - image: gcr.io/example/fn-a:latest
    condition: "true"
  - image: gcr.io/example/fn-b:latest
    condition: "false"
  - image: gcr.io/example/fn-c:latest
`

func makeRL(t *testing.T, items ...string) *fn.ResourceList {
	t.Helper()
	rl := &fn.ResourceList{Items: fn.KubeObjects{}}
	for _, raw := range items {
		ko, err := fn.ParseKubeObject([]byte(raw))
		if err != nil {
			t.Fatalf("ParseKubeObject: %v", err)
		}
		rl.Items = append(rl.Items, ko)
	}
	return rl
}

func TestEvaluateCondition(t *testing.T) {
	cases := map[string]struct {
		expr    string
		wantOk  bool
		wantErr bool
	}{
		"EmptyExpressionIsTrue": {expr: "", wantOk: true},
		"LiteralTrue":           {expr: "true", wantOk: true},
		"LiteralFalse":          {expr: "false", wantOk: false},
		"InvalidExpr":           {expr: "not_a_bool()", wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rl := makeRL(t)
			got, err := EvaluateCondition(tc.expr, rl)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantOk {
				t.Errorf("got %v, want %v", got, tc.wantOk)
			}
		})
	}
}

func TestEvaluateConditionForImage(t *testing.T) {
	cases := map[string]struct {
		image  string
		wantOk bool
	}{
		"ConditionTrue":    {image: "gcr.io/example/fn-a:latest", wantOk: true},
		"ConditionFalse":   {image: "gcr.io/example/fn-b:latest", wantOk: false},
		"NoCondition":      {image: "gcr.io/example/fn-c:latest", wantOk: true},
		"UnknownImage":     {image: "gcr.io/example/fn-unknown:latest", wantOk: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			kfko, err := fn.ParseKubeObject([]byte(kptfileWithCondition))
			if err != nil {
				t.Fatalf("ParseKubeObject: %v", err)
			}
			rl := &fn.ResourceList{Items: fn.KubeObjects{kfko}}

			got, err := EvaluateConditionForImage(rl, tc.image)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantOk {
				t.Errorf("image %q: got %v, want %v", tc.image, got, tc.wantOk)
			}
		})
	}
}

func TestEvaluateConditionForImage_NoKptfile(t *testing.T) {
	rl := makeRL(t)
	got, err := EvaluateConditionForImage(rl, "gcr.io/example/fn-a:latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Errorf("expected true when no Kptfile present")
	}
}
