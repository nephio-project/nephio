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
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateGVKRef(t *testing.T) {
	cases := map[string]struct {
		input       corev1.ObjectReference
		errExpected bool
	}{
		"Normal": {
			input:       corev1.ObjectReference{APIVersion: "a", Kind: "b", Name: "c", Namespace: "d"},
			errExpected: false,
		},
		"APIVersionNotPresent": {
			input:       corev1.ObjectReference{Kind: "b", Name: "c", Namespace: "d"},
			errExpected: true,
		},
		"KindNotPresent": {
			input:       corev1.ObjectReference{APIVersion: "a", Name: "c", Namespace: "d"},
			errExpected: true,
		},
	}

	for _, tc := range cases {
		err := validateGVKRef(tc.input)

		if tc.errExpected {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestValidateGVKNRef(t *testing.T) {
	cases := map[string]struct {
		input       corev1.ObjectReference
		errExpected bool
	}{
		"Normal": {
			input:       corev1.ObjectReference{APIVersion: "a", Kind: "b", Name: "c", Namespace: "d"},
			errExpected: false,
		},
		"APIVersionNotPresent": {
			input:       corev1.ObjectReference{Kind: "b", Name: "c", Namespace: "d"},
			errExpected: true,
		},
		"KindNotPresent": {
			input:       corev1.ObjectReference{APIVersion: "a", Name: "c", Namespace: "d"},
			errExpected: true,
		},
		"NameNotPresent": {
			input:       corev1.ObjectReference{APIVersion: "a", Kind: "b", Namespace: "d"},
			errExpected: true,
		},
	}

	for _, tc := range cases {
		err := validateGVKNRef(tc.input)

		if tc.errExpected {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestGetGVKRefFromGVKNref(t *testing.T) {
	cases := map[string]struct {
		input *corev1.ObjectReference
	}{
		"Normal": {
			input: &corev1.ObjectReference{APIVersion: "a", Kind: "b", Name: "c", Namespace: "d"},
		},
	}

	for _, tc := range cases {
		got := getGVKRefFromGVKNref(tc.input)

		if diff := cmp.Diff(&corev1.ObjectReference{APIVersion: tc.input.APIVersion, Kind: tc.input.Kind}, got); diff != "" {
			t.Errorf("TestGetKubeObject: -want, +got:\n%s", diff)
		}
	}
}

func TestIsRefsValid(t *testing.T) {
	cases := map[string]struct {
		input   []corev1.ObjectReference
		isValid bool
	}{
		"NormalLen1": {
			input: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "b", Name: "c", Namespace: "d"},
			},
			isValid: true,
		},
		"NormalLen2": {
			input: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "b", Name: "c", Namespace: "d"},
				{APIVersion: "a", Kind: "b", Name: "c", Namespace: "d"},
			},
			isValid: true,
		},
		"Len0": {
			input:   []corev1.ObjectReference{},
			isValid: false,
		},
		"Len3": {
			input: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "b", Name: "c", Namespace: "d"},
				{APIVersion: "a", Kind: "b", Name: "c", Namespace: "d"},
				{APIVersion: "a", Kind: "b", Name: "c", Namespace: "d"},
				{APIVersion: "a", Kind: "b", Name: "c", Namespace: "d"},
			},
			isValid: false,
		},
		"NotInitializedLen2": {
			input: []corev1.ObjectReference{
				{APIVersion: "a", Kind: "b", Name: "c", Namespace: "d"},
				{APIVersion: "a", Name: "c", Namespace: "d"},
			},
			isValid: false,
		},
		"NotInitializedLen1": {
			input: []corev1.ObjectReference{
				{Kind: "b", Name: "c", Namespace: "d"},
			},
			isValid: false,
		},
	}

	for name, tc := range cases {
		ok := isRefsValid(tc.input)

		if tc.isValid != ok {
			t.Errorf("%s expected %t, got %t", name, tc.isValid, ok)
		}
	}
}

func TestIsGVKNNEqual(t *testing.T) {
	cases := map[string]struct {
		input1  any
		input2  any
		isEqual bool
	}{
		"Equal": {
			input1: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "a",
					Kind:       "b",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "c",
				},
			},
			input2: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "a",
					Kind:       "b",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "c",
				},
			},
			isEqual: true,
		},
		"NotEqual": {
			input1: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "a",
					Kind:       "b",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "c",
				},
			},
			input2: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "a",
					Kind:       "b",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "d",
				},
			},
			isEqual: false,
		},
	}

	for name, tc := range cases {
		o1, _ := fn.NewFromTypedObject(tc.input1)
		o2, _ := fn.NewFromTypedObject(tc.input2)
		ok := isGVKNNEqual(o1, o2)

		if tc.isEqual != ok {
			t.Errorf("%s expected %t, got %t", name, tc.isEqual, ok)
		}
	}
}
