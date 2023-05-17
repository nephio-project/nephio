/*
Copyright 2022 Nokia.

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

package resource

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAddAnnotations(t *testing.T) {
	key, value := "k", "v"
	existingKey, existingValue := "ek", "ev"

	type args struct {
		o           metav1.Object
		annotations map[string]string
	}

	cases := map[string]struct {
		args args
		want map[string]string
	}{
		"ExistingAnnotations": {
			args: args{
				o: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							existingKey: existingValue,
						},
					},
				},
				annotations: map[string]string{key: value},
			},
			want: map[string]string{
				existingKey: existingValue,
				key:         value,
			},
		},
		"NoExistingAnnotations": {
			args: args{
				o:           &corev1.Pod{},
				annotations: map[string]string{key: value},
			},
			want: map[string]string{key: value},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			AddAnnotations(tc.args.o, tc.args.annotations)

			got := tc.args.o.GetAnnotations()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("tc.args.o.GetAnnotations(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestRemoveAnnotations(t *testing.T) {
	keyA, valueA := "kA", "vA"
	keyB, valueB := "kB", "vB"

	type args struct {
		o           metav1.Object
		annotations []string
	}

	cases := map[string]struct {
		args args
		want map[string]string
	}{
		"ExistingAnnotations": {
			args: args{
				o: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							keyA: valueA,
							keyB: valueB,
						},
					},
				},
				annotations: []string{keyA},
			},
			want: map[string]string{keyB: valueB},
		},
		"NoExistingAnnotations": {
			args: args{
				o:           &corev1.Pod{},
				annotations: []string{keyA},
			},
			want: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			RemoveAnnotations(tc.args.o, tc.args.annotations...)

			got := tc.args.o.GetAnnotations()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("tc.args.o.GetAnnotations(...): -want, +got:\n%s", diff)
			}
		})
	}
}
