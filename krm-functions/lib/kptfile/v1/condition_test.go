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

	corev1 "k8s.io/api/core/v1"
)

func TestGetConditionType(t *testing.T) {
	type object struct {
		apiVersion string
		kind       string
		name       string
		dummy      string
	}

	tests := map[string]struct {
		input object
		want  string
	}{
		"GetConditionType": {
			input: object{
				apiVersion: "a.a/a",
				kind:       "b",
				name:       "c",
			},
			want: "a.a/a.b.c",
		},
		"GetConditionTypeNoAPIVersion": {
			input: object{
				kind: "b",
				name: "c",
			},
			want: "b.c",
		},
		"GetConditionTypeWrongAPIVersion": {
			input: object{
				apiVersion: "a.a",
				kind:       "b",
				name:       "c",
			},
			want: "a.a.b.c",
		},
		"GetConditionTypeOnlyName": {
			input: object{
				name: "c",
			},
			want: "c",
		},
		"GetConditionTypeEmpty": {
			input: object{},
			want:  "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetConditionType(&corev1.ObjectReference{
				APIVersion: tt.input.apiVersion,
				Kind:       tt.input.kind,
				Name:       tt.input.name,
				Namespace:  tt.input.dummy,
			})
			if got != tt.want {
				t.Errorf("got %s want %s", got, tt.want)
			}
		})
	}
}
